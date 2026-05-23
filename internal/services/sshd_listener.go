package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"netokeep/internal/protocol"
	"netokeep/internal/sessions"
	"sync"
)

func StartSshdListener(ctx context.Context, manager *sessions.SessionManager, portSsh uint16) error {
	var wg sync.WaitGroup
	lc := net.ListenConfig{}
	// Use local adress to avoid external connections
	la := fmt.Sprintf("127.0.0.1:%d", portSsh)
	ln, err := lc.Listen(ctx, "tcp", la)
	if err != nil {
		return fmt.Errorf("error in listening port %d: %v", portSsh, err)
	}
	log.Printf("🌐 SSH listener started at port %d", portSsh)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("[listener] SSH listener closed.")
				return
			}
			wg.Go(func() {
				socConn := &protocol.SocConn{
					Conn: conn,
					Host: "placeholder",
					Port: 0,
				}

				// Add a ssh signature header for identification.
				header := socConn.CreateSocHeader(protocol.SshPattern)
				// Select one accessible session to forward outgoing traffic
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
