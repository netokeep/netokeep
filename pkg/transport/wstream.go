package transport

import (
	"io"
	"time"

	"github.com/gorilla/websocket"
)

type Wstream struct {
	*websocket.Conn
	reader io.Reader
}

/*
Wstream wraps a websocket.

Conn to implement the net.Conn interface,
allowing it to be used as a regular network connection.
*/
func NewWstream(conn *websocket.Conn) *Wstream {
	return &Wstream{Conn: conn}
}

func (w *Wstream) Read(p []byte) (n int, err error) {
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

func (w *Wstream) Write(p []byte) (n int, err error) {
	err = w.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *Wstream) SetDeadline(t time.Time) error {
	return w.Conn.UnderlyingConn().SetDeadline(t)
}
func (w *Wstream) SetReadDeadline(t time.Time) error {
	return w.Conn.UnderlyingConn().SetReadDeadline(t)
}
func (w *Wstream) SetWriteDeadline(t time.Time) error {
	return w.Conn.UnderlyingConn().SetWriteDeadline(t)
}
