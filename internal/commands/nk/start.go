package nk

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"netokeep/pkg/protocol"
	"netokeep/pkg/sessions"
	"netokeep/pkg/traffic"
	"netokeep/pkg/transport"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	xproxy "golang.org/x/net/proxy"

	"github.com/spf13/cobra"
)

func makeEgressDialer(proxyAddr, noProxy string) (func(network, addr string) (net.Conn, error), error) {
	direct := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if proxyAddr == "" {
		return direct.Dial, nil
	}

	proxyURL, err := url.Parse(proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid egress proxy URL: %w", err)
	}

	proxyDialer, err := xproxy.FromURL(proxyURL, direct)
	if err != nil {
		return nil, fmt.Errorf("build egress proxy dialer: %w", err)
	}

	if noProxy == "" {
		return proxyDialer.Dial, nil
	}

	perHost := xproxy.NewPerHost(proxyDialer, direct)
	perHost.AddFromString(noProxy)
	return perHost.Dial, nil
}

func CreateStartCmd() *cobra.Command {
	var remoteAddr string
	var sshPort uint16
	var forwardTraffic bool
	var egressProxy string
	var egressNoProxy string
	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the netokeep client.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			egressDial, err := makeEgressDialer(egressProxy, egressNoProxy)
			if err != nil {
				log.Fatalf("Failed to initialize egress dialer: %v", err)
			}

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
					target := net.JoinHostPort(host, strconv.Itoa(int(port)))
					log.Printf("Connection request to: %s", target)
					remoteConn, err := egressDial("tcp", target)
					if err != nil {
						log.Printf("Failed to connect to target %s: %v", target, err)
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
	startCmd.Flags().StringVar(&egressProxy, "egress-proxy", "", "Client-side proxy for forwarded outbound traffic, e.g. socks5://127.0.0.1:7891")
	startCmd.Flags().StringVar(&egressNoProxy, "egress-no-proxy", "localhost,127.0.0.1,::1", "Bypass list for the client-side egress proxy")
	startCmd.MarkFlagRequired("remoteAddress")

	return startCmd
}
