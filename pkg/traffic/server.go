package traffic

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"netokeep/pkg/sessions"
	"netokeep/pkg/transport"

	"github.com/hashicorp/yamux"
)

func StartServer(ctx context.Context, manager *sessions.SessionManager, sshPort uint16, outPort uint16, handler func(conn net.Conn)) {
	var wg sync.WaitGroup
	// Setup yamux config
	cfg := yamux.DefaultConfig()
	// cfg.LogOutput = io.Discard
	cfg.EnableKeepAlive = false
	cfg.MaxStreamWindowSize = 4 * 1024 * 1024 // 4MB

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		sid, ipAddr, forwardTraffic, ok := transport.IsWsRequest(w, r)
		if !ok {
			log.Printf("Invalid request from: %s", ipAddr)
			return
		}
		wsConn, err := transport.Upgrade2Ws(w, r)
		if err != nil {
			log.Printf("Failed to upgrade HTTP to Ws: %v", err)
			return
		}
		log.Printf("✨ New connection received from: %s", ipAddr)
		if ok := manager.HasSession(sid); ok {
			log.Printf("Session already exists, updating ws connection.")
			manager.UpdateSession(sid, wsConn, forwardTraffic)
			return
		}
		arwstream := transport.NewARWStream(ctx, wsConn, nil)
		session, err := yamux.Server(arwstream, cfg)
		if err != nil {
			arwstream.Close()
			log.Printf("Failed to create session: %v", err)
			return
		}
		manager.NewSession(sid, session, arwstream, forwardTraffic)

		go func() {
			defer manager.RemoveSession(sid)

			for {
				conn, err := session.Accept()
				if err != nil {
					log.Printf("Session [%s] closed: %v", sid, err)
					return
				}

				wg.Go(func() {
					defer conn.Close()
					handler(conn)
				})
			}
		}()
	})

	// Start the HTTP server to listen for incoming WebSocket connections.
	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", outPort),
		Handler: mux,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("NetoKeep service stopped with unexpected error: %v", err)
			return
		}
	}()
	log.Printf("🚀 NetoKeep Server started, forwarding port: %d", outPort)

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
	manager.Close()
	wg.Wait()
}
