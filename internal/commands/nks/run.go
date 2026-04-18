package nks

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"netokeep/pkg/protocol"
	"netokeep/pkg/sessions"
	"netokeep/pkg/traffic"
	"netokeep/pkg/transport"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/natefinch/lumberjack"
	"github.com/spf13/cobra"
)

func CreateRunCmd() *cobra.Command {
	var sshPort uint16
	var tcpPort uint16
	var outPort uint16

	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the netokeep server.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// Create logger with lumberjack
			runDir, err := getXDGDir()
			if err != nil {
				log.Fatalf("Failed to get XDG directory: %v", err)
			}
			logPath := filepath.Join(runDir, "netokeep.log")
			lumberjackLogger := &lumberjack.Logger{
				Filename:   logPath,
				MaxSize:    1,      // Size of one log file (MB)
				MaxBackups: 1,       // Max number of old log files to keep
				MaxAge:     28,      // Max age of old log files (days)
				Compress:   true,    // Compress old log files
			}
			multiWriter := io.MultiWriter(os.Stdout, lumberjackLogger)
			log.SetOutput(multiWriter)

			// Create a session manager to handle all user sessions
			manager := sessions.NewSessionManager()

			// Handle outgoing traffic
			go protocol.StartProxyListener(ctx, tcpPort, func(conn *protocol.SocConn) {
				header := conn.CreateSocHeader(protocol.ProPattern)
				// Select one accessible session to forward outgoing traffic
				manager.Traffic2Session(conn, header)
			})

			traffic.StartServer(ctx, manager, sshPort, outPort, func(conn net.Conn) {
				pattern, _, _, err := protocol.ParseSocHeader(conn)
				if err != nil {
					log.Printf("Failed to initialize the connection: %v", err)
					return
				}
				switch pattern {
				// The client will just actively send ssh request using channel
				case protocol.SshPattern:
					// For ssh request, the host and port in header are meaningless.
					remoteConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", sshPort))
					if err != nil {
						log.Printf("Failed to connect to local ssh server: %v", err)
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

	runCmd.Flags().Uint16VarP(&sshPort, "sshPort", "s", 22, "port to proxy SSH traffic.")
	runCmd.Flags().Uint16VarP(&tcpPort, "tcpPort", "t", 7890, "port to proxy TCP traffic.")
	runCmd.Flags().Uint16VarP(&outPort, "outPort", "o", 7222, "port to forward traffic.")

	return runCmd
}
