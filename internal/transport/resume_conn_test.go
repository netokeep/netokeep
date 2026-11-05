package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// newPair starts a server and returns a connected client/server ResumableConn pair.
func newPair(t *testing.T) (client, server *ResumableConn) {
	t.Helper()
	var srvConn atomic.Pointer[ResumableConn]
	ready := make(chan struct{})
	up := websocket.Upgrader{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		if r.Header.Get("NK-Resume") == "true" {
			if err := srvConn.Load().UpdateWsConn(ws); err != nil {
				t.Logf("server resume: %v", err)
			}
			return
		}
		srvConn.Store(NewResumableConn(context.Background(), ws, nil))
		close(ready)
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	dial := func(resume bool) (*websocket.Conn, error) {
		h := http.Header{}
		h.Set("NK-Resume", strconv.FormatBool(resume))
		ws, _, err := websocket.DefaultDialer.Dial(url, h)
		return ws, err
	}
	ws, err := dial(false)
	if err != nil {
		t.Fatal(err)
	}
	client = NewResumableConn(context.Background(), ws, dial)
	<-ready
	server = srvConn.Load()
	t.Cleanup(func() { client.Close(); server.Close() })
	return client, server
}

// dropTCP kills the client's current TCP conn without any WebSocket close frame.
func dropTCP(c *ResumableConn) {
	c.mu.Lock()
	ws := c.ws
	c.mu.Unlock()
	if ws != nil {
		ws.UnderlyingConn().Close()
	}
}

// send writes data in small chunks with pauses so drops land mid-transfer.
func send(c *ResumableConn, data []byte, errc chan<- error) {
	const chunk = 8 << 10
	for off := 0; off < len(data); off += chunk {
		if _, err := c.Write(data[off:min(off+chunk, len(data))]); err != nil {
			errc <- err
			return
		}
		time.Sleep(time.Millisecond)
	}
	errc <- nil
}

func TestResumableConnSurvivesDrops(t *testing.T) {
	client, server := newPair(t)

	const total = 2 << 20
	up, down := make([]byte, total), make([]byte, total)
	rand.Read(up)
	rand.Read(down)

	// Both directions at once, while the TCP conn is killed repeatedly.
	errc := make(chan error, 2)
	go send(client, up, errc)
	go send(server, down, errc)

	stop := make(chan struct{})
	go func() {
		for range 5 {
			select {
			case <-stop:
				return
			case <-time.After(80 * time.Millisecond):
				dropTCP(client)
			}
		}
	}()
	defer close(stop)

	type result struct {
		data []byte
		err  error
	}
	readAll := func(c *ResumableConn) <-chan result {
		ch := make(chan result, 1)
		go func() {
			buf := make([]byte, total)
			_, err := io.ReadFull(c, buf)
			ch <- result{buf, err}
		}()
		return ch
	}
	gotUp, gotDown := readAll(server), readAll(client)

	timeout := time.After(20 * time.Second)
	for _, tc := range []struct {
		name string
		ch   <-chan result
		want []byte
	}{{"client->server", gotUp, up}, {"server->client", gotDown, down}} {
		select {
		case r := <-tc.ch:
			if r.err != nil {
				t.Fatalf("%s: read: %v", tc.name, r.err)
			}
			if !bytes.Equal(r.data, tc.want) {
				t.Fatalf("%s: data corrupted", tc.name)
			}
		case <-timeout:
			t.Fatalf("%s: timed out", tc.name)
		}
	}
	for range 2 {
		if err := <-errc; err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	// After the dust settles, acks should have drained the replay buffers.
	time.Sleep(pingInterval + time.Second)
	for name, c := range map[string]*ResumableConn{"client": client, "server": server} {
		c.mu.Lock()
		n := len(c.unacked)
		c.mu.Unlock()
		if n != 0 {
			t.Errorf("%s still holds %d unacked segments", name, n)
		}
	}
}

func TestResumableConnCloseUnblocks(t *testing.T) {
	client, _ := newPair(t)
	done := make(chan error, 1)
	go func() {
		_, err := client.Read(make([]byte, 1))
		done <- err
	}()
	time.Sleep(50 * time.Millisecond)
	client.Close()
	select {
	case err := <-done:
		if err != io.EOF {
			t.Fatalf("want io.EOF, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not return after Close")
	}
	if _, err := client.Write([]byte("x")); err == nil {
		t.Fatal("Write after Close should fail")
	}
}

func TestResumableConnErrReportsSessionGone(t *testing.T) {
	client, _ := newPair(t)
	// From now on the "server" answers every resume with 410.
	client.dial = func(bool) (*websocket.Conn, error) { return nil, ErrSessionGone }
	if client.Err() != nil {
		t.Fatalf("Err before close: %v", client.Err())
	}
	dropTCP(client)

	done := make(chan error, 1)
	go func() {
		_, err := client.Read(make([]byte, 1))
		done <- err
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Read did not return after the session was gone")
	}
	if !errors.Is(client.Err(), ErrSessionGone) {
		t.Fatalf("Err() = %v, want ErrSessionGone", client.Err())
	}
}
