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

// rand.Text generates a random string for the client ID with length 26.
var clientId = rand.Text()

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

	// Initialize the wait group for managing concurrent operations
	var wg sync.WaitGroup
	defer wg.Wait()

	// Setup yamux config()
	yamuxConf := yamux.DefaultConfig()
	yamuxConf.EnableKeepAlive = false
	yamuxConf.ConnectionWriteTimeout = 10 * time.Second
	yamuxConf.MaxStreamWindowSize = 2 * 1024 * 1024 // 2MB

	// Load client matcher for selecting the traffic to be proxied
	proxyDialer, err := func() (func(ctx context.Context, network, target string) (net.Conn, error), error) {
		defaultDailer := &net.Dialer{Timeout: 3 * time.Second}
		if !conf.UseProxy {
			return defaultDailer.DialContext, nil
		}
		proxyConf, err := storage.LoadClientConf()
		if err != nil {
			return nil, fmt.Errorf("failed to load client proxy configuration: %w", err)
		}
		proxyURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", proxyConf.Type, proxyConf.Addr, proxyConf.Port))
		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
		}
		proxyDialer, err := xproxy.FromURL(proxyURL, defaultDailer)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy dialer: %w", err)
		}
		dialer := xproxy.NewPerHost(defaultDailer, proxyDialer)
		dialer.AddFromString(strings.Join(proxyConf.AllowList, ","))
		return dialer.DialContext, nil
	}()
	if err != nil {
		return fmt.Errorf("failed to create proxy dialer: %w", err)
	}

	// Create ws connection
	wsDialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	dailer := func(resume bool) (*websocket.Conn, error) {
		header := http.Header{}
		header.Set("NK-Session-ID", clientId)
		header.Set("NK-Forward-Traffic", strconv.FormatBool(conf.ForwardTraffic))
		header.Set("NK-Resume", strconv.FormatBool(resume))
		conn, _, err := wsDialer.DialContext(ctx, conf.RemoteAddr, header)
		return conn, err
	}

	wc, err := dailer(false)
	if err != nil {
		return fmt.Errorf("failed to create WebSocket connection: %w", err)
	}
	rc := transport.NewResumableConn(ctx, wc, dailer)
	session, err := yamux.Client(rc, yamuxConf)
	if err != nil {
		rc.Close()
		return fmt.Errorf("failed to create yamux session: %w", err)
	}
	defer session.Close()
	defer context.AfterFunc(ctx, func() { session.Close() })()

	conf.Manager.AddSession(clientId, &sessions.UserSession{
		Session:        session,
		Rc:             rc,
		ForwardTraffic: false,
	})
	defer conf.Manager.RemoveSession(clientId)

	close(ready)
	log.Printf("Traffic client started for session ID: %s", clientId)

	for {
		conn, err := session.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("failed to accept connection: %w", err)
		}
		wg.Go(func() {
			defer conn.Close()
			// Handle the connection here, e.g., proxy traffic or perform other tasks
			property, err := protocol.ResolveConnProperty(conn)
			if err != nil {
				log.Printf("Failed to resolve connection property: %v", err)
				return
			}
			switch property.Pattern {
			case protocol.ProxyPattern:
				target := net.JoinHostPort(property.Host, strconv.Itoa(int(property.Port)))
				log.Printf("Connection request to: %s", target)
				remoteConn, err := proxyDialer(ctx, "tcp", target)
				if err != nil {
					log.Printf("Failed to connect to remote server: %v", err)
					return
				}
				defer remoteConn.Close()
				utils.Relay(conn, remoteConn)
			default:
				log.Printf("Unhandled connection pattern: %v", property.Pattern)
				return
			}
		})
	}
}
