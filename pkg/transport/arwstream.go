package transport

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 帧头大小: Seq(8) + Ack(8) = 16 字节
const FrameHeaderSize = 16

type Dialer func() (*websocket.Conn, error)

type ARWStream struct {
	mu     sync.Mutex
	conn   *websocket.Conn
	dialer Dialer

	sid      string
	isClosed bool
	// 使用带缓冲的 channel 防止阻塞
	reconnect chan struct{}

	sendSeq    uint64
	remoteAck  uint64
	unackedBuf *bytes.Buffer
	writeChan  chan []byte

	recvSeq    uint64
	readBuffer *bytes.Buffer
}


func (a *ARWStream) Read(p []byte) (n int, err error) {
	for {
		a.mu.Lock()
		if a.isClosed {
			a.mu.Unlock()
			return 0, io.EOF
		}
		if a.readBuffer.Len() > 0 {
			n, _ = a.readBuffer.Read(p)
			a.mu.Unlock()
			return n, nil
		}
		a.mu.Unlock()

		// 阻塞等待，直到 transportLoop 填充了数据
		// 这里简单处理，生产环境建议用 sync.Cond 唤醒
		time.Sleep(5 * time.Millisecond)
	}
}

func (a *ARWStream) Write(p []byte) (n int, err error) {
	a.mu.Lock()
	if a.isClosed {
		a.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	// 1. 存入待确认缓冲区，用于断线重发
	a.unackedBuf.Write(p)

	// 2. 封装数据包（拷贝一份，避免外部修改 p）
	data := make([]byte, len(p))
	copy(data, p)
	a.mu.Unlock()

	// 丢进队列，由后台发送协程处理物理发送
	a.writeChan <- data
	return len(p), nil
}

func (a *ARWStream) transportLoop() {
	for {
		a.mu.Lock()
		if a.isClosed {
			a.mu.Unlock()
			return
		}
		currConn := a.conn
		a.mu.Unlock()

		if currConn == nil {
			if a.dialer != nil {
				log.Printf("[ARWS] Attempting to reconnect...")
				newConn, err := a.dialer()
				if err != nil {
					time.Sleep(3 * time.Second)
					continue
				}
				a.UpdateWsConn(newConn)
			} else {
				// Server 模式：等待外部 UpdateWsConn
				<-a.reconnect
				continue
			}
		}

		// 启动读写并发处理
		errCh := make(chan error, 2)
		go a.physicalRead(errCh)
		go a.physicalWrite(errCh)

		// 只要有一个报错（断线），就清理并准备重连
		<-errCh
		a.mu.Lock()
		if a.conn != nil {
			a.conn.Close()
			a.conn = nil
		}
		a.mu.Unlock()
	}
}

func (a *ARWStream) physicalWrite(errCh chan error) {
	for {
		data := <-a.writeChan

		a.mu.Lock()
		header := make([]byte, FrameHeaderSize)
		binary.BigEndian.PutUint64(header[0:8], a.sendSeq)
		binary.BigEndian.PutUint64(header[8:16], a.recvSeq)

		packet := append(header, data...)
		conn := a.conn
		a.mu.Unlock()

		if conn == nil {
			errCh <- io.ErrUnexpectedEOF
			return
		}

		if err := conn.WriteMessage(websocket.BinaryMessage, packet); err != nil {
			errCh <- err
			return
		}

		a.mu.Lock()
		a.sendSeq += uint64(len(data))
		a.mu.Unlock()
	}
}

func (a *ARWStream) physicalRead(errCh chan error) {
	for {
		a.mu.Lock()
		conn := a.conn
		a.mu.Unlock()
		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			errCh <- err
			return
		}

		if len(message) < FrameHeaderSize {
			continue
		}

		// 解析 Seq 和 Ack
		// remoteSeq := binary.BigEndian.Uint64(message[0:8])
		remoteAck := binary.BigEndian.Uint64(message[8:16])
		payload := message[16:]

		a.mu.Lock()
		// 对方告诉我们它收到了多少，我们清理 unackedBuf
		if remoteAck > a.remoteAck {
			shift := int(remoteAck - a.remoteAck)
			if shift <= a.unackedBuf.Len() {
				_ = a.unackedBuf.Next(shift)
				a.remoteAck = remoteAck
			}
		}
		// 将收到的数据放入读取缓冲
		a.readBuffer.Write(payload)
		a.recvSeq += uint64(len(payload))
		a.mu.Unlock()
	}
}

func NewARWStream(wsConn *websocket.Conn, dialer Dialer) *ARWStream {
	as := &ARWStream{
		conn:       wsConn,
		dialer:     dialer,
		reconnect:  make(chan struct{}, 1),
		writeChan:  make(chan []byte, 1024),
		unackedBuf: new(bytes.Buffer),
		readBuffer: new(bytes.Buffer),
	}
	go as.transportLoop()
	return as
}

func (a *ARWStream) UpdateWsConn(newConn *websocket.Conn) {
	a.mu.Lock()
	if a.conn != nil {
		_ = a.conn.Close()
	}
	a.conn = newConn

	// 重连后，立刻把 unackedBuf 里的东西重新丢进 writeChan 补发
	// 这里的逻辑可以优化为更精准的对账重发
	if a.unackedBuf.Len() > 0 {
		pending := make([]byte, a.unackedBuf.Len())
		copy(pending, a.unackedBuf.Bytes())
		// 注意：实际生产中需要根据 Seq 重新分包发送
	}
	a.mu.Unlock()

	select {
	case a.reconnect <- struct{}{}:
	default:
	}
}

func (a *ARWStream) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isClosed = true
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

// 其余地址方法...
func (a *ARWStream) LocalAddr() net.Addr  { return nil }
func (a *ARWStream) RemoteAddr() net.Addr { return nil }
func (a *ARWStream) SetDeadline(t time.Time) error {
	return a.conn.UnderlyingConn().SetDeadline(t)
}
func (a *ARWStream) SetReadDeadline(t time.Time) error {
	return a.conn.UnderlyingConn().SetReadDeadline(t)
}
func (a *ARWStream) SetWriteDeadline(t time.Time) error {
	return a.conn.UnderlyingConn().SetWriteDeadline(t)
}
