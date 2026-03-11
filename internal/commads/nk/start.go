package nk

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"netokeep/pkg/transport"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
	"github.com/spf13/cobra"
)

func CreateStartCmd() *cobra.Command {
	var remoteAddr string
	var sid string

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the netokeep client.",
		Run: func(cmd *cobra.Command, args []string) {
			// Setup graceful shutdown context
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// Preprocess the URL
			if strings.Contains(remoteAddr, "://") {
				// Currently only supports ws
				remoteAddr = "ws://" + strings.Split(remoteAddr, "://")[1]
			}

			// 2. 创建一个空的 PersistentConn (此时还没有真正的物理连接)
			// 注意：你需要确保你的 transport.PersistentConn 结构体里有 RequireWS chan struct{}
			pConn := transport.NewPersistentConn(nil)

			// 3. 启动 Yamux 客户端 (套在 pConn 这个壳子上)
			conf := yamux.DefaultConfig()
			conf.KeepAliveInterval = 10 * time.Second
			conf.ConnectionWriteTimeout = 60 * time.Second // 给重连留够 60s 时间

			session, err := yamux.Client(pConn, conf)
			if err != nil {
				log.Fatalf("❌ 建立隧道失败: %v", err)
			}

			// 4. 开启业务监听 (只启动一次)
			go func() {
				for {
					stream, err := session.Accept()
					if err != nil {
						log.Printf("⚠️ Yamux 会话已关闭: %v", err)
						return
					}
					go handleServerRequest(stream)
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
			log.Println("👋 客户端正在退出...")
			session.Close()
		},
	}

	startCmd.Flags().StringVarP(&remoteAddr, "remoteAddress", "r", "127.0.0.1:7222", "NKS server address")
	startCmd.Flags().StringVarP(&sid, "id", "n", "shun-client", "Session ID for identification")
	startCmd.MarkFlagRequired("remoteAddress")

	return startCmd
}

// 核心业务分发逻辑
func handleServerRequest(stream net.Conn) {
	defer stream.Close()

	// 1. 读取 1 字节暗号（由 NKS 发出）
	protocolType := make([]byte, 1)
	if _, err := io.ReadFull(stream, protocolType); err != nil {
		return
	}

	switch protocolType[0] {
	case 0x01: // SSH 业务：NKS 想访问我这边的 22 端口
		log.Println("🔑 接入 SSH 转发请求 -> localhost:22")
		target, err := net.DialTimeout("tcp", "127.0.0.1:22", 5*time.Second)
		if err != nil {
			log.Printf("❌ 无法连接本地 SSH 服务: %v", err)
			return
		}
		defer target.Close()
		transport.Relay(stream, target)

	case 0x02: // 流量网关业务：NKS 想借我这边的网出去
		log.Println("🌐 接入出口流量请求 -> 准备代理转发")
		// 这里暂代：直接调用处理函数，下一阶段我们实现 Socks5 握手
		handleExitTraffic(stream)

	default:
		log.Printf("❓ 收到未知业务暗号: 0x%x", protocolType[0])
	}
}

func handleExitTraffic(stream net.Conn) {
	// TODO: 实现真正的出口转发逻辑（Socks5 或直接透传）
	// 顺哥，这里如果是 Socks5，我们需要在 NK 端跑一个真正的代理逻辑。
}
