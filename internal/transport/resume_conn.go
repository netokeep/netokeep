package transport

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	pingInterval     = 5 * time.Second
	idleTimeout      = 15 * time.Second // no pong or data for this long: conn is dead
	writeTimeout     = 10 * time.Second
	handshakeTimeout = 3 * time.Second
	reconnectBudget  = 5 * time.Second // client: total time allowed for redialing
	// server: the client may notice the drop up to idleTimeout later, then redials within its budget
	resumeGrace = idleTimeout + reconnectBudget
	ackEvery    = 256 * 1024 // send a standalone ack after receiving this many bytes
	ackSize     = 8
)

var errProtocol = errors.New("resumable conn: protocol violation")

// ErrSessionGone is returned by a dialer when the server no longer knows the session
// (e.g. HTTP 410). Redialing stops immediately instead of burning the budget.
var ErrSessionGone = errors.New("resumable conn: session gone on server")

/*
ResumableConn is a byte stream over a WebSocket that survives reconnects.

Every frame is: ack (8 bytes, total bytes this side has received) + payload.
Sequence numbers are implicit: after a (re)connect both sides first exchange
their ack, so the sender replays exactly what the peer is missing.

Read and Write must each be called from a single goroutine (yamux does this).
*/
type ResumableConn struct {
	ctx    context.Context
	cancel context.CancelFunc
	dial   func(resume bool) (*websocket.Conn, error) // nil on the server side
	lost   chan struct{}                              // wakes the supervisor to recover

	mu        sync.Mutex
	ws        *websocket.Conn // nil while reconnecting or closed
	changed   chan struct{}   // closed and replaced when a conn is attached or rc closes
	recvSeq   uint64          // bytes received in order, i.e. the ack we send
	ackSent   uint64          // highest ack already sent to the peer
	unacked   [][]byte        // payloads sent but not acked yet
	unackedAt uint64          // stream offset of unacked[0]
	sendSeq   uint64          // total bytes written

	writeMu   sync.Mutex // one WebSocket writer at a time; also orders replay before new writes
	pending   []byte     // received but not yet returned by Read; reader goroutine only
	closeOnce sync.Once
}

// NewResumableConn wraps an established WebSocket. Pass dialer on the client
// (used to redial with resume=true); pass nil on the server and call UpdateWsConn
// when the client comes back.
func NewResumableConn(ctx context.Context, conn *websocket.Conn, dialer func(resume bool) (*websocket.Conn, error)) *ResumableConn {
	ctx, cancel := context.WithCancel(ctx)
	c := &ResumableConn{
		ctx:     ctx,
		cancel:  cancel,
		dial:    dialer,
		lost:    make(chan struct{}, 1),
		changed: make(chan struct{}),
		ws:      conn,
	}
	setupHeartbeat(conn)
	go c.heartbeat(conn)
	go c.supervise()
	context.AfterFunc(ctx, func() { c.Close() })
	return c
}

// UpdateWsConn attaches a new WebSocket from a reconnecting client (server side).
func (c *ResumableConn) UpdateWsConn(ws *websocket.Conn) error {
	return c.attach(ws)
}

func (c *ResumableConn) Read(p []byte) (int, error) {
	for len(c.pending) == 0 {
		ws, err := c.current()
		if err != nil {
			return 0, io.EOF
		}
		_, msg, err := ws.ReadMessage()
		if err != nil {
			c.broken(ws)
			continue // current() waits for the next conn
		}
		if len(msg) < ackSize {
			c.Close()
			return 0, errProtocol
		}
		ws.SetReadDeadline(time.Now().Add(idleTimeout))

		c.mu.Lock()
		if c.ws != ws {
			// Arrived on a conn that has been replaced; the peer will replay it.
			c.mu.Unlock()
			continue
		}
		if err := c.ackLocked(binary.BigEndian.Uint64(msg)); err != nil {
			c.mu.Unlock()
			c.Close()
			return 0, err
		}
		c.recvSeq += uint64(len(msg) - ackSize)
		needAck := c.recvSeq-c.ackSent >= ackEvery
		c.mu.Unlock()

		c.pending = msg[ackSize:]
		// Never block the reader on writeMu: if a write is in flight, it carries the ack anyway.
		if needAck && c.writeMu.TryLock() {
			err := c.writeFrame(ws, nil)
			c.writeMu.Unlock()
			if err != nil {
				c.broken(ws)
			}
		}
	}
	n := copy(p, c.pending)
	c.pending = c.pending[n:]
	return n, nil
}

func (c *ResumableConn) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	data := bytes.Clone(p) // the caller may reuse p, but we keep it until acked

	c.writeMu.Lock()
	c.mu.Lock()
	if c.ctx.Err() != nil {
		c.mu.Unlock()
		c.writeMu.Unlock()
		return 0, net.ErrClosed
	}
	c.unacked = append(c.unacked, data)
	c.sendSeq += uint64(len(data))
	ws := c.ws
	c.mu.Unlock()

	var err error
	if ws != nil {
		err = c.writeFrame(ws, data)
	}
	c.writeMu.Unlock()
	if ws != nil && err == nil {
		return len(p), nil
	}
	if ws != nil {
		c.broken(ws)
	}
	// data stays in unacked and is replayed when the next conn is attached.
	if _, err := c.current(); err != nil {
		return 0, net.ErrClosed
	}
	return len(p), nil
}

func (c *ResumableConn) Close() error {
	c.closeOnce.Do(func() {
		c.cancel()
		c.mu.Lock()
		ws := c.ws
		c.ws = nil
		c.unacked = nil
		close(c.changed)
		c.mu.Unlock()
		if ws != nil {
			ws.Close()
		}
	})
	return nil
}

// current returns the live conn, waiting while a reconnect is in progress.
func (c *ResumableConn) current() (*websocket.Conn, error) {
	for {
		if c.ctx.Err() != nil {
			return nil, net.ErrClosed
		}
		c.mu.Lock()
		ws, ch := c.ws, c.changed
		c.mu.Unlock()
		if ws != nil {
			return ws, nil
		}
		select {
		case <-ch:
		case <-c.ctx.Done():
			return nil, net.ErrClosed
		}
	}
}

// broken marks ws as dead and wakes the supervisor. Only the first caller for a given ws acts.
func (c *ResumableConn) broken(ws *websocket.Conn) {
	c.mu.Lock()
	if c.ws != ws {
		c.mu.Unlock()
		return
	}
	c.ws = nil
	c.mu.Unlock()
	ws.Close()
	c.signalLost()
}

func (c *ResumableConn) signalLost() {
	select {
	case c.lost <- struct{}{}:
	default:
	}
}

// detach drops the current conn (if any) and returns our receive position.
// While ws is nil the reader discards everything, so recvSeq cannot move.
func (c *ResumableConn) detach() uint64 {
	c.mu.Lock()
	old := c.ws
	c.ws = nil
	ack := c.recvSeq
	c.mu.Unlock()
	if old != nil {
		// The peer reconnected before we noticed the old conn was dead.
		old.Close()
		c.signalLost()
	}
	return ack
}

// attach performs the resume handshake on ws, replays what the peer missed and makes ws current.
func (c *ResumableConn) attach(ws *websocket.Conn) error {
	// Close the old conn first so a write stuck on it releases writeMu.
	c.detach()
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	myAck := c.detach() // again under writeMu, in case another attach slipped in

	peerAck, err := handshake(ws, myAck)
	if err != nil {
		ws.Close()
		return fmt.Errorf("resume handshake: %w", err)
	}

	c.mu.Lock()
	if c.ctx.Err() != nil {
		c.mu.Unlock()
		ws.Close()
		return net.ErrClosed
	}
	if err := c.ackLocked(peerAck); err != nil || c.unackedAt != peerAck {
		c.mu.Unlock()
		ws.Close()
		c.Close()
		return fmt.Errorf("%w: peer ack %d, unacked from %d", errProtocol, peerAck, c.unackedAt)
	}
	replay := slices.Clone(c.unacked)
	c.ackSent = myAck
	setupHeartbeat(ws)
	c.ws = ws
	close(c.changed)
	c.changed = make(chan struct{})
	c.mu.Unlock()

	go c.heartbeat(ws)
	for _, p := range replay {
		if err := c.writeFrame(ws, p); err != nil {
			c.broken(ws)
			break
		}
	}
	return nil
}

// ackLocked drops payloads the peer has confirmed. Caller holds mu.
func (c *ResumableConn) ackLocked(ack uint64) error {
	if ack > c.sendSeq {
		return fmt.Errorf("%w: ack %d beyond sent %d", errProtocol, ack, c.sendSeq)
	}
	for len(c.unacked) > 0 && c.unackedAt+uint64(len(c.unacked[0])) <= ack {
		c.unackedAt += uint64(len(c.unacked[0]))
		c.unacked[0] = nil
		c.unacked = c.unacked[1:]
	}
	return nil
}

// writeFrame sends ack + payload as one WebSocket message. Caller holds writeMu.
func (c *ResumableConn) writeFrame(ws *websocket.Conn, payload []byte) error {
	c.mu.Lock()
	ack := c.recvSeq
	c.mu.Unlock()

	var hdr [ackSize]byte
	binary.BigEndian.PutUint64(hdr[:], ack)
	ws.SetWriteDeadline(time.Now().Add(writeTimeout))
	w, err := ws.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}
	w.Write(hdr[:])
	w.Write(payload)
	if err := w.Close(); err != nil {
		return err
	}

	c.mu.Lock()
	c.ackSent = max(c.ackSent, ack)
	c.mu.Unlock()
	return nil
}

// handshake exchanges receive positions on a fresh conn.
func handshake(ws *websocket.Conn, myAck uint64) (uint64, error) {
	deadline := time.Now().Add(handshakeTimeout)
	ws.SetWriteDeadline(deadline)
	ws.SetReadDeadline(deadline)

	var hdr [ackSize]byte
	binary.BigEndian.PutUint64(hdr[:], myAck)
	if err := ws.WriteMessage(websocket.BinaryMessage, hdr[:]); err != nil {
		return 0, err
	}
	_, msg, err := ws.ReadMessage()
	if err != nil {
		return 0, err
	}
	if len(msg) != ackSize {
		return 0, errProtocol
	}
	return binary.BigEndian.Uint64(msg), nil
}

// setupHeartbeat arms the idle deadline; every pong pushes it back.
func setupHeartbeat(ws *websocket.Conn) {
	ws.SetReadDeadline(time.Now().Add(idleTimeout))
	ws.SetPongHandler(func(string) error {
		return ws.SetReadDeadline(time.Now().Add(idleTimeout))
	})
}

// heartbeat pings ws and flushes pending acks until ws is replaced or rc closes.
func (c *ResumableConn) heartbeat(ws *websocket.Conn) {
	t := time.NewTicker(pingInterval)
	defer t.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-t.C:
		}
		c.mu.Lock()
		alive := c.ws == ws
		behind := c.recvSeq > c.ackSent
		c.mu.Unlock()
		if !alive {
			return
		}
		if err := ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeTimeout)); err != nil {
			c.broken(ws)
			return
		}
		if behind {
			c.writeMu.Lock()
			err := c.writeFrame(ws, nil)
			c.writeMu.Unlock()
			if err != nil {
				c.broken(ws)
				return
			}
		}
	}
}

// supervise is the only goroutine that drives recovery.
func (c *ResumableConn) supervise() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.lost:
		}
		var ok bool
		if c.dial != nil {
			ok = c.redial()
		} else {
			ok = c.awaitResume()
		}
		if !ok {
			log.Printf("[resume] connection lost for good, closing")
			c.Close()
			return
		}
	}
}

func (c *ResumableConn) hasConn() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ws != nil
}

// redial reconnects with backoff 1s, 2s, 4s (+ jitter) within reconnectBudget.
func (c *ResumableConn) redial() bool {
	deadline := time.Now().Add(reconnectBudget)
	for attempt := 0; ; attempt++ {
		if c.hasConn() {
			return true
		}
		log.Printf("[resume] reconnecting (attempt %d)", attempt+1)
		ws, err := c.dial(true)
		if err == nil {
			if err = c.attach(ws); err == nil {
				log.Printf("[resume] reconnected")
				return true
			}
		}
		log.Printf("[resume] reconnect failed: %v", err)
		if errors.Is(err, ErrSessionGone) {
			return false
		}

		wait := time.Second<<attempt + rand.N(500*time.Millisecond)
		if time.Now().Add(wait).After(deadline) {
			return false
		}
		select {
		case <-time.After(wait):
		case <-c.ctx.Done():
			return false
		}
	}
}

// awaitResume waits for the client to come back through UpdateWsConn.
func (c *ResumableConn) awaitResume() bool {
	timer := time.NewTimer(resumeGrace)
	defer timer.Stop()
	for {
		c.mu.Lock()
		ws, ch := c.ws, c.changed
		c.mu.Unlock()
		if ws != nil {
			return true
		}
		select {
		case <-ch:
		case <-timer.C:
			return false
		case <-c.ctx.Done():
			return false
		}
	}
}
