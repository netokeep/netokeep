package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"netokeep/internal/protocol"
	"netokeep/internal/sessions"
	"netokeep/pkg/transport"
	"netokeep/pkg/utils"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
)

func StartTrafficServer(ctx context.Context, _ context.CancelFunc, manager *sessions.SessionManager, portOut uint16, portSsh uint16) error {
	var wg sync.WaitGroup
	// Setup yamux config
	cfg := yamux.DefaultConfig()
	// cfg.LogOutput = io.Discard
	// cfg.EnableKeepAlive = false
	cfg.MaxStreamWindowSize = 4 * 1024 * 1024 // 4MB

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		sid, ipAddr, forwardTraffic, ok := transport.IsWsRequest(w, r)
		if !ok {
			log.Printf("[traffic] Invalid request from: %s", ipAddr)
			return
		}
		wsConn, err := transport.Upgrade2Ws(w, r)
		if err != nil {
			log.Printf("[traffic] Failed to upgrade HTTP to Ws: %v", err)
			return
		}
		log.Printf("[traffic] New connection received from: %s", ipAddr)
		if ok := manager.HasSession(sid); ok {
			log.Printf("[traffic] Session already exists, updating ws connection.")
			manager.UpdateSession(sid, wsConn, forwardTraffic)
			return
		}
		arwstream := transport.NewARWStream(ctx, wsConn, nil)
		session, err := yamux.Server(arwstream, cfg)
		if err != nil {
			arwstream.Close()
			log.Printf("[traffic] Failed to create session: %v", err)
			return
		}
		manager.NewSession(sid, session, arwstream, forwardTraffic)

		go func() {
			defer manager.RemoveSession(sid)

			for {
				conn, err := session.Accept()
				if err != nil {
					log.Printf("[traffic] Session [%s] closed: %v", sid, err)
					return
				}

				wg.Go(func() {
					defer conn.Close()
					pattern, host, _, err := protocol.ParseSocHeader(conn)
					if err != nil {
						log.Printf("[traffic] Failed to parse header form %s: %v", host, err)
						return
					}
					switch pattern {
					// The client will just actively send ssh request using channel
					case protocol.SshPattern:
						remoteConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", portSsh))
						if err != nil {
							log.Printf("[traffic] Failed to connect to local ssh server: %v", err)
							return
						}
						utils.Relay(conn, remoteConn)
					default:
						log.Printf("[traffic] Invalid request from %s.", host)
						return
					}
				})
			}
		}()
	})

	// Start the HTTP server to listen for incoming WebSocket connections.
	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", portOut),
		Handler: mux,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[traffic] NetoKeep service stopped with unexpected error: %v", err)
			return
		}
	}()
	log.Printf("🚀 NetoKeep Server started, forwarding port: %d", portOut)

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
	wg.Wait()
	return nil
}
