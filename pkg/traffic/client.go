package traffic

import (
	"context"
	"log"
	"net"
	"net/http"
	"netokeep/pkg/session"
	"netokeep/pkg/transport"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

func StartClient(ctx context.Context, manager *session.SessionManager, remoteAddr string, handler func(conn net.Conn)) {
	var wg sync.WaitGroup
	// Generate a unique session ID for this client instance
	sid := uuid.New().String()

	// Process the remote address to ensure it has the correct WebSocket scheme
	if strings.Contains(remoteAddr, "://") {
		remoteAddr = "ws://" + strings.Split(remoteAddr, "://")[1]
	}

	// Create ws connection
	header := http.Header{}
	header.Add("X-Session-ID", sid)
	wsConn, _, err := websocket.DefaultDialer.Dial(remoteAddr, header)
	if err != nil {
		log.Fatalf("Failed to reconnect to server: %v", err)
	}

	// Transfer to ws stream
	wstream := transport.NewWstream(wsConn)
	pConn := transport.NewPersistentConn(wstream)
	s, err := yamux.Client(pConn, nil)
	if err != nil {
		pConn.Close()
		log.Fatalf("Failed to create session: %v", err)
	}
	manager.NewSession(sid, pConn, s)

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
	log.Printf("✨ NetoKeep connects to server successfully!")

	// Handle the session connection and reconnection
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case pendingSid := <-manager.PendingActiveCh:
				if pendingSid != sid {
					continue
				}
				log.Printf("Received pending active signal, attempting to reconnect...")
				var success bool
				for range 5 {
					if ctx.Err() != nil {
						return
					}

					// Reconnecting
					wsConn, _, err := websocket.DefaultDialer.Dial(remoteAddr, header)
					if err != nil {
						log.Printf("Failed to reconnect to server: %v", err)
						time.Sleep(3 * time.Second)
						continue
					}
					wstream := transport.NewWstream(wsConn)
					manager.Reconnect(sid, wstream)
					log.Printf("Reconnected successfully!")
					success = true
					break
				}
				if !success {
					log.Printf("Failed to reconnect to server after 5 attempts.")
					return
				}
			}
		}
	}()

	<-ctx.Done()
	manager.Close()
	wg.Wait()
}
