package transport

import (
	"io"
	"net"
	"sync"
)

type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnected
	StateReconnecting
)

type PersistentConn struct {
	net.Conn

	mu   sync.Mutex
	writeMu sync.Mutex
	readMu sync.Mutex
	cond *sync.Cond

	raw net.Conn

	State         ConnectionState
	StateChangeCh chan ConnectionState
}

func NewPersistentConn(conn net.Conn) *PersistentConn {
	pc := &PersistentConn{
		Conn:          conn,
		raw:           conn,
		State:         StateConnected,
		StateChangeCh: make(chan ConnectionState, 1),
	}
	pc.cond = sync.NewCond(&pc.mu)
	return pc
}

func (pc *PersistentConn) UpdateConn(newConn net.Conn) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.raw = newConn
	pc.setState(StateConnected)
	pc.cond.Broadcast()
}

func (pc *PersistentConn) Write(p []byte) (n int, err error) {
	pc.mu.Lock()
	pc.writeMu.Lock()
	defer pc.mu.Unlock()
	defer pc.writeMu.Unlock()

	for {
		if pc.State == StateDisconnected {
			return 0, io.ErrClosedPipe
		}
		if pc.raw != nil {
			r := pc.raw

			pc.mu.Unlock()
			n, err = r.Write(p)
			pc.mu.Lock()
			if err == nil {
				return n, nil
			}
			// If the write fails (due to an underlying disconnect),
			// reset the old connection to null and enter a waiting state.
			if pc.raw == r {
				pc.raw = nil
				pc.setState(StateReconnecting)
			}
		}
		// Block until UpdateConn is called
		pc.cond.Wait()
	}
}

func (pc *PersistentConn) Read(p []byte) (n int, err error) {
	pc.mu.Lock()
	pc.readMu.Lock()
	defer pc.mu.Unlock()
	defer pc.readMu.Unlock()

	for {
		if pc.State == StateDisconnected {
			return 0, io.ErrClosedPipe
		}

		if pc.raw != nil {
			r := pc.raw
			pc.mu.Unlock()
			n, err = r.Read(p)
			pc.mu.Lock()

			if err == nil {
				return n, nil
			}
			if pc.raw == r {
				pc.raw = nil
				pc.setState(StateReconnecting)
			}
		}
		pc.cond.Wait()
	}
}

func (pc *PersistentConn) Close() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.setState(StateDisconnected)
	if pc.raw != nil {
		pc.raw.Close()
	}
	pc.cond.Broadcast()
	return nil
}

/*
setState safely updates the connection state and notifies listeners.
It ensures that state changes are atomic and that listeners are notified of every change.
*/
func (pc *PersistentConn) setState(state ConnectionState) {
	if pc.State != state {
		pc.State = state
		select {
		case pc.StateChangeCh <- state:
		default:
			// If the channel is full, it means the state change hasn't been consumed yet.
			// We don't need to worry about it.
		}
	}
}
