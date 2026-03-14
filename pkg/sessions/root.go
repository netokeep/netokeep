package sessions

import (
	"log"
	"net"
	"netokeep/pkg/transport"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

type SessionManager struct {
	sessions sync.Map
}

type UserSession struct {
	session   *yamux.Session
	arwstream *transport.ARWStream
}

/*
NewSessionManager creates one session manager with at most 16 sessions.
*/
func NewSessionManager() *SessionManager {
	return &SessionManager{}
}

func (sm *SessionManager) Close() {
	// Close all active sessions.
	sm.sessions.Range(func(key, val any) bool {
		userSession, ok := val.(*UserSession)
		if ok {
			userSession.session.Close()
			userSession.arwstream.Close()
		}
		sm.sessions.Delete(key)
		return true
	})
}

func (sm *SessionManager) NewSession(sid string, session *yamux.Session, arwstream *transport.ARWStream) {
	sm.sessions.Store(sid, &UserSession{
		session:   session,
		arwstream: arwstream,
	})
}

func (sm *SessionManager) HasSession(sid string) bool {
	_, ok := sm.sessions.Load(sid)
	return ok
}

func (sm *SessionManager) UpdateSession(sid string, wsConn *websocket.Conn) {
	if val, ok := sm.sessions.Load(sid); ok {
		userSession, ok := val.(*UserSession)
		if ok {
			userSession.arwstream.UpdateWsConn(wsConn)
		}
	}
}

func (sm *SessionManager) RemoveSession(sid string) {
	if val, ok := sm.sessions.Load(sid); ok {
		userSession, ok := val.(*UserSession)
		if ok {
			userSession.session.Close()
			userSession.arwstream.Close()
		}
	}
	sm.sessions.Delete(sid)
}

/*
Traffic2Session allows you to push the `clientConn` into one available session.

Returns if no connection is available.
*/
func (sm *SessionManager) Traffic2Session(clientConn net.Conn, header []byte) {
	var success bool
	sm.sessions.Range(func(key, val any) bool {
		sid := key.(string)
		userSession := val.(*UserSession)
		session := userSession.session
		if session == nil || session.IsClosed() {
			return true
		}

		stream, err := session.Open()
		if err != nil {
			log.Printf("[SESSION] Session [%s] failed to open stream, falling back to another session...", sid)
			return true
		}

		_, err = stream.Write(header)
		if err != nil {
			log.Printf("[SESSION] Session [%s] failed to write header, falling back to another session...", sid)
			stream.Close()
			return true
		}

		// Entering Relay, exit after completion
		go func() {
			transport.Relay(clientConn, stream)
			clientConn.Close()
			stream.Close()
		}()
		success = true
		return false // stop ranging after successfully forwarding the traffic
	})
	if !success {
		log.Printf("[SESSION] No available session, waiting for new connections...")
		clientConn.Close()
	}
}
