package protocol

import (
	"encoding/binary"
	"net"
)

type RoutePattern int

const (
	SocksHeader RoutePattern = iota
	SshHeader
)

const (
	socksByte = 0x02
	sshByte   = 0x01
)

func CreateSocksHeader(conn *SocksConn) []byte {
	host := conn.Host()
	port := conn.Port()

	header := make([]byte, 0, 1+2+1+len(host))
	header = append(header, socksByte)
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
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return -1
	}

	switch buf[0] {
	case socksByte:
		return SocksHeader
	case sshByte:
		return SshHeader
	default:
		return -1
	}
}
