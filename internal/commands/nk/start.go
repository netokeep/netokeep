package nk

import (
	"context"
	"fmt"
	"log"
	"net"
	"netokeep/pkg/protocol"
	"netokeep/pkg/sessions"
	"netokeep/pkg/traffic"
	"netokeep/pkg/transport"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func CreateStartCmd() *cobra.Command {
	var remoteAddr string
	var sshPort uint16
	var forwardTraffic bool
	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the netokeep client.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// Create a session manager to handle all user sessions
			manager := sessions.NewSessionManager()

			// Handle SSH request
			go protocol.StartSshListener(ctx, sshPort, func(conn *protocol.SocConn) {
				header := conn.CreateSocHeader(protocol.SshPattern)
				// Select one accessible session to forward outgoing traffic
				manager.Traffic2Session(conn, header)
			})

			traffic.StartClient(ctx, manager, remoteAddr, forwardTraffic, func(conn net.Conn) {
				pattern, host, port, err := protocol.ParseSocHeader(conn)
				if err != nil {
					log.Printf("Failed to initialize the connection: %v", err)
					return
				}
				switch pattern {
				/// The server will just actively send tcp request using channel
				case protocol.ProPattern:
					log.Printf("Connection request to: %s", host)
					remoteConn, err := net.Dial("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
					if err != nil {
						log.Printf("Failed to connect to target %s:%d: %v", host, port, err)
						return
					}
					transport.Relay(conn, remoteConn)
				default:
					log.Printf("Invalid request.")
					return
				}
			})
		},
	}

	startCmd.Flags().BoolVarP(&forwardTraffic, "forwardTraffic", "f", false, "Forward SSH traffic")
	startCmd.Flags().StringVarP(&remoteAddr, "remoteAddress", "r", "", "NKS server address")
	startCmd.Flags().Uint16VarP(&sshPort, "sshPort", "s", 2222, "SSH port")
	startCmd.MarkFlagRequired("remoteAddress")

	return startCmd
}
