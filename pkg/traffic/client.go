package traffic

import (
	"context"
	"log"
	"net"
	"net/http"
	"netokeep/pkg/sessions"
	"netokeep/pkg/transport"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

func StartClient(
	ctx context.Context,
	manager *sessions.SessionManager,
	remoteAddr string,
	forwardTraffic bool,
	handler func(conn net.Conn),
) {
	var wg sync.WaitGroup
	// Generate a unique session ID for this client instance
	sid := uuid.New().String()
	// Setup yamux config
	cfg := yamux.DefaultConfig()
	// cfg.LogOutput = io.Discard
	cfg.EnableKeepAlive = false
	cfg.MaxStreamWindowSize = 4 * 1024 * 1024 // 4MB

	// Process the remote address to ensure it has the correct WebSocket scheme
	if strings.Contains(remoteAddr, "://") {
		remoteAddr = "ws://" + strings.Split(remoteAddr, "://")[1]
	}

	// Create ws connection
	dailer := func() (*websocket.Conn, error) {
		header := http.Header{}
		header.Add("X-Session-ID", sid)
		header.Add("X-Forward-Traffic", strconv.FormatBool(forwardTraffic))

		wsConn, _, err := websocket.DefaultDialer.Dial(remoteAddr, header)
		return wsConn, err
	}
	wsConn, err := dailer()
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	// Create a yamux client session to store the ws connection
	arwstream := transport.NewARWStream(ctx, wsConn, dailer)
	session, err := yamux.Client(arwstream, cfg)
	if err != nil {
		arwstream.Close()
		log.Fatalf("Failed to create session: %v", err)
	}
	// Send the server to control the traffic forwarding.
	// For client side, forward ssh traffic by default.
	manager.NewSession(sid, session, arwstream, true)

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
	log.Printf("✨ NetoKeep connects to server successfully!")

	<-ctx.Done()
	manager.Close()
	wg.Wait()
}
