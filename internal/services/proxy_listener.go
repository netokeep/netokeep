package services

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"netokeep/internal/protocol"
	"netokeep/internal/rules"
	"netokeep/internal/sessions"
	"strconv"
	"sync"
)

/*
StartProxyListener create one http proxy server to receive local traffic from `listenPort`

StartProxy dose not defer the connection, the caller should handle the connection lifecycle in the handler function.
*/
func StartProxyListener(ctx context.Context, manager *sessions.SessionManager, listenPort uint16) error {
	var wg sync.WaitGroup
	lc := net.ListenConfig{}

	// Load server matcher for validating incoming connection requests
	matcher, err := rules.LoadServerMatcher()
	if err != nil {
		return fmt.Errorf("error in loading server matcher: %v", err)
	}

	// Use local adress to avoid external connections
	la := fmt.Sprintf("127.0.0.1:%d", listenPort)
	ln, err := lc.Listen(ctx, "tcp", la)
	if err != nil {
		return fmt.Errorf("error in listening %d: %v", listenPort, err)
	}
	log.Printf("🌐 HTTP proxy listener started at port %d", listenPort)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("[listener] Proxy listener closed.")
				return
			}
			wg.Go(func() {
				// Handle the handshake of HTTP and return the conn with host and port
				request, err := http.ReadRequest(bufio.NewReader(conn))
				if err != nil {
					log.Printf("[listener] Error in reading the request header: %v", err)
					conn.Close()
					return
				}
				host, portStr, err := net.SplitHostPort(request.Host)
				if err != nil {
					host = request.Host
					if request.Method == http.MethodConnect {
						portStr = "443"
					} else {
						portStr = "80"
					}
				}

				// match the request host with server rules. If not matched, close the connection directly.
				if !matcher.Match(host) {
					log.Printf("[listener] Unmatched host: %s, closing.", host)
					conn.Close()
					return
				}
				log.Printf("[listener] Connection request to: %s", host)
				p, _ := strconv.ParseUint(portStr, 10, 16)
				port := uint16(p)

				// Construct SocConn
				socConn := &protocol.SocConn{
					Conn: conn,
					Host: host,
					Port: port,
				}

				// Handle CONNECT (usually used for HTTPS traffic)
				if request.Method == http.MethodConnect {
					_, err = conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
					if err != nil {
						log.Printf("[listener] Error in CONNECT request handshake: %v", err)
						conn.Close()
						return
					}
				} else {
					request.RequestURI = ""
					var buffer bytes.Buffer
					request.Write(&buffer)

					socConn.Conn = &protocol.PrependConn{
						Conn:   conn,
						Buffer: buffer.Bytes(),
					}
				}

				// Add a simple header for the proxy traffic detection.
				header := socConn.CreateSocHeader(protocol.ProPattern)
				// Select one accessible session to forward outgoing traffic
				// The control of conn is handed to Traffic2Session if no error is returned.
				if err := manager.Traffic2Session(socConn, header); err != nil {
					log.Printf("[listener] Failed to forward traffic to session: %v", err)
					conn.Close()
					return
				}
			})
		}
	}()

	<-ctx.Done()
	ln.Close()
	wg.Wait()
	return nil
}
