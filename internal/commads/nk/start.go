package nk

import (
	"context"
	"fmt"
	"log"
	"net"
	"netokeep/pkg/protocol"
	"netokeep/pkg/traffic"
	"netokeep/pkg/transport"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func CreateStartCmd() *cobra.Command {
	var remoteAddr string
	var sid string

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the netokeep client.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// TODO: Setup SSH Listener

			traffic.StartClient(ctx, remoteAddr, func(conn net.Conn) {
				switch header := protocol.MatchHeader(conn); header {
				case protocol.SocksHeader:
					host, port := protocol.ParseSocHeader(conn)
					remoteConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
					if err != nil {
						log.Printf("Failed to connect to target %s:%d: %v", host, port, err)
						return
					}
					transport.Relay(conn, remoteConn)
				default:
					log.Fatal("Invalid request.")
					return
				}
			})
		},
	}

	startCmd.Flags().StringVarP(&remoteAddr, "remoteAddress", "r", "127.0.0.1:7222", "NKS server address")
	startCmd.Flags().StringVarP(&sid, "id", "n", "shun-client", "Session ID for identification")
	startCmd.MarkFlagRequired("remoteAddress")

	return startCmd
}
