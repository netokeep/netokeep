package traffic

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"net"
// 	"netokeep/pkg/protocol"
// 	"sync"
// )

// func StartSocksProxy(ctx context.Context, port uint16) error {
// 	var wg sync.WaitGroup
// 	lc := net.ListenConfig{}
// 	la := fmt.Sprintf("127.0.0.1:%d", port)
// 	l, err := lc.Listen(ctx, "tcp", la)
// 	if err != nil {
// 		return err
// 	}

// 	go func() {
// 		for {
// 			conn, err := l.Accept()
// 			if err != nil {
// 				return
// 			}
// 			wg.Go(func() {
// 				defer conn.Close()
// 				// Small protocol to pack data.

// 			}()
// 		}
// 	}
