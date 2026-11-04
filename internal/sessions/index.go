package sessions

import (
	"fmt"
	"net"
	"netokeep/internal/protocol"
	"netokeep/internal/transport"
	"netokeep/internal/utils"
	"sync"

	"github.com/hashicorp/yamux"
)

type SessionManager struct {
	sessions sync.Map
}

type UserSession struct {
	Session        *yamux.Session
	Rc             *transport.ResumableConn
	ForwardTraffic bool
}

func (m *SessionManager) AddSession(id string, us *UserSession) error {
	if _, loaded := m.sessions.LoadOrStore(id, us); loaded {
		return fmt.Errorf("session %s already exists", id)
	}
	return nil
}

func (m *SessionManager) RemoveSession(id string) {
	m.sessions.Delete(id)
}

// Get returns the session registered under id.
func (m *SessionManager) Get(id string) (*UserSession, bool) {
	val, ok := m.sessions.Load(id)
	if !ok {
		return nil, false
	}
	return val.(*UserSession), true
}

func (m *SessionManager) pick(needForward bool) (*yamux.Session, error) {
	var session *yamux.Session
	m.sessions.Range(func(key, val any) bool {
		userSession := val.(*UserSession)
		if userSession.Session != nil && !userSession.Session.IsClosed() && (!needForward || userSession.ForwardTraffic) {
			session = userSession.Session
			return false
		}
		return true
	})
	if session == nil {
		return nil, fmt.Errorf("no available session")
	}
	return session, nil
}

func (m *SessionManager) OpenStream(property protocol.ConnProperty, needForward bool) (net.Conn, error) {
	encodedProperty, err := protocol.EncodeConnProperty(property)
	if err != nil {
		return nil, err
	}
	// Pick one session that has the ability to forward traffic
	session, err := m.pick(needForward)
	if err != nil {
		return nil, err
	}
	stream, err := session.Open()
	if err != nil {
		return nil, err
	}
	_, err = stream.Write(encodedProperty)
	if err != nil {
		stream.Close()
		return nil, err
	}
	return stream, nil
}

/*
Traffic2Session allows you to push the `clientConn` into one available session.

Blocks until the relay finishes. On error the caller still owns conn.
*/
func (m *SessionManager) Traffic2Session(conn net.Conn, property protocol.ConnProperty) error {
	stream, err := m.OpenStream(property, false)
	if err != nil {
		return err
	}
	utils.Relay(conn, stream)
	return nil
}
