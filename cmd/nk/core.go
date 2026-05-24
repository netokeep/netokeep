package main

import (
	"context"
	"log"
	"netokeep/internal/logging"
	"netokeep/internal/services"
	"netokeep/internal/sessions"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func createCoreCmd() *cobra.Command {
	var portSsh uint16
	var remoteAddr string
	var forwardTraffic bool
	var useProxy bool

	var coreCmd = &cobra.Command{
		Use:    "core [name]",
		Hidden: true,
		Short:  "Start the netokeep client in the foreground.",
		Args:   cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := "default"
			if len(args) > 0 {
				name = args[0]
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			eg, egCtx := errgroup.WithContext(ctx)
			logging.InitLogging(name)

			// Create a session manager to handle all user sessions
			manager := sessions.NewSessionManager()

			// Handle ssh traffic
			eg.Go(func() error {
				return services.StartSshdListener(egCtx, manager, portSsh)
			})

			// Handle incoming traffic
			eg.Go(func() error {
				return services.StartTrafficClient(egCtx, manager, remoteAddr, forwardTraffic, useProxy)
			})

			<-egCtx.Done()
			manager.Close()

			if err := eg.Wait(); err != nil {
				log.Printf("[nks] Error in client execution: %v", err)
			}
			log.Printf("[nks] Netokeep client stopped.")
		},
	}

	coreCmd.Flags().StringVarP(&remoteAddr, "remote-address", "r", "", "NKS server address")
	coreCmd.Flags().Uint16VarP(&portSsh, "ssh-port", "s", 2222, "SSH port")
	coreCmd.Flags().BoolVarP(&forwardTraffic, "forward-traffic", "f", false, "forward SSH traffic")
	coreCmd.Flags().BoolVarP(&useProxy, "use-proxy", "p", false, "use proxy for outgoing traffic")
	coreCmd.MarkFlagRequired("remote-address")

	return coreCmd
}
