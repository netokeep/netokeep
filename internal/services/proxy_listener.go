package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"netokeep/internal/protocol"
	"netokeep/internal/sessions"
	"netokeep/internal/storage"
	"netokeep/internal/utils"
	"strconv"
	"strings"
	"time"

	xproxy "golang.org/x/net/proxy"
)

type ProxyListenerConfig struct {
	Manager *sessions.SessionManager
	Port    uint16
}

func StartProxyListener(ctx context.Context, stop context.CancelFunc, ready chan struct{}, conf ProxyListenerConfig) error {
	// Stop the whole program once the proxy listener exits.
	defer stop()

	// Create the proxy handler before setting up the listener.
	dial, err := buildRuleDialer(conf.Manager)
	if err != nil {
		return fmt.Errorf("error in building rule dialer: %v", err)
	}
	handler := createProxyHandler(dial)

	// Create TCP listener for the proxy server
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(int(conf.Port)))
	lc := net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("error in listening %d: %v", conf.Port, err)
	}

	// Construct the HTTP server
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	close(ready)
	log.Printf("[proxy] 🌐 HTTP proxy listener started at port %d", conf.Port)

	select {
	case err := <-serveErr:
		return fmt.Errorf("proxy server error: %v", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
		return nil
	}
}

var errHostDenied = errors.New("host denied by rules")

// rejectDialer refuses every connection; used for denied hosts.
type rejectDialer struct{}

func (rejectDialer) Dial(_, addr string) (net.Conn, error) {
	return nil, fmt.Errorf("%w: %s", errHostDenied, addr)
}

// tunnelDialer opens a yamux stream to a client and sends the target as a ConnProperty header.
type tunnelDialer struct{ manager *sessions.SessionManager }

func (d tunnelDialer) Dial(_, addr string) (net.Conn, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid port %q: %w", portStr, err)
	}
	return d.manager.OpenStream(protocol.ConnProperty{
		Pattern: protocol.ProxyPattern,
		Host:    host,
		Port:    uint16(port),
	}, true)
}

// buildRuleDialer applies deny → allow → default, matching hosts with xproxy rules.
func buildRuleDialer(manager *sessions.SessionManager) (xproxy.ContextDialer, error) {
	rules, err := storage.LoadServerConf()
	if err != nil {
		return nil, fmt.Errorf("error in loading server configuration: %w", err)
	}

	tunnel := tunnelDialer{manager: manager}

	// Hosts matching no list fall back to the default policy
	var fallback xproxy.Dialer = rejectDialer{}
	if rules.Default != "deny" {
		fallback = tunnel
	}

	// AllowList → tunnel, otherwise → fallback
	allow := xproxy.NewPerHost(fallback, tunnel)
	allow.AddFromString(strings.Join(rules.AllowList, ","))

	// DenyList → reject, otherwise → allow layer
	deny := xproxy.NewPerHost(allow, rejectDialer{})
	deny.AddFromString(strings.Join(rules.DenyList, ","))
	return deny, nil
}

func createProxyHandler(dialer xproxy.ContextDialer) http.Handler {
	rp := &httputil.ReverseProxy{
		// Forward-proxy requests already carry an absolute URL and Host; nothing to rewrite.
		Rewrite:   func(*httputil.ProxyRequest) {},
		Transport: &http.Transport{DialContext: dialer.DialContext, IdleConnTimeout: 90 * time.Second},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("[proxy] %s %s: %v", r.Method, r.URL.Host, err)
			if !errors.Is(err, errHostDenied) {
				w.WriteHeader(http.StatusBadGateway)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodConnect:
			remote, err := dialer.DialContext(r.Context(), "tcp", r.Host)
			if err != nil {
				log.Printf("[proxy] error connecting to %s: %v", r.Host, err)
				if !errors.Is(err, errHostDenied) {
					http.Error(w, err.Error(), http.StatusBadGateway)
				} else {
					http.Error(w, err.Error(), http.StatusForbidden)
				}
				return
			}
			defer remote.Close()

			conn, buf, err := http.NewResponseController(w).Hijack()
			if err != nil {
				log.Printf("[proxy] error hijacking connection: %v", err)
				return
			}
			defer conn.Close()
			// Close the hijacked conn on server shutdown; http.Server does not track it
			defer context.AfterFunc(r.Context(), func() { conn.Close() })()

			if _, err := conn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n")); err != nil {
				log.Printf("[proxy] error writing to hijacked connection: %v", err)
				return
			}
			// Bytes the client sent right after CONNECT may already sit in the server's buffer
			if n := buf.Reader.Buffered(); n > 0 {
				pending, _ := buf.Reader.Peek(n)
				if _, err := remote.Write(pending); err != nil {
					return
				}
			}
			utils.Relay(conn, remote)
		case r.URL.IsAbs():
			rp.ServeHTTP(w, r)
		default:
			// A request addressed to the proxy itself, not through it
			http.Error(w, "Bad request", http.StatusBadRequest)
		}
	})
}
