package transport

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WsStream struct {
	*websocket.Conn
	reader io.Reader
}

func NewWsStream(conn *websocket.Conn) *WsStream {
	return &WsStream{Conn: conn}
}

func (w *WsStream) Read(p []byte) (n int, err error) {
	if w.reader == nil {
		_, r, err := w.NextReader()
		if err != nil {
			return 0, err
		}
		w.reader = r
	}
	n, err = w.reader.Read(p)
	if err == io.EOF {
		w.reader = nil
		return n, nil
	}
	return n, err
}

func (w *WsStream) Write(p []byte) (n int, err error) {
	err = w.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *WsStream) LocalAddr() net.Addr           { return w.Conn.LocalAddr() }
func (w *WsStream) RemoteAddr() net.Addr          { return w.Conn.RemoteAddr() }
func (w *WsStream) SetDeadline(t time.Time) error { return w.Conn.UnderlyingConn().SetDeadline(t) }
func (w *WsStream) SetReadDeadline(t time.Time) error {
	return w.Conn.UnderlyingConn().SetReadDeadline(t)
}
func (w *WsStream) SetWriteDeadline(t time.Time) error {
	return w.Conn.UnderlyingConn().SetWriteDeadline(t)
}

type PersistentConn struct {
	net.Conn

	mu     sync.Mutex
	cond   *sync.Cond
	raw    net.Conn
	closed bool
}

func NewPersistentConn(conn net.Conn) *PersistentConn {
	pc := &PersistentConn{
		Conn: conn,
		raw:  conn,
	}
	pc.cond = sync.NewCond(&pc.mu)
	return pc
}

func (pc *PersistentConn) UpdateConn(newConn net.Conn) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.raw = newConn
	pc.cond.Broadcast()
}

func (pc *PersistentConn) Write(p []byte) (n int, err error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	for {
		if pc.closed {
			return 0, io.ErrClosedPipe
		}
		if pc.raw != nil {
			n, err = pc.raw.Write(p)
			if err == nil {
				return n, nil
			}
			// If the write fails (due to an underlying disconnect),
			// reset the old connection to null and enter a waiting state.
			pc.raw = nil
		}
		// Block until UpdateConn is called
		pc.cond.Wait()
	}
}

func (pc *PersistentConn) Read(p []byte) (n int, err error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	for {
		if pc.closed {
			return 0, io.EOF
		}
		if pc.raw != nil {
			n, err = pc.raw.Read(p)
			if err == nil {
				return n, nil
			}
			pc.raw = nil
		}
		pc.cond.Wait()
	}
}

func (pc *PersistentConn) Close() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.closed = true
	if pc.raw != nil {
		pc.raw.Close()
	}
	pc.cond.Broadcast()
	return nil
}
