package traffic

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"netokeep/pkg/session"
	"netokeep/pkg/transport"

	"github.com/hashicorp/yamux"
)

func StartServer(ctx context.Context, manager *session.SessionManager, sshPort uint16, outPort uint16, handler func(conn net.Conn)) {
	var wg sync.WaitGroup
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		sid, ok := transport.IsWsRequest(w, r)
		if !ok {
			return
		}
		wsConn, err := transport.Upgrade2Ws(w, r)
		if err != nil {
			log.Printf("Failed to upgrade HTTP to Ws: %v", err)
			return
		}
		wstream := transport.NewWstream(wsConn)

		// Check whether reconnection
		if manager.Reconnect(sid, wstream) {
			log.Printf("Recorded session. Reconnected.")
			return
		}

		// For new input, create a new session and bind it with the ws connection.
		pConn := transport.NewPersistentConn(wstream)
		s, _ := yamux.Server(pConn, nil)

		if !manager.NewSession(sid, pConn, s) {
			log.Printf("Failed to create session.")
		}

		go func() {
			defer pConn.Close()
			defer s.Close()

			for {
				conn, err := s.Accept()
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

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", outPort),
		Handler: mux,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Printf("Failed to start server: %v", err)
		}
	}()
	log.Printf("🚀 NetoKeep Server started, forwarding port: %d", outPort)

	<-ctx.Done()
	server.Shutdown(context.Background())
	wg.Wait()
}
