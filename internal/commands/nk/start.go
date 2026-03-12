package nk

import (
	"context"
	"fmt"
	"log"
	"net"
	"netokeep/pkg/protocol"
	"netokeep/pkg/session"
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

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the netokeep client.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// Create a session manager to handle all user sessions
			manager := session.NewSessionManager()

			// TODO: Setup SSH Listener
			go traffic.StartSshListener(ctx, sshPort, func(conn net.Conn) {
				header := protocol.CreateSshHeader(conn)
				// Select one accessible session to forward outgoing traffic
				manager.Traffic2Session(conn, header)
			})

			traffic.StartClient(ctx, manager, remoteAddr, func(conn net.Conn) {
				switch header := protocol.MatchHeader(conn); header {
				/// The server will just actively send tcp request using channel
				case protocol.SocksHeader:
					host, port, err := protocol.ParseSocHeader(conn)
					if err != nil {
						log.Printf("Error in parse the header of soc: %v", err)
						return
					}
					remoteConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
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

	startCmd.Flags().StringVarP(&remoteAddr, "remoteAddress", "r", "", "NKS server address")
	startCmd.Flags().Uint16VarP(&sshPort, "sshPort", "s", 2222, "SSH port")
	startCmd.MarkFlagRequired("remoteAddress")

	return startCmd
}
