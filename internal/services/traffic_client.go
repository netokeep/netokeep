package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"netokeep/internal/protocol"
	"netokeep/internal/sessions"
	"netokeep/internal/storage"
	"netokeep/internal/transport"
	"netokeep/internal/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
	xproxy "golang.org/x/net/proxy"
)

type TrafficClientConf struct {
	Manager        *sessions.SessionManager
	RemoteAddr     string
	ForwardTraffic bool
	UseProxy       bool
}

const maxRetries = 5

func StartTrafficClient(
	ctx context.Context,
	stop context.CancelFunc,
	ready chan struct{},
	conf TrafficClientConf,
) error {
	// Stop the whole program once the traffic client exits
	defer stop()

	// Parameter validation
	if !strings.Contains(conf.RemoteAddr, "://") {
		return fmt.Errorf("invalid remote address: %s", conf.RemoteAddr)
	}
	conf.RemoteAddr = "ws://" + strings.Split(conf.RemoteAddr, "://")[1]

	// Setup yamux config()
	yamuxConf := yamux.DefaultConfig()
	yamuxConf.EnableKeepAlive = false
	yamuxConf.ConnectionWriteTimeout = 10 * time.Second
	yamuxConf.MaxStreamWindowSize = 2 * 1024 * 1024 // 2MB

	// Load client matcher for selecting the traffic to be proxied
	proxyDialer, err := func() (func(ctx context.Context, network, target string) (net.Conn, error), error) {
		defaultDialer := &net.Dialer{Timeout: 3 * time.Second}
		if !conf.UseProxy {
			return defaultDialer.DialContext, nil
		}
		proxyConf, err := storage.LoadClientConf()
		if err != nil {
			return nil, fmt.Errorf("failed to load client proxy configuration: %w", err)
		}
		proxyURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", proxyConf.Type, proxyConf.Addr, proxyConf.Port))
		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
		}
		proxyDialer, err := xproxy.FromURL(proxyURL, defaultDialer)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy dialer: %w", err)
		}
		dialer := xproxy.NewPerHost(defaultDialer, proxyDialer)
		dialer.AddFromString(strings.Join(proxyConf.AllowList, ","))
		return dialer.DialContext, nil
	}()
	if err != nil {
		return fmt.Errorf("failed to create proxy dialer: %w", err)
	}

	// Create ws connection
	wsDialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	dial := func(id string, resume bool) (*websocket.Conn, error) {
		header := http.Header{}
		header.Set("NK-Session-ID", id)
		header.Set("NK-Forward-Traffic", strconv.FormatBool(conf.ForwardTraffic))
		header.Set("NK-Resume", strconv.FormatBool(resume))
		conn, resp, err := wsDialer.DialContext(ctx, conf.RemoteAddr, header)
		if resp != nil && resp.StatusCode == http.StatusGone {
			return nil, transport.ErrSessionGone
		}
		return conn, err
	}

	runSession := func(successHandler func()) error {
		// Initialize the wait group for managing concurrent operations
		var wg sync.WaitGroup
		defer wg.Wait()

		// Every run, generate a new client ID for the session
		clientId := rand.Text()
		dialer := func(resume bool) (*websocket.Conn, error) { return dial(clientId, resume) }

		wc, err := dialer(false)
		if err != nil {
			return fmt.Errorf("failed to create WebSocket connection: %w", err)
		}
		rc := transport.NewResumableConn(ctx, wc, dialer)
		session, err := yamux.Client(rc, yamuxConf)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create yamux session: %w", err)
		}
		defer session.Close()
		defer context.AfterFunc(ctx, func() { session.Close() })()

		if err := conf.Manager.AddSession(clientId, &sessions.UserSession{
			Session: session,
			Rc:      rc,
			// Client does not receive and forward traffic by default
			ForwardTraffic: false,
		}); err != nil {
			return fmt.Errorf("failed to add session: %w", err)
		}
		defer conf.Manager.RemoveSession(clientId)

		successHandler()
		log.Printf("[traffic-client] traffic client started for session ID: %s", clientId)

		for {
			conn, err := session.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				// Prefer the transport's reason (e.g. session gone on server) over yamux's bare EOF
				if cause := rc.Err(); cause != nil {
					return fmt.Errorf("session lost: %w", cause)
				}
				return fmt.Errorf("failed to accept connection: %w", err)
			}
			wg.Go(func() {
				defer conn.Close()
				// Handle the connection here, e.g., proxy traffic or perform other tasks
				property, err := protocol.ResolveConnProperty(conn)
				if err != nil {
					log.Printf("[traffic-client] failed to resolve connection property: %v", err)
					return
				}
				switch property.Pattern {
				case protocol.ProxyPattern:
					target := net.JoinHostPort(property.Host, strconv.Itoa(int(property.Port)))
					log.Printf("[traffic-client] connection request to: %s", target)
					remoteConn, err := proxyDialer(ctx, "tcp", target)
					if err != nil {
						log.Printf("[traffic-client] failed to connect to remote server: %v", err)
						return
					}
					defer remoteConn.Close()
					utils.Relay(conn, remoteConn)
				default:
					log.Printf("[traffic-client] unhandled connection pattern: %v", property.Pattern)
					return
				}
			})
		}
	}

	// The whole session management and connection handling logic
	var once sync.Once
	markReady := func() { once.Do(func() { close(ready) }) }
	establishable := false
	for attempt := 0; ; attempt++ {
		err := runSession(func() {
			attempt = 0
			establishable = true
			markReady()
		})
		// Canceled (e.g. Ctrl+C): runSession returns nil here, so check ctx first
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			log.Printf("[traffic-client] session failed: %v", err)
			if !establishable {
				return fmt.Errorf("session failed: %w", err)
			}
		}
		if attempt >= maxRetries {
			return fmt.Errorf("gave up after %d retries: %w", maxRetries, err)
		}
		wait := min(time.Second<<min(attempt, 5), 30*time.Second)
		log.Printf("[traffic-client] retry %d/%d in %v", attempt+1, maxRetries, wait)
		select {
		case <-time.After(wait):
			// Retry the session
		case <-ctx.Done():
			return nil
		}
	}
}
