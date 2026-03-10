package protocol

import (
	"log"
	"net"
	"netokeep/pkg/transport"
	"sync"

	"github.com/hashicorp/yamux"
)

type UserSession struct {
	PC  *transport.PersistentConn
	Mux *yamux.Session
}

var (
	sessions    sync.Map
	activateIDs []string
	mu          sync.Mutex
)

func Reconnect(sid string, conn net.Conn) bool {
	if val, ok := sessions.Load(sid); ok {
		user := val.(*UserSession)
		user.PC.UpdateConn(conn)
		return true
	}
	return false
}

func NewSession(sid string, pConn *transport.PersistentConn, session *yamux.Session) bool {
	user := &UserSession{PC: pConn, Mux: session}
	sessions.Store(sid, user)

	mu.Lock()
	activateIDs = append(activateIDs, sid)
	mu.Unlock()

	return true
}

func RemoveSession(sid string) {
	mu.Lock()
	defer mu.Unlock()
	for i, id := range activateIDs {
		for id == sid {
			activateIDs = append(activateIDs[:i], activateIDs[i+1:]...)
			break
		}
	}
	sessions.Delete(sid)
}

func Traffic2Session(clientConn net.Conn) {
	for {
		mu.Lock()
		if len(activateIDs) == 0 {
			mu.Unlock()
			log.Printf("⚠️ 无可用 Session，出口流量已阻断")
			return
		}

		// 1. 总是尝试当前队列的第一个（“承载者”）
		sid := activateIDs[0]
		val, ok := sessions.Load(sid)
		mu.Unlock()

		if !ok {
			// 这种属于索引不一致，保险起见清理一下
			cleanAndRetry(sid)
			continue
		}

		user := val.(*UserSession)

		// 2. 尝试在这个 Session 上开启一个逻辑流
		// 如果 PersistentConn 正在重连，这里会阻塞等待
		// 如果 PersistentConn 彻底断开了，这里会报 error
		stream, err := user.Mux.Open()
		if err != nil {
			log.Printf("❌ Session [%s] 已失效，尝试补位...", sid)
			RemoveSession(sid) // 踢掉挂了的，进入下一次循环尝试 activateIDs[1]
			continue
		}

		// 3. 成功开启流，发送暗号 0x02 (Socks5/流量转发)
		_, err = stream.Write([]byte{0x02})
		if err != nil {
			stream.Close()
			RemoveSession(sid)
			continue
		}

		// 4. 进入双向拷贝，完成接力
		// 这里的 Relay 结束后，协程自然退出
		transport.Relay(clientConn, stream)
		return
	}
}

// 辅助工具：强制清理
func cleanAndRetry(sid string) {
	mu.Lock()
	defer mu.Unlock()
	for i, id := range activateIDs {
		if id == sid {
			activateIDs = append(activateIDs[:i], activateIDs[i+1:]...)
			break
		}
	}
}
