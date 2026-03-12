package protocol

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
)

type RoutePattern int

const (
	SocksHeader RoutePattern = iota
	SshHeader
)

const (
	socByte = 0x02
	sshByte = 0x01
)

/*
CreateSocHeader creates a header for simple protocol called soc.

The format is as follows:
  - 1 byte: header type (sock identification)
  - 2 bytes: destination port (big endian)
  - 1 byte: length of destination host
  - N bytes: destination host (domain or IP)
*/
func CreateSocHeader(conn *SocksConn) []byte {
	host := conn.Host()
	port := conn.Port()

	header := make([]byte, 0, 1+2+1+len(host))
	header = append(header, socByte)
	header = binary.BigEndian.AppendUint16(header, port)
	header = append(header, byte(len(host)))
	header = append(header, []byte(host)...)

	return header
}

func CreateSshHeader(conn net.Conn) []byte {
	header := make([]byte, 1)
	header[0] = sshByte
	return header
}

func MatchHeader(conn net.Conn) RoutePattern {
	buf := make([]byte, 1)
	n, err := io.ReadFull(conn, buf)
	if err != nil || n == 0 {
		return -1
	}

	switch buf[0] {
	case socByte:
		return SocksHeader
	case sshByte:
		return SshHeader
	default:
		return -1
	}
}

/*
ParseSocHeader reads the soc header from the connection and returns the destination host and port.
*/
func ParseSocHeader(conn net.Conn) (string, uint16, error) {
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return "", 0, errors.New("failed to read host port")
	}
	port := binary.BigEndian.Uint16(portBuf)

	lenBuf := make([]byte, 1)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return "", 0, errors.New("failed to read the length of host")
	}
	hostLen := int(lenBuf[0])

	hostBuf := make([]byte, hostLen)
	if _, err := io.ReadFull(conn, hostBuf); err != nil {
		return "", 0, errors.New("failed to read host name")
	}
	host := string(hostBuf)

	return host, port, nil
}
