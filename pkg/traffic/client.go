package traffic

import (
	"context"
	"log"
	"net"
	"net/http"
	"netokeep/pkg/transport"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

func StartClient(ctx context.Context, remoteAddr string, handler func(conn net.Conn)) {
	var wg sync.WaitGroup
	// Generate a unique session ID for this client instance
	sid := uuid.New().String()

	// Process the remote address to ensure it has the correct WebSocket scheme
	if strings.Contains(remoteAddr, "://") {
		remoteAddr = "ws://" + strings.Split(remoteAddr, "://")[1]
	}

	pConn := transport.NewPersistentConn(nil)
	conf := yamux.DefaultConfig()
	conf.KeepAliveInterval = 10 * time.Second
	conf.ConnectionWriteTimeout = 60 * time.Second // 给重连留够 60s 时间

	s, err := yamux.Client(pConn, conf)
	if err != nil {
		log.Fatalf("❌ 建立隧道失败: %v", err)
	}

	go func() {
		defer s.Close()

		for {
			conn, err := s.Accept()
			if err != nil {
				log.Printf("⚠️ Yamux 会话已关闭: %v", err)
				return
			}
			wg.Go(func() {
				defer conn.Close()
				handler(conn)
			})
		}
	}()

	// 5. 【核心重连循环】参考 rtunnel 机制
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-pConn.RequireWS: // 阻塞等待断线信号
				log.Println("🔄 检测到链路断开或初始化，正在发起连接...")

				// 最多尝试 5 次连接，每次间隔 3 秒

				for { // 内部重连死循环
					if ctx.Err() != nil {
						return
					}

					header := http.Header{}
					header.Add("X-Session-ID", sid)

					wsConn, _, err := websocket.DefaultDialer.Dial(remoteAddr, header)
					if err != nil {
						log.Printf("❌ 拨号失败: %v，3秒后重试...", err)
						time.Sleep(3 * time.Second)
						continue
					}

					// 成功连上，把新连接包装并注入到 pConn 壳子里
					pConn.UpdateConn(transport.NewWsStream(wsConn))
					log.Printf("🚀 NetoKeep 隧道已就绪！[ID: %s]", sid)
					break // 跳出内部重连，回到外层等待下一次信号
				}

			}
		}
	}()

	// 首次启动手动触发一次连接信号
	pConn.RequireWS <- struct{}{}
	<-ctx.Done()
	wg.Wait()
	log.Println("👋 客户端正在退出...")
}
