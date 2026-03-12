package session

import (
	"log"
	"net"
	"netokeep/pkg/transport"
	"sync"

	"github.com/hashicorp/yamux"
)

type SessionManager struct {
	// mu is assigned to protect the following fields:
	// 	- activateIDs
	// 	- pendingIDs
	mu sync.Mutex

	sessions        sync.Map
	connectedIDs    []string
	pendingIDs      []string
	disconnectedIDs []string

	PendingActiveCh chan string
}

type UserSession struct {
	PC  *transport.PersistentConn
	Mux *yamux.Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		PendingActiveCh: make(chan string, 16),
	}
}

func (sm *SessionManager) NewSession(sid string, pConn *transport.PersistentConn, session *yamux.Session) bool {
	user := &UserSession{PC: pConn, Mux: session}
	sm.sessions.Store(sid, user)

	sm.mu.Lock()
	sm.connectedIDs = append(sm.connectedIDs, sid)
	sm.mu.Unlock()

	// Listen for session state change in a separate goroutine
	go func() {
		for state := range user.PC.StateChangeCh {
			log.Printf("Session [%s] state changed to: %d", sid, state)
			switch state {
			case transport.StateDisconnected:
				sm.mu.Lock()
				sm.moveID(&sm.connectedIDs, &sm.disconnectedIDs, sid)
				sm.moveID(&sm.pendingIDs, &sm.disconnectedIDs, sid)
				sm.sessions.Delete(sid)
				sm.mu.Unlock()
			case transport.StateReconnecting:
				sm.mu.Lock()
				sm.moveID(&sm.connectedIDs, &sm.pendingIDs, sid)
				sm.mu.Unlock()
			case transport.StateConnected:
				sm.mu.Lock()
				sm.moveID(&sm.pendingIDs, &sm.connectedIDs, sid)
				sm.mu.Unlock()
			}
		}
	}()

	return true
}

func (sm *SessionManager) Reconnect(sid string, conn net.Conn) bool {
	if val, ok := sm.sessions.Load(sid); ok {
		user := val.(*UserSession)
		user.PC.UpdateConn(conn)
		return true
	}
	return false
}

func (sm *SessionManager) Traffic2Session(clientConn net.Conn, header []byte) {
	for {
		sm.mu.Lock()
		if len(sm.connectedIDs) == 0 {
			sm.mu.Unlock()
			log.Printf("[SESSION] No available session, waiting for reconnection...")
			clientConn.Close()
			return
		}

		// 1. Find the first available session to forward traffic
		sid := sm.connectedIDs[0]
		val, _ := sm.sessions.Load(sid)
		sm.mu.Unlock()

		// 2. Try to open a logical stream on this Session.
		// If the PersistentConn is reconnecting, this will block until it finishes.
		// If the PersistentConn is fully disconnected, this will error out.
		user := val.(*UserSession)
		stream, err := user.Mux.Open()
		if err != nil {
			log.Printf("[SESSION] Session [%s] failed to open stream, falling back to another session...", sid)
			continue
		}

		// 3. Succeed in opening a stream, send the constructed header
		_, err = stream.Write(header)
		if err != nil {
			log.Printf("[SESSION] Session [%s] failed to write header, falling back to another session...", sid)
			stream.Close()
			continue
		}

		// 4. Entering Relay, exit after completion
		transport.Relay(clientConn, stream)
		return
	}
}

func (sm *SessionManager) moveID(from *[]string, to *[]string, sid string) {
	for i, id := range *from {
		if id == sid {
			*from = append((*from)[:i], (*from)[i+1:]...)
			break
		}
	}
	*to = append(*to, sid)

	// Handle pending active signal
	if to == &sm.pendingIDs && len(sm.pendingIDs) > 0 {
		select {
		case sm.PendingActiveCh <- sid:
		default:
		}
	}
}
