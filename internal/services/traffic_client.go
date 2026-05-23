package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"netokeep/internal/protocol"
	"netokeep/internal/rules"
	"netokeep/internal/sessions"
	"netokeep/pkg/transport"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
	xproxy "golang.org/x/net/proxy"
)

func StartTrafficClient(ctx context.Context, manager *sessions.SessionManager, remoteAddr string, forwardTraffic bool, useProxy bool) error {
	var wg sync.WaitGroup
	// Generate a unique session ID for this client instance
	sid := uuid.New().String()
	// Setup yamux config
	cfg := yamux.DefaultConfig()
	// cfg.LogOutput = io.Discard
	cfg.EnableKeepAlive = false
	cfg.MaxStreamWindowSize = 4 * 1024 * 1024 // 4MB

	// Load client matcher for selecting the traffic to be proxied
	trafficDailer, err := createTrafficDialer(useProxy)
	if err != nil {
		return fmt.Errorf("failed to create traffic dialer: %v", err)
	}

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
		return fmt.Errorf("failed to connect to server: %v", err)
	}

	// Create a yamux client session to store the ws connection
	arwstream := transport.NewARWStream(ctx, wsConn, dailer)
	session, err := yamux.Client(arwstream, cfg)
	if err != nil {
		arwstream.Close()
		return fmt.Errorf("failed to create session: %v", err)
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
				pattern, host, port, err := protocol.ParseSocHeader(conn)
				if err != nil {
					log.Printf("[traffic] Failed to parse header form %s: %v", host, err)
					return
				}
				switch pattern {
				// The server will just actively send tcp request using channel
				case protocol.ProPattern:
					target := net.JoinHostPort(host, strconv.Itoa(int(port)))
					log.Printf("[traffic] Connection request to: %s", target)
					remoteConn, err := trafficDailer("tcp", target)
					if err != nil {
						log.Printf("[traffic] Failed to connect to target %s: %v", target, err)
						return
					}
					transport.Relay(conn, remoteConn)
				default:
					log.Printf("[traffic] Invalid request.")
					return
				}
			})
		}
	}()
	log.Printf("✨ NetoKeep connects to server successfully!")

	<-ctx.Done()
	manager.Close()
	wg.Wait()
	return nil
}

func createTrafficDialer(useProxy bool) (func(network, target string) (net.Conn, error), error) {
	defaultDialer := &net.Dialer{}

	if !useProxy {
		return defaultDialer.Dial, nil
	}

	allowList, proxyType, proxyAddr, proxyPort, err := rules.LoadClientRules()
	if err != nil {
		return nil, fmt.Errorf("error in loading client rules: %v", err)
	}

	proxyURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", proxyType, proxyAddr, proxyPort))
	if err != nil {
		return nil, fmt.Errorf("error in parsing proxy URL: %v", err)
	}
	proxyDialer, err := xproxy.FromURL(proxyURL, defaultDialer)
	if err != nil {
		return nil, fmt.Errorf("error in creating proxy dialer: %v", err)
	}
	dialer := xproxy.NewPerHost(defaultDialer, proxyDialer)
	dialer.AddFromString(strings.Join(allowList, ","))
	return dialer.Dial, nil
}
