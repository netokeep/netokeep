package nks

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
	var sshPort uint16
	var httPort uint16
	var outPort uint16

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the netokeep server.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// Create a session manager to handle all user sessions
			manager := session.NewSessionManager()

			// Handle outgoing traffic
			go traffic.StartSocksListener(ctx, httPort, func(conn *protocol.SocksConn) {
				header := protocol.CreateSocksHeader(conn)
				// Select one accessible session to forward outgoing traffic
				manager.Traffic2Session(conn, header)
			})

			traffic.StartServer(ctx, manager, sshPort, outPort, func(conn net.Conn) {
				switch pattern := protocol.MatchHeader(conn); pattern {
				// The client will just actively send ssh request using channel
				case protocol.SshHeader:
					remoteConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", sshPort))
					if err != nil {
						log.Printf("Failed to connect to local ssh server: %v", err)
						return
					}
					transport.Relay(conn, remoteConn)
				default:
					println("asdfasdf")
					return
				}
			})
		},
	}

	startCmd.Flags().Uint16VarP(&sshPort, "sshPort", "s", 22, "port to proxy traffic.")
	startCmd.Flags().Uint16VarP(&httPort, "httPort", "t", 1080, "port to proxy traffic.")
	startCmd.Flags().Uint16VarP(&outPort, "outPort", "o", 7222, "port to forward traffic.")

	return startCmd
}
