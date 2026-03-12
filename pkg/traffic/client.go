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

	s, pConn, err := createSession(ctx, sid, remoteAddr)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}
	manager.NewSession(sid, pConn, s)
	go func() {
		defer pConn.Close()
		defer s.Close()

		for {
			conn, err := s.Accept()
			if err != nil {
				log.Fatalf("Session [%s] closed: %v", sid, err)
			}
			wg.Go(func() {
				defer conn.Close()
				handler(conn)
			})
		}
	}()
	log.Printf("🚀 NetoKeep tunnel is ready! [ID: %s]", sid)

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
				log.Println("Received pending active signal, attempting to reconnect...")
				for i := 0; i < 5; i++ {
					if ctx.Err() != nil {
						return
					}
					wsStream, err := dialRaw(ctx, sid, remoteAddr)
					if err != nil {
						time.Sleep(3 * time.Second)
						continue
					}
					manager.Reconnect(sid, wsStream)
					log.Printf("Reconnected successfully! [ID: %s]", sid)
					break
				}
				log.Fatalf("Failed to reconnect to server after 5 attempts.")
			}
		}
	}()

	<-ctx.Done()
	wg.Wait()
}

// dialRaw establishes a new raw WebSocket connection and returns it as a net.Conn.
// Used for reconnection — does NOT create a new PersistentConn or yamux session.
func dialRaw(ctx context.Context, sid string, remoteAddr string) (net.Conn, error) {
	header := http.Header{}
	header.Add("X-Session-ID", sid)

	wsConn, _, err := websocket.DefaultDialer.Dial(remoteAddr, header)
	if err != nil {
		log.Printf("Failed to reconnect to server: %v", err)
		return nil, err
	}
	return transport.NewWsStream(wsConn), nil
}

func createSession(ctx context.Context, sid string, remoteAddr string) (s *yamux.Session, pConn *transport.PersistentConn, err error) {
	wsStream, err := dialRaw(ctx, sid, remoteAddr)
	if err != nil {
		return nil, nil, err
	}
	pConn = transport.NewPersistentConn(wsStream)
	s, _ = yamux.Client(pConn, nil)
	return s, pConn, nil
}
