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
	var portIn uint16
	var portOut uint16

	var coreCmd = &cobra.Command{
		Use:    "core",
		Hidden: true,
		Short:  "Start the netokeep server in the foreground.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			eg, egCtx := errgroup.WithContext(ctx)
			logging.InitLogging("nks")

			// Start sshd service
			portSsh, stopSshd, err := services.StartSshdService()
			if err != nil {
				log.Fatalf("[nks] Failed to start sshd service: %v", err)
			}

			// Create a session manager to handle all user sessions
			manager := sessions.NewSessionManager()

			// Handle outgoing traffic
			eg.Go(func() error {
				return services.StartProxyListener(egCtx, manager, portIn)
			})

			// Handle incoming traffic
			eg.Go(func() error {
				return services.StartTrafficServer(egCtx, manager, portOut, portSsh)
			})

			// traffic.StartServer(ctx, manager, outPort, func(conn net.Conn) {
			// 	pattern, _, _, err := protocol.ParseSocHeader(conn)
			// 	if err != nil {
			// 		log.Printf("Failed to initialize the connection: %v", err)
			// 		return
			// 	}
			// 	switch pattern {
			// 	// The client will just actively send ssh request using channel
			// 	case protocol.SshPattern:
			// 		// For ssh request, the host and port in header are meaningless.
			// 		remoteConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", sshPort))
			// 		if err != nil {
			// 			log.Printf("Failed to connect to local ssh server: %v", err)
			// 			return
			// 		}
			// 		transport.Relay(conn, remoteConn)
			// 	default:
			// 		log.Printf("Invalid request.")
			// 		return
			// 	}
			// })

			<- egCtx.Done()
			stopSshd()
			manager.Close()

			if err := eg.Wait(); err != nil {
				log.Printf("[nks] Error in server execution: %v", err)
			}
			log.Printf("[nks] Netokeep server stopped.")
		},
	}

	coreCmd.Flags().Uint16VarP(&portIn, "in-port", "i", 7890, "port to proxy incoming traffic (HTTP protocol).")
	coreCmd.Flags().Uint16VarP(&portOut, "out-port", "o", 7222, "port to forward outgoing traffic.")

	return coreCmd
}
