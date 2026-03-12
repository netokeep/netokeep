package traffic

import (
	"context"
	"fmt"
	"log"
	"net"
	"netokeep/pkg/protocol"
	"sync"
)

func StartSocksListener(ctx context.Context, port uint16, handler func(conn *protocol.SocksConn)) error {
	var wg sync.WaitGroup
	lc := net.ListenConfig{}
	la := fmt.Sprintf("127.0.0.1:%d", port)
	l, err := lc.Listen(ctx, "tcp", la)
	if err != nil {
		log.Fatalf("Error in listening port %d: %v", port, err)
	}
	log.Printf("🌐 Container gateway started at port %d", port)

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			wg.Go(func() {
				defer conn.Close()
				// Small protocol to pack data.
				socksConn, err := protocol.GetSocksHandler(conn)
				if err != nil {
					log.Printf("[LISTENER] Error in getting socks handler: %v", err)
					return
				}
				handler(socksConn)
			})
		}
	}()

	<-ctx.Done()
	l.Close()
	wg.Wait()
	return nil
}

/*
StartSshListener starts a listener for SSH traffic on the specified port and handles incoming connections using the provided handler function.
*/
func StartSshListener(ctx context.Context, port uint16, handler func(conn net.Conn)) error {
	var wg sync.WaitGroup
	lc := net.ListenConfig{}
	la := fmt.Sprintf("127.0.0.1:%d", port)
	l, err := lc.Listen(ctx, "tcp", la)
	if err != nil {
		log.Fatalf("Error in listening port %d: %v", port, err)
	}
	log.Printf("🌐 SSH listener started at port %d", port)

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			wg.Go(func() {
				defer conn.Close()
				handler(conn)
			})
		}
	}()

	<-ctx.Done()
	l.Close()
	wg.Wait()
	return nil
}
