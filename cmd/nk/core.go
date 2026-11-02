package main

import (
	"context"
	"log"
	"netokeep/internal/logging"
	"netokeep/internal/services"
	"netokeep/internal/sessions"
	"netokeep/internal/states"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func createCoreCmd() *cobra.Command {
	var remoteAddr string
	var sshPort uint16
	var forwardTraffic bool
	var useProxy bool

	var coreCmd = &cobra.Command{
		Use:    "core <nickname>",
		Hidden: true,
		Short:  "Run the client in the foreground",
		Long:   "Run the client in the foreground, console logs in the terminal.",
		Args:   cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			eg, egCtx := errgroup.WithContext(ctx)
			logging.InitLogging(name)

			manager := &sessions.SessionManager{}

			// Start the ssh server in a separate goroutine
			sshReady := make(chan struct{})
			eg.Go(func() error {
				return services.StartSSHForwardListener(egCtx, stop, sshReady, services.SSHForwardConf{
					Manager: manager,
					SSHPort: sshPort,
				})
			})

			// Start the client in a separate goroutine
			clientReady := make(chan struct{})
			eg.Go(func() error {
				return services.StartTrafficClient(egCtx, stop, clientReady, services.TrafficClientConf{
					Manager:       manager,
					RemoteAddr:    remoteAddr,
					ForwardTraffic: forwardTraffic,
					UseProxy:      useProxy,
				})
			})

			// Wait for both ready signals (union of sshReady and clientReady) while
			// still watching egCtx.Done(), so a failure in either goroutine (which
			// cancels egCtx via errgroup) is never missed.
			remaining := 2
			for {
				select {
				case <-sshReady:
					sshReady = nil
					remaining--
				case <-clientReady:
					clientReady = nil
					remaining--
				case <-egCtx.Done():
					// Context canceled, exit the loop
					manager.Close()

					finalState := states.StateStopped
					if err := eg.Wait(); err != nil {
						log.Printf("Error in client execution: %v", err)
						finalState = states.StateFailed
					}

					if err := states.WriteState(name, &states.StateData{
						State: finalState,
						Pid:   os.Getpid(),
					}); err != nil {
						log.Printf("Failed to write stopped state: %v\n", err)
					}
					log.Printf("NetoKeep client stopped.")
					return
				}

				if remaining == 0 {
					// All components are ready, proceed with the rest of the logic
					if err := states.WriteState(name, &states.StateData{
						State:          states.StateRunning,
						Pid:            os.Getpid(),
						RemoteAddr:     remoteAddr,
						SSHPort:        sshPort,
						ForwardTraffic: forwardTraffic,
						UseProxy:       useProxy,
					}); err != nil {
						log.Printf("Failed to write running state: %v\n", err)
					}
				}
			}
		},
	}

	coreCmd.Flags().Uint16VarP(&sshPort, "ssh-port", "s", 0, "ssh port to use")
	coreCmd.Flags().StringVarP(&remoteAddr, "remote-addr", "r", "", "remote address to connect to")
	coreCmd.Flags().BoolVarP(&forwardTraffic, "forward-traffic", "f", false, "enable traffic forwarding (default false)")
	coreCmd.Flags().BoolVarP(&useProxy, "use-proxy", "p", false, "use proxy for connection (default false)")
	coreCmd.MarkFlagRequired("remote-addr")
	coreCmd.MarkFlagRequired("ssh-port")

	return coreCmd
}
