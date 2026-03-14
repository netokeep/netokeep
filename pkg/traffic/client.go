package traffic

import (
	"context"
	"log"
	"net"
	"net/http"
	"netokeep/pkg/sessions"
	"netokeep/pkg/transport"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

func StartClient(ctx context.Context, manager *sessions.SessionManager, remoteAddr string, handler func(conn net.Conn)) {
	var wg sync.WaitGroup
	// Generate a unique session ID for this client instance
	sid := uuid.New().String()

	// Process the remote address to ensure it has the correct WebSocket scheme
	if strings.Contains(remoteAddr, "://") {
		remoteAddr = "ws://" + strings.Split(remoteAddr, "://")[1]
	}

	// Create ws connection
	dailer := func() (*websocket.Conn, error) {
		header := http.Header{}
		header.Add("X-Session-ID", sid)
		wsConn, _, err := websocket.DefaultDialer.Dial(remoteAddr, header)
		return wsConn, err
	}
	wsConn, err := dailer()
	if err != nil {
		log.Fatalf("Failed to reconnect to server: %v", err)
	}
	log.Printf("✨ NetoKeep connects to server successfully!")

	// Create a yamux client session to store the ws connection
	arwstream := transport.NewARWStream(wsConn, dailer)
	session, err := yamux.Client(arwstream, nil)
	if err != nil {
		arwstream.Close()
		log.Fatalf("Failed to create session: %v", err)
	}
	manager.NewSession(sid, session, arwstream)

	go func() {
		defer arwstream.Close()
		defer session.Close()

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
	log.Printf("✨ NetoKeep connects to server successfully!")

	<-ctx.Done()
	manager.Close()
	wg.Wait()
}
