package protocol

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
)

type SocPattern uint8

const (
	UnknownPattern SocPattern = 0x00
	// Proxy pattern
	ProPattern SocPattern = 0x01
	// SSH request pattern
	SshPattern SocPattern = 0x02
)

type SocConn struct {
	net.Conn
	Host string
	Port uint16
}

/*
PrependConn is used to replay the handshake data cosumed in the proxy listener
*/
type PrependConn struct {
	net.Conn
	Buffer []byte
}

func (pc *PrependConn) Read(p []byte) (int, error) {
	if len(pc.Buffer) > 0 {
		n := copy(p, pc.Buffer)
		pc.Buffer = pc.Buffer[n:]
		return n, nil
	}
	return pc.Conn.Read(p)
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
	header := make([]byte, 0, 1+2+1+len(sc.Host))
	header = append(header, byte(pattern))
	header = binary.BigEndian.AppendUint16(header, sc.Port)
	header = append(header, byte(len(sc.Host)))
	header = append(header, []byte(sc.Host)...)

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
