package transport

import (
	"io"
	"time"

	"github.com/gorilla/websocket"
)

type WsStream struct {
	*websocket.Conn
	reader io.Reader
}

/*
WsStream wraps a websocket.

Conn to implement the net.Conn interface,
allowing it to be used as a regular network connection.
*/
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

func (w *WsStream) SetDeadline(t time.Time) error {
	return w.Conn.UnderlyingConn().SetDeadline(t)
}
func (w *WsStream) SetReadDeadline(t time.Time) error {
	return w.Conn.UnderlyingConn().SetReadDeadline(t)
}
func (w *WsStream) SetWriteDeadline(t time.Time) error {
	return w.Conn.UnderlyingConn().SetWriteDeadline(t)
}
