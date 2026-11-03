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

type SSHForwardConf struct {
	Manager *sessions.SessionManager
	SSHPort uint16
}

func StartSSHForwardListener(
	ctx context.Context,
	stop context.CancelFunc,
	ready chan struct{},
	conf SSHForwardConf,
) error {
	// Stop the whole program once the SSH forward listener exits
	defer stop()

	var wg sync.WaitGroup
	defer wg.Wait()

	lc := net.ListenConfig{}
	la := fmt.Sprintf("127.0.0.1:%d", conf.SSHPort)
	ln, err := lc.Listen(ctx, "tcp", la)
	if err != nil {
		return fmt.Errorf("failed to start SSH forward listener: %w", err)
	}
	defer ln.Close()
	defer context.AfterFunc(ctx, func() { ln.Close() })()

	close(ready)
	log.Printf("🌐 SSH listener started at port %d", conf.SSHPort)

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("SSH listener closed: %v", err)
		}
		wg.Go(func() {
			log.Printf("Accepted SSH connection from %s", conn.RemoteAddr())

			// Add a ssh signature header for identification.
			conn, err = protocol.InsertConnProperty(conn, protocol.ConnProperty{Pattern: protocol.SSHPattern})
			if err != nil {
				log.Printf("Failed to insert SSH connection property: %v", err)
				conn.Close()
				return
			}
			// Select one accessible session to forward outgoing traffic
			if err := conf.Manager.Traffic2Session(conn); err != nil {
				log.Printf("Failed to forward traffic to session: %v", err)
				conn.Close()
				return
			}
		})
	}
}
