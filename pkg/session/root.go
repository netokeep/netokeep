package session

import (
	"log"
	"net"
	"netokeep/pkg/transport"
	"sync"

	"github.com/hashicorp/yamux"
)

type SessionManager struct {
	sessions    sync.Map
	activateIDs []string
	mu          sync.Mutex
}

type UserSession struct {
	PC  *transport.PersistentConn
	Mux *yamux.Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{}
}

func (sm *SessionManager) Reconnect(sid string, conn net.Conn) bool {
	if val, ok := sm.sessions.Load(sid); ok {
		user := val.(*UserSession)
		user.PC.UpdateConn(conn)
		return true
	}
	return false
}

func (sm *SessionManager) NewSession(sid string, pConn *transport.PersistentConn, session *yamux.Session) bool {
	println(sid)
	user := &UserSession{PC: pConn, Mux: session}
	sm.sessions.Store(sid, user)

	sm.mu.Lock()
	sm.activateIDs = append(sm.activateIDs, sid)
	sm.mu.Unlock()

	return true
}

func (sm *SessionManager) RemoveSession(sid string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for i, id := range sm.activateIDs {
		for id == sid {
			sm.activateIDs = append(sm.activateIDs[:i], sm.activateIDs[i+1:]...)
			break
		}
	}
	sm.sessions.Delete(sid)
}

func (sm *SessionManager) Traffic2Session(clientConn net.Conn, header []byte) {
	for {
		sm.mu.Lock()
		if len(sm.activateIDs) == 0 {
			sm.mu.Unlock()
			log.Printf("⚠️ 无可用 Session，出口流量已阻断")
			clientConn.Close()
			return
		}

		// 1. 总是尝试当前队列的第一个（“承载者”）
		sid := sm.activateIDs[0]
		val, ok := sm.sessions.Load(sid)
		sm.mu.Unlock()

		if !ok {
			// 这种属于索引不一致，保险起见清理一下
			sm.cleanAndRetry(sid)
			continue
		}

		user := val.(*UserSession)

		// 2. 尝试在这个 Session 上开启一个逻辑流
		// 如果 PersistentConn 正在重连，这里会阻塞等待
		// 如果 PersistentConn 彻底断开了，这里会报 error
		stream, err := user.Mux.Open()
		if err != nil {
			log.Printf("❌ Session [%s] 已失效，尝试补位...", sid)
			sm.RemoveSession(sid) // 踢掉挂了的，进入下一次循环尝试 activateIDs[1]
			continue
		}

		// 3. 成功开启流，发送构建的头部
		_, err = stream.Write(header)
		if err != nil {
			stream.Close()
			sm.RemoveSession(sid)
			continue
		}

		// 4. 进入双向拷贝，完成接力
		// 这里的 Relay 结束后，协程自然退出
		transport.Relay(clientConn, stream)
		return
	}
}

// 辅助工具：强制清理
func (sm *SessionManager) cleanAndRetry(sid string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for i, id := range sm.activateIDs {
		if id == sid {
			sm.activateIDs = append(sm.activateIDs[:i], sm.activateIDs[i+1:]...)
			break
		}
	}
}
