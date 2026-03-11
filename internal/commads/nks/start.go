package nks

import (
	"context"
	"net"
	"netokeep/pkg/protocol"
	"netokeep/pkg/session"
	"netokeep/pkg/traffic"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func startService(ctx context.Context, sshPort uint16, httPort uint16, outPort uint16) {
	// Create a session manager to handle all user sessions
	manager := session.NewSessionManager()

	// Handle outgoing traffic
	go traffic.StartSocksListener(ctx, httPort, func(conn *protocol.SocksConn) {
		header := protocol.CreateSocksHeader(conn)
		// Select one accessible session to forward outgoing traffic
		manager.Traffic2Session(conn, header)
	})

	traffic.StartServer(ctx, manager, sshPort, outPort, func(conn net.Conn) {
		print("Find connection.")
	})
}

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

			startService(ctx, sshPort, httPort, outPort)
		},
	}

	startCmd.Flags().Uint16VarP(&sshPort, "sshPort", "s", 22, "port to proxy traffic.")
	startCmd.Flags().Uint16VarP(&httPort, "httPort", "t", 1080, "port to proxy traffic.")
	startCmd.Flags().Uint16VarP(&outPort, "outPort", "o", 7222, "port to forward traffic.")

	return startCmd
}
