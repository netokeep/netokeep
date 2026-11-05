package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"netokeep/internal/protocol"
	"netokeep/internal/sessions"
	"netokeep/internal/transport"
	"netokeep/internal/utils"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

type TrafficServerConfig struct {
	Manager *sessions.SessionManager
	Port    uint16
	SSHPort uint16
}

func StartTrafficServer(
	ctx context.Context,
	_ context.CancelFunc,
	ready chan struct{},
	conf TrafficServerConfig,
) error {
	// Wait for every session to finish before returning
	var wg sync.WaitGroup
	defer wg.Wait()

	// Own ctx so sessions also stop when Serve fails; runs before wg.Wait (LIFO)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Must match the client: the receiver advertises the window
	yamuxConf := yamux.DefaultConfig()
	yamuxConf.EnableKeepAlive = false
	yamuxConf.ConnectionWriteTimeout = 10 * time.Second
	yamuxConf.MaxStreamWindowSize = 2 * 1024 * 1024 // 2MB

	upgrader := websocket.Upgrader{
		ReadBufferSize:  256 * 1024,
		WriteBufferSize: 256 * 1024,
		CheckOrigin:     func(*http.Request) bool { return true },
	}

	// serveSession runs one client session until it closes or ctx is canceled.
	serveSession := func(id string, ws *websocket.Conn, forwardTraffic bool) {
		rc := transport.NewResumableConn(ctx, ws, nil)
		session, err := yamux.Server(rc, yamuxConf)
		if err != nil {
			rc.Close()
			log.Printf("[traffic-server] failed to create session %s: %v", id, err)
			return
		}
		defer session.Close()
		defer context.AfterFunc(ctx, func() { session.Close() })()

		if err := conf.Manager.AddSession(id, &sessions.UserSession{
			Session:        session,
			Rc:             rc,
			ForwardTraffic: forwardTraffic,
		}); err != nil {
			log.Printf("[traffic-server] %v", err)
			return
		}
		defer conf.Manager.RemoveSession(id)
		log.Printf("[traffic-server] session %s started (forward traffic: %v)", id, forwardTraffic)

		var streams sync.WaitGroup
		defer streams.Wait()
		for {
			stream, err := session.Accept()
			if err != nil {
				if cause := rc.Err(); cause != nil {
					err = cause
				}
				log.Printf("[traffic-server] session %s closed: %v", id, err)
				return
			}
			streams.Go(func() { handleServerStream(ctx, stream, conf.SSHPort) })
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("NK-Session-ID")
		if !websocket.IsWebSocketUpgrade(r) || id == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		resume := r.Header.Get("NK-Resume") == "true"
		forwardTraffic := r.Header.Get("NK-Forward-Traffic") == "true"

		// Reject before upgrading: once upgraded, no HTTP status can be sent
		us, exists := conf.Manager.Get(id)
		switch {
		case resume && !exists:
			http.Error(w, "session expired", http.StatusGone)
			return
		case !resume && exists:
			http.Error(w, "session id in use", http.StatusConflict)
			return
		case resume && us.ForwardTraffic != forwardTraffic:
			http.Error(w, "forward traffic option cannot change on resume", http.StatusConflict)
			return
		}

		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			// Upgrade has already written the error response
			log.Printf("[traffic-server] failed to upgrade %s: %v", r.RemoteAddr, err)
			return
		}

		if resume {
			if err := us.Rc.UpdateWsConn(ws); err != nil {
				log.Printf("[traffic-server] failed to resume session %s: %v", id, err)
				return
			}
			log.Printf("[traffic-server] session %s resumed from %s", id, r.RemoteAddr)
			return
		}
		// Hijacked conns are not tracked by http.Server, so track sessions ourselves
		wg.Go(func() { serveSession(id, ws, forwardTraffic) })
	}

	addr := net.JoinHostPort("0.0.0.0", strconv.Itoa(int(conf.Port)))
	lc := net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	defer ln.Close()

	srv := &http.Server{
		Handler:           http.HandlerFunc(handler),
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	close(ready)
	log.Printf("[traffic-server] 🚀 NetoKeep server listening on port %d", conf.Port)

	select {
	case err := <-serveErr:
		return fmt.Errorf("traffic server stopped: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdown()
		srv.Shutdown(shutdownCtx)
		return nil
	}
}

// handleServerStream reads the property header and serves the stream accordingly.
func handleServerStream(ctx context.Context, stream net.Conn, sshPort uint16) {
	defer stream.Close()

	// A peer that never sends the header must not hold the stream forever
	stream.SetReadDeadline(time.Now().Add(10 * time.Second))
	property, err := protocol.ResolveConnProperty(stream)
	if err != nil {
		log.Printf("[traffic-server] failed to resolve connection property: %v", err)
		return
	}
	stream.SetReadDeadline(time.Time{})

	switch property.Pattern {
	case protocol.SSHPattern:
		d := net.Dialer{Timeout: 3 * time.Second}
		local, err := d.DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(int(sshPort))))
		if err != nil {
			log.Printf("[traffic-server] failed to connect to local sshd: %v", err)
			return
		}
		defer local.Close()
		utils.Relay(stream, local)
	default:
		log.Printf("[traffic-server] unhandled connection pattern: %v", property.Pattern)
	}
}
