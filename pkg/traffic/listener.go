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
		return err
	}
	log.Printf("🌐 Container gateway started: %d", port)

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
					log.Fatalf("[LISTENER] Error in getting socks handler: %v", err)
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
