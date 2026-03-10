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
	var sid string // 对应服务端的 X-Session-ID

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the netokeep client.",
		Run: func(cmd *cobra.Command, args []string) {
			// 1. 信号控制与上下文
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// 格式化 URL，确保以 ws:// 开头
			remoteURL := remoteAddr
			if !strings.HasPrefix(remoteURL, "ws://") && !strings.HasPrefix(remoteURL, "wss://") {
				remoteURL = "ws://" + remoteURL
			}

			// 2. 发起带身份标识的 WebSocket 连接
			header := http.Header{}
			header.Add("X-Session-ID", sid)

			dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
			wsConn, _, err := dialer.Dial(remoteURL, header)
			if err != nil {
				log.Fatalf("❌ 无法连接到 NKS: %v", err)
			}

			// 3. 包装流并穿上“重连防弹衣”
			wsStream := transport.NewWsStream(wsConn)
			// 这里 pConn 会处理 30s 内的断线静默等待
			pConn := transport.NewPersistentConn(wsStream)

			// 4. 启动 Yamux 客户端
			conf := yamux.DefaultConfig()
			conf.KeepAliveInterval = 10 * time.Second
			// 这里的超时要配合服务端的重连时间
			conf.ConnectionWriteTimeout = 35 * time.Second

			session, err := yamux.Client(pConn, conf)
			if err != nil {
				log.Fatalf("❌ 建立隧道失败: %v", err)
			}

			log.Printf("🚀 NetoKeep 隧道已接通！[ID: %s]", sid)

			// 5. 开启业务监听循环（等待服务端指令）
			go func() {
				for {
					// Accept 会阻塞，直到服务端调用 mux.Open()
					stream, err := session.Accept()
					if err != nil {
						log.Printf("⚠️ 隧道会话已关闭: %v", err)
						return
					}
					// 异步处理每一个来自服务端的请求
					go handleServerRequest(stream)
				}
			}()

			// 阻塞直到收到退出信号
			<-ctx.Done()
			log.Println("👋 客户端正在退出...")
			session.Close()
		},
	}

	// 这里的参数要和服务端对齐
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
