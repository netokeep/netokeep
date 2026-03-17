package protocol

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
)

type SocConn struct {
	net.Conn
	host string
	port uint16
}

/*
PrependConn is used to replay the handshake data cosumed in the proxy listener
*/
type PrependConn struct {
	net.Conn
	buffer []byte
}

func (pc *PrependConn) Read(p []byte) (int, error) {
	if len(pc.buffer) > 0 {
		n := copy(p, pc.buffer)
		pc.buffer = pc.buffer[n:]
		return n, nil
	}
	return pc.Conn.Read(p)
}

type SocPattern uint8

const (
	UnknownPattern SocPattern = 0x00
	// Proxy pattern
	ProPattern SocPattern = 0x01
	// SSH request pattern
	SshPattern SocPattern = 0x02
)

/*
StartProxyListener create one http proxy server to receive local traffic from `listenPort`

StartProxy dose not defer the connection, the caller should handle the connection lifecycle in the handler function.
*/
func StartProxyListener(ctx context.Context, listenPort uint16, handler func(*SocConn)) error {
	var wg sync.WaitGroup
	lc := net.ListenConfig{}
	// Use local adress to avoid external connections
	la := fmt.Sprintf("127.0.0.1:%d", listenPort)
	ln, err := lc.Listen(ctx, "tcp", la)
	if err != nil {
		log.Fatalf("[LISTENER] Error in listening port %d: %v", listenPort, err)
	}
	log.Printf("🌐 HTTP proxy listener started at port %d", listenPort)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("[LISTENER] Proxy listener closed.")
				return
			}
			wg.Go(func() {
				// Handle the handshake of HTTP and return the conn with host and port
				request, err := http.ReadRequest(bufio.NewReader(conn))
				if err != nil {
					log.Printf("[LISTENER] Error in reading the request header: %v", err)
					conn.Close()
					return
				}
				host, portStr, err := net.SplitHostPort(request.Host)
				if err != nil {
					host = request.Host
					if request.Method == http.MethodConnect {
						portStr = "443"
					} else {
						portStr = "80"
					}
				}

				p, _ := strconv.ParseUint(portStr, 10, 16)
				port := uint16(p)

				// Construct SocConn
				socConn := &SocConn{
					Conn: conn,
					host: host,
					port: port,
				}

				// Handle CONNECT (usually used for HTTPS traffic)
				if request.Method == http.MethodConnect {
					_, err = conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
					if err != nil {
						log.Printf("[LISTENER] Error in CONNECT request handshake: %v", err)
						conn.Close()
						return
					}
				} else {
					request.RequestURI = ""
					var buffer bytes.Buffer
					request.Write(&buffer)

					socConn.Conn = &PrependConn{
						Conn:   conn,
						buffer: buffer.Bytes(),
					}
				}
				handler(socConn)
			})
		}
	}()

	<-ctx.Done()
	ln.Close()
	wg.Wait()
	return nil
}

/*
StartSshListener creates a listener for ssh connection, and the connection will be handled in the handler function.

StartSshListener does not defer the connection, the caller should handle the connection lifecycle in the handler function.
*/
func StartSshListener(ctx context.Context, listenPort uint16, handler func(*SocConn)) error {
	var wg sync.WaitGroup
	lc := net.ListenConfig{}
	// Use local adress to avoid external connections
	la := fmt.Sprintf("127.0.0.1:%d", listenPort)
	ln, err := lc.Listen(ctx, "tcp", la)
	if err != nil {
		log.Fatalf("[LISTENER] Error in listening port %d: %v", listenPort, err)
	}
	log.Printf("🌐 SSH listener started at port %d", listenPort)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("[LISTENER] SSH listener closed.")
				return
			}
			wg.Go(func() {
				handler(&SocConn{
					Conn: conn,
					host: "placeholder",
					port: 0,
				})
			})
		}
	}()

	<-ctx.Done()
	ln.Close()
	wg.Wait()
	return nil
}

/*
CreateSocHeader creates a header for simple protocol called soc.

The format is as follows:
  - 1 byte: header type (sock identification)
  - 2 bytes: destination port (big endian)
  - 1 byte: length of destination host
  - N bytes: destination host (domain or IP)
*/
func (sc *SocConn) CreateSocHeader(pattern SocPattern) []byte {
	header := make([]byte, 0, 1+2+1+len(sc.host))
	header = append(header, byte(pattern))
	header = binary.BigEndian.AppendUint16(header, sc.port)
	header = append(header, byte(len(sc.host)))
	header = append(header, []byte(sc.host)...)

	return header
}

/*
ParseSocHeader reads the soc header from the connection and returns the destination host and port.

Returns SocPattern, host, port and error.

The caller can use the pattern to determine how to handle the connection.
*/
func ParseSocHeader(conn net.Conn) (SocPattern, string, uint16, error) {
	var headerBuf [4]byte
	if _, err := io.ReadFull(conn, headerBuf[:]); err != nil {
		return UnknownPattern, "", 0, errors.New("failed to read soc header")
	}
	pattern := SocPattern(headerBuf[0])
	port := binary.BigEndian.Uint16(headerBuf[1:3])
	hostLen := int(headerBuf[3])

	hostBuf := make([]byte, hostLen)
	if _, err := io.ReadFull(conn, hostBuf); err != nil {
		return pattern, "", 0, errors.New("failed to read host name")
	}
	host := string(hostBuf)

	return pattern, host, port, nil
}
