package main

import (
	"context"
	"log"
	"netokeep/internal/logging"
	"netokeep/internal/services"
	"netokeep/internal/sessions"
	"netokeep/internal/states"
	"netokeep/internal/utils"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func createCoreCmd() *cobra.Command {
	var inPort uint16
	var outPort uint16

	var coreCmd = &cobra.Command{
		Use:    "core",
		Hidden: true,
		Short:  "Run the server in the foreground.",
		Long:   "Run the server in the foreground, console logs in the terminal.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			eg, egCtx := errgroup.WithContext(ctx)
			logging.InitLogging(states.ServerName)

			manager := &sessions.SessionManager{}

			// Start the ssh server
			sshReady := make(chan struct{})
			sshPort, err := utils.FindFreePort()
			if err != nil {
				log.Printf("Failed to find free port for SSH service: %v", err)
				return
			}
			eg.Go(func() error {
				return services.StartSSHService(egCtx, stop, sshReady, sshPort)
			})

			// Handle outgoing traffic
			proxyReady := make(chan struct{})
			if inPort == 0 {
				close(proxyReady)
			} else {
				eg.Go(func() error {
					return services.StartProxyListener(egCtx, stop, proxyReady, services.ProxyListenerConfig{
						Manager: manager,
						Port:    inPort,
					})
				})
			}

			// Handle incoming traffic
			serverReady := make(chan struct{})
			eg.Go(func() error {
				return services.StartTrafficServer(egCtx, stop, serverReady, services.TrafficServerConfig{
					Manager: manager,
					Port:    outPort,
					SSHPort: sshPort,
				})
			})

			remaining := 3
			for {
				select {
				case <-sshReady:
					sshReady = nil
					remaining--
				case <-proxyReady:
					proxyReady = nil
					remaining--
				case <-serverReady:
					serverReady = nil
					remaining--
				case <-egCtx.Done():
					finalState := states.StateStopped
					if err := eg.Wait(); err != nil {
						log.Printf("Error in server execution: %v", err)
						finalState = states.StateFailed
					}

					if err := states.WriteState(states.ServerName, &states.StateData{
						State:   finalState,
						Pid:     os.Getpid(),
						InPort:  inPort,
						OutPort: outPort,
					}); err != nil {
						log.Printf("Failed to write final state: %v", err)
					}
					log.Printf("NetoKeep server stopped.\n")
					return
				}

				if remaining == 0 {
					// All components are ready
					if err := states.WriteState(states.ServerName, &states.StateData{
						State:   states.StateRunning,
						Pid:     os.Getpid(),
						InPort:  inPort,
						OutPort: outPort,
					}); err != nil {
						log.Printf("Failed to write running state: %v", err)
					}
				}
			}
		},
	}

	coreCmd.Flags().Uint16VarP(&inPort, "in-port", "i", 0, "port to proxy incoming traffic (HTTP protocol).")
	coreCmd.Flags().Uint16VarP(&outPort, "out-port", "o", 7222, "port to forward outgoing traffic.")

	return coreCmd
}
