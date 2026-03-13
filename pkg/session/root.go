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

	sessions     sync.Map
	connectedIDs []string
	pendingIDs   []string

	PendingActiveCh chan string
}

type UserSession struct {
	pc  *transport.PersistentConn
	mux *yamux.Session
}

/*
NewSessionManager creates one session manager with at most 16 sessions.

Remember that the `PendingActiveCh` does not have the close logic, so do not use it in `for range`.
*/
func NewSessionManager() *SessionManager {
	return &SessionManager{
		PendingActiveCh: make(chan string, 16),
	}
}

func (sm *SessionManager) Close() {
	sm.mu.Lock()
	ch := sm.PendingActiveCh
	// Set to nil first so any future moveID() calls won't send into a closed channel.
	sm.PendingActiveCh = nil
	sm.mu.Unlock()

	// Close all active sessions.
	sm.sessions.Range(func(key, val any) bool {
		user, ok := val.(*UserSession)
		if ok {
			if user.mux != nil {
				user.mux.Close()
			}
			if user.pc != nil {
				user.pc.Close()
			}
		}
		sm.sessions.Delete(key)
		return true
	})

	if ch != nil {
		close(ch)
	}
}

func (sm *SessionManager) NewSession(sid string, pConn *transport.PersistentConn, session *yamux.Session) {
	user := &UserSession{pc: pConn, mux: session}
	sm.sessions.Store(sid, user)

	sm.mu.Lock()
	sm.connectedIDs = append(sm.connectedIDs, sid)
	sm.mu.Unlock()

	// Listen for session state change in a separate goroutine
	go func() {
		for state := range user.pc.StateChangeCh {
			switch state {
			case transport.StateDisconnected:
				sm.mu.Lock()
				sm.removeID(&sm.connectedIDs, sid)
				sm.removeID(&sm.pendingIDs, sid)
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
}

/*
Reconnect trys to replace the old `conn` to the session `sid` to restore the connection.
*/
func (sm *SessionManager) Reconnect(sid string, conn net.Conn) bool {
	if val, ok := sm.sessions.Load(sid); ok {
		user := val.(*UserSession)
		user.pc.UpdateConn(conn)
		return true
	}
	return false
}

/*
Traffic2Session allows you to push the `clientConn` into one available session.

Returns if no connection is available.
*/
func (sm *SessionManager) Traffic2Session(clientConn net.Conn, header []byte) {
	for {
		sm.mu.Lock()
		if len(sm.connectedIDs) == 0 {
			sm.mu.Unlock()
			log.Printf("[SESSION] No available session, waiting for new connections...")
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
		stream, err := user.mux.Open()
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
		stream.Close()
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

func (sm *SessionManager) removeID(from *[]string, sid string) {
	for i, id := range *from {
		if id == sid {
			*from = append((*from)[:i], (*from)[i+1:]...)
			break
		}
	}
}
