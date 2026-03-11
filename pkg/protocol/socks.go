package protocol

import (
	"errors"
	"net"
)

type SocksConn struct {
	net.Conn
	host string
	port uint16
}

func (sc *SocksConn) Host() string {
	return sc.host
}

func (sc *SocksConn) Port() uint16 {
	return sc.port
}

/*
GetSocksHandler resolves the socks5 handshake and skip the header, then return the connection for further processing.
*/
func GetSocksHandler(conn net.Conn) (*SocksConn, error) {
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	// Only support socks5, and no authentication.
	if n < 3 || buf[0] != 0x05 {
		return nil, errors.New("invalid socks5 handshake")
	}
	conn.Write([]byte{0x05, 0x00})

	// Read the request header, and skip it.
	n, err = conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if n < 10 || buf[0] != 0x05 {
		return nil, errors.New("invalid socks5 request")
	}

	var host string
	switch buf[3] {
	case 0x01: // IPv4
		host = net.IP(buf[4:8]).String()
	case 0x03: // Domain name
		host = string(buf[5 : 5+buf[4]])
	case 0x04: // IPv6
		host = net.IP(buf[4:20]).String()
	default:
		return nil, errors.New("invalid socks5 address type")
	}
	port := uint16(buf[n-2])<<8 | uint16(buf[n-1])
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	return &SocksConn{
		Conn: conn,
		host: host,
		port: port,
	}, nil
}
