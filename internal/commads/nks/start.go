package nks

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"netokeep/pkg/protocol"
	"netokeep/pkg/router"
	"netokeep/pkg/transport"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/yamux"
	"github.com/spf13/cobra"
)

func startProxy(ctx context.Context, port uint16) {
	if port == 0 {
		return
	}
	log.Printf("🌐 Container gateway started: %d", port)
	lc := net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		log.Printf("Proxy failed: %v", err)
	}

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		clientConn, _ := listener.Accept()

		// Find the first session to forward internet traffic
		go func(conn net.Conn) {
			defer conn.Close()
			protocol.Traffic2Session(clientConn)
		}(clientConn)
	}
}

func startService(ctx context.Context, sshPort uint16, outPort uint16) {
	// Handle HTTP connection
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		sid, ok := transport.IsWsRequest(w, r)
		if !ok {
			return
		}

		// Upgrade to the WebSocket protocol.
		wsConn, err := transport.Upgrade2Ws(w, r)
		if err != nil {
			log.Fatalf("Failed to upgrade HTTP to Ws: %v", err)
		}
		// Pack ws to stream
		wsStream := transport.NewWsStream(wsConn)

		// Check whether reconnection
		if protocol.Reconnect(sid, wsStream) {
			log.Printf("Recorded session. Reconnected.")
			return
		}

		// For new input
		pConn := transport.NewPersistentConn(wsStream)
		session, _ := yamux.Server(pConn, nil)

		if !protocol.NewSession(sid, pConn, session) {
			log.Printf("Failed to create session.")
			return
		}

		for {
			stream, err := session.Accept()
			if err != nil {
				break
			}
			go func(s net.Conn) {
				defer s.Close()
				router.HandleLogicStream(stream, sshPort)
			}(stream)
		}
		protocol.RemoveSession(sid)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", outPort),
		Handler: mux,
	}
	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	// Start the Server using http
	log.Printf("🚀 NetoKeep Server started, forwarding port: %d", outPort)
	server.ListenAndServe()
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

			// Start HTTP traffic proxy
			go startProxy(ctx, httPort)

			startService(ctx, sshPort, outPort)
		},
	}

	startCmd.Flags().Uint16VarP(&sshPort, "sshPort", "s", 22, "port to proxy traffic.")
	startCmd.Flags().Uint16VarP(&httPort, "httPort", "t", 1080, "port to proxy traffic.")
	startCmd.Flags().Uint16VarP(&outPort, "outPort", "o", 7222, "port to forward traffic.")

	return startCmd
}
