package transport

/*
TODO: The current version treats playback, real-time writing, and ping-pong equally.
*/

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Frame header consists of:
// Seq (8 bytes) + Ack (8 bytes)
const FrameHeaderSize = 16

type Dialer func() (*websocket.Conn, error)

type sendSegment struct {
	seq  uint64
	data []byte
}

type ARWStream struct {
	*websocket.Conn

	mu      sync.Mutex
	writemu sync.Mutex
	readmu  sync.Mutex
	dialer  Dialer
	ctx     context.Context
	cancel  context.CancelFunc

	isClosed     bool
	reconnecting bool
	reconnected  sync.Cond
	dataReady    sync.Cond

	// sendSeq is the sequence number for outgoing data
	sendSeq   uint64
	remoteAck uint64
	expSeq    uint64

	recvBuf bytes.Buffer
	recvSeg map[uint64][]byte // key is the starting Seq of the segment
	sendQue []sendSegment
}

func NewARWStream(ctx context.Context, wsConn *websocket.Conn, dialer Dialer) *ARWStream {
	ctx, cancel := context.WithCancel(ctx)
	as := &ARWStream{
		Conn:    wsConn,
		ctx:     ctx,
		cancel:  cancel,
		dialer:  dialer,
		recvSeg: make(map[uint64][]byte),
		sendQue: make([]sendSegment, 0),
	}
	as.reconnected.L = &as.mu
	as.dataReady.L = &as.mu
	// Create goroutine to handle keep-alive and read operations
	go as.keepAlive()
	go as.wsReadLoop()

	return as
}

func (as *ARWStream) keepAlive() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			as.mu.Lock()
			if as.isClosed {
				as.mu.Unlock()
				return
			}
			conn := as.Conn
			as.mu.Unlock()
			as.writemu.Lock()
			err := conn.WriteMessage(websocket.PingMessage, nil)
			as.writemu.Unlock()
			if err != nil {
				as.mu.Lock()
				if !as.isClosed && as.Conn == conn {
					as.mu.Unlock()
					go as.reconnect(as.dialer)
					as.mu.Lock()
				}
				for !as.isClosed && as.Conn == conn {
					as.reconnected.Wait()
				}
				as.mu.Unlock()
			}
		case <-as.ctx.Done():
			return
		}
	}
}

func (as *ARWStream) reconnect(dialer Dialer) {
	var success bool
	var newConn *websocket.Conn
	var err error

	as.mu.Lock()
	if as.reconnecting {
		as.mu.Unlock()
		return
	}
	as.reconnecting = true
	as.mu.Unlock()
	defer func() {
		as.mu.Lock()
		as.reconnecting = false
		as.mu.Unlock()
	}()

	if dialer == nil {
		// For server side, if dialer is nil, it means we rely on external UpdateWsConn to trigger reconnection
		log.Printf("[ARWS] Waiting for external UpdateWsConn to trigger reconnection...")
		return
	}
	for range 5 {
		log.Printf("[ARWS] Attempting to reconnect...")
		newConn, err = dialer()
		if err != nil {
			log.Printf("Failed to reconnect to server: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		success = true
		as.UpdateWsConn(newConn)
		break
	}
	if !success {
		log.Printf("[ARWS] Reconnection attempts failed for 5 times, giving up.")
		as.Close()
		as.cancel()
	}
}

func (as *ARWStream) UpdateWsConn(newConn *websocket.Conn) {
	as.mu.Lock()
	if as.Conn != nil {
		_ = as.Conn.Close()
	}
	as.Conn = newConn
	as.reconnected.Broadcast()
	as.mu.Unlock()

	go as.replayUnacked(newConn)
}

func (as *ARWStream) wsReadLoop() {
	for {
		as.mu.Lock()
		if as.isClosed {
			as.mu.Unlock()
			return
		}
		conn := as.Conn
		as.mu.Unlock()
		as.readmu.Lock()
		_, message, err := conn.ReadMessage()
		as.readmu.Unlock()
		if err != nil {
			as.mu.Lock()
			if !as.isClosed && as.Conn == conn {
				as.mu.Unlock()
				go as.reconnect(as.dialer)
				as.mu.Lock()
			}
			for !as.isClosed && as.Conn == conn {
				as.reconnected.Wait()
			}
			as.mu.Unlock()
			continue
		}
		// Validate frame
		if len(message) < FrameHeaderSize {
			log.Printf("[ARWS] Received invalid frame (too short), ignoring.")
			continue
		}
		// Parse the frame: Seq(8) + Ack(8) + Payload
		remoteSeq := binary.BigEndian.Uint64(message[0:8])
		remoteAck := binary.BigEndian.Uint64(message[8:16])
		payload := message[16:]

		as.mu.Lock()
		// Update remoteAck and remove acknowledged segments from sendQue
		if remoteAck > as.remoteAck {
			as.remoteAck = remoteAck
		}
		for len(as.sendQue) > 0 {
			head := as.sendQue[0]
			if head.seq+uint64(len(head.data)) > as.remoteAck {
				break
			}
			as.sendQue[0].data = nil
			as.sendQue = as.sendQue[1:]
		}
		// Parse incoming Seq and store in recvSeg
		if remoteSeq == as.expSeq {
			// If it's the expected Seq, write directly to recvBuf
			as.recvBuf.Write(payload)
			as.expSeq += uint64(len(payload))

			// Check if we have buffered segments that can now be written
			for {
				if seg, ok := as.recvSeg[as.expSeq]; ok {
					as.recvBuf.Write(seg)
					delete(as.recvSeg, as.expSeq)
					as.expSeq += uint64(len(seg))
				} else {
					break
				}
			}
		} else if remoteSeq > as.expSeq {
			// If it's a future Seq, buffer it in recvSeg
			as.recvSeg[remoteSeq] = payload
		} else {
			// Duplicate or old Seq, ignore
		}
		as.dataReady.Broadcast() // Wake up Read() when data is added to recvBuf
		as.mu.Unlock()
	}
}

func (as *ARWStream) Close() error {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.isClosed = true
	as.dataReady.Broadcast() // Wake up any waiting goroutines
	as.reconnected.Broadcast()
	if as.Conn != nil {
		return as.Conn.Close()
	}
	return nil
}

func (as *ARWStream) Read(p []byte) (n int, err error) {
	as.mu.Lock()
	defer as.mu.Unlock()
	for {
		if as.isClosed {
			return 0, io.EOF
		}

		// If there's data in the recvBuf
		if as.recvBuf.Len() > 0 {
			n, _ = as.recvBuf.Read(p)
			return n, nil
		}
		as.dataReady.Wait() // block until data arrives, reconnect, or close
	}
}

func (as *ARWStream) replayUnacked(conn *websocket.Conn) {
	as.mu.Lock()
	if as.isClosed || as.Conn != conn {
		as.mu.Unlock()
		return
	}
	ack := as.expSeq
	sendQue := make([]sendSegment, len(as.sendQue))
	for i, seg := range as.sendQue {
		buf := make([]byte, len(seg.data))
		copy(buf, seg.data)
		sendQue[i] = sendSegment{seq: seg.seq, data: buf}
	}
	as.mu.Unlock()

	for _, seg := range sendQue {
		// Construct the frame: Seq(8) + Ack(8) + Payload
		frame := make([]byte, FrameHeaderSize+len(seg.data))
		binary.BigEndian.PutUint64(frame[0:8], seg.seq)
		binary.BigEndian.PutUint64(frame[8:16], ack)
		copy(frame[16:], seg.data)

		as.writemu.Lock()
		err := conn.WriteMessage(websocket.BinaryMessage, frame)
		as.writemu.Unlock()
		if err != nil {
			log.Printf("[ARWS] Failed to replay unacked segment Seq=%d: %v", seg.seq, err)
			return
		}
	}
}

func (as *ARWStream) Write(p []byte) (n int, err error) {
	as.mu.Lock()
	if as.isClosed {
		as.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	seq := as.sendSeq
	as.sendSeq += uint64(len(p))
	// Copy p so caller can reuse the buffer after Write returns
	payload := make([]byte, len(p))
	copy(payload, p)
	as.sendQue = append(as.sendQue, sendSegment{seq: seq, data: payload})
	as.mu.Unlock()

	for {
		as.mu.Lock()
		if as.isClosed {
			as.mu.Unlock()
			return 0, io.ErrClosedPipe
		}
		conn := as.Conn
		exp := as.expSeq
		as.mu.Unlock()

		// Construct the frame: Seq(8) + Ack(8) + Payload
		frame := make([]byte, FrameHeaderSize+len(payload))
		binary.BigEndian.PutUint64(frame[0:8], seq)
		binary.BigEndian.PutUint64(frame[8:16], exp)
		copy(frame[16:], payload)

		as.writemu.Lock()
		err = conn.WriteMessage(websocket.BinaryMessage, frame)
		as.writemu.Unlock()
		if err == nil {
			return len(p), nil
		}

		// Write failed: wait for reconnect, then retry
		as.mu.Lock()
		if !as.isClosed && as.Conn == conn {
			as.mu.Unlock()
			go as.reconnect(as.dialer)
			as.mu.Lock()
		}
		for !as.isClosed && as.Conn == conn {
			as.reconnected.Wait()
		}
		as.mu.Unlock()
	}
}

func (a *ARWStream) SetDeadline(t time.Time) error {
	return a.Conn.UnderlyingConn().SetDeadline(t)
}
func (a *ARWStream) SetReadDeadline(t time.Time) error {
	return a.Conn.UnderlyingConn().SetReadDeadline(t)
}
func (a *ARWStream) SetWriteDeadline(t time.Time) error {
	return a.Conn.UnderlyingConn().SetWriteDeadline(t)
}
