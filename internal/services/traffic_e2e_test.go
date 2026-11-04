package services

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"netokeep/internal/protocol"
	"netokeep/internal/sessions"
	"netokeep/internal/utils"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// startEcho starts a TCP echo server standing in for sshd or a proxy target.
func startEcho(t *testing.T) uint16 {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { defer c.Close(); io.Copy(c, c) }()
		}
	}()
	return uint16(ln.Addr().(*net.TCPAddr).Port)
}

func echoRoundTrip(t *testing.T, c net.Conn, msg string) {
	t.Helper()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := c.Write([]byte(msg)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(c, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != msg {
		t.Fatalf("got %q, want %q", buf, msg)
	}
}

func TestTrafficEndToEnd(t *testing.T) {
	echoPort := startEcho(t)
	serverPort, err := utils.FindFreePort()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	serverMgr, clientMgr := &sessions.SessionManager{}, &sessions.SessionManager{}

	serverReady, clientReady := make(chan struct{}), make(chan struct{})
	serverDone, clientDone := make(chan error, 1), make(chan error, 1)
	go func() {
		serverDone <- StartTrafficServer(ctx, cancel, serverReady, TrafficServerConfig{
			Manager: serverMgr, Port: serverPort, SSHPort: echoPort,
		})
	}()
	<-serverReady
	go func() {
		clientDone <- StartTrafficClient(ctx, cancel, clientReady, TrafficClientConf{
			Manager:        clientMgr,
			RemoteAddr:     fmt.Sprintf("ws://127.0.0.1:%d", serverPort),
			ForwardTraffic: true,
		})
	}()
	select {
	case <-clientReady:
	case err := <-clientDone:
		t.Fatalf("client exited early: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("client not ready")
	}

	// The server registers the session in its handler goroutine; wait for it.
	deadline := time.Now().Add(2 * time.Second)
	for {
		if s, err := serverMgr.OpenStream(protocol.ConnProperty{Pattern: protocol.UnknownPattern}, true); err == nil {
			s.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server never registered the session")
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Run("ssh: client -> server -> sshd", func(t *testing.T) {
		local, remote := net.Pipe()
		go func() {
			if err := clientMgr.Traffic2Session(remote, protocol.ConnProperty{Pattern: protocol.SSHPattern}); err != nil {
				remote.Close()
			}
		}()
		defer local.Close()
		echoRoundTrip(t, local, "SSH-2.0-test\r\n")
	})

	t.Run("proxy: server -> client -> target", func(t *testing.T) {
		stream, err := serverMgr.OpenStream(protocol.ConnProperty{
			Pattern: protocol.ProxyPattern, Host: "127.0.0.1", Port: echoPort,
		}, true)
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		echoRoundTrip(t, stream, "GET / HTTP/1.1\r\n\r\n")
	})

	t.Run("handshake rejections", func(t *testing.T) {
		url := fmt.Sprintf("ws://127.0.0.1:%d", serverPort)
		status := func(id string, resume bool) int {
			h := http.Header{}
			h.Set("NK-Session-ID", id)
			h.Set("NK-Resume", strconv.FormatBool(resume))
			ws, resp, _ := websocket.DefaultDialer.Dial(url, h)
			if ws != nil {
				ws.Close()
			}
			if resp == nil {
				return 0
			}
			return resp.StatusCode
		}
		if got := status("unknown-id", true); got != http.StatusGone {
			t.Errorf("resume unknown session: got %d, want 410", got)
		}
		// Open a raw fresh session, then try to open another with the same id.
		h := http.Header{}
		h.Set("NK-Session-ID", "dup-id")
		first, _, err := websocket.DefaultDialer.Dial(url, h)
		if err != nil {
			t.Fatal(err)
		}
		defer first.Close()
		for i := 0; ; i++ {
			if _, ok := serverMgr.Get("dup-id"); ok {
				break
			}
			if i > 100 {
				t.Fatal("dup-id never registered")
			}
			time.Sleep(20 * time.Millisecond)
		}
		if got := status("dup-id", false); got != http.StatusConflict {
			t.Errorf("reuse live id: got %d, want 409", got)
		}
	})

	cancel()
	for name, ch := range map[string]chan error{"server": serverDone, "client": clientDone} {
		select {
		case err := <-ch:
			if err != nil {
				t.Errorf("%s returned %v after cancel", name, err)
			}
		case <-time.After(10 * time.Second):
			t.Errorf("%s did not shut down", name)
		}
	}
}
