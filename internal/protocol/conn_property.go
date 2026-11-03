package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

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

type Pattern uint8

const (
	UnknownPattern Pattern = 0x00
	ProxyPattern   Pattern = 0x01
	SSHPattern     Pattern = 0x02
)

type ConnProperty struct {
	Pattern Pattern
	Host    string
	Port    uint16
}

func ResolveConnProperty(conn net.Conn) (ConnProperty, error) {
	var headerBuf [4]byte
	if _, err := io.ReadFull(conn, headerBuf[:]); err != nil {
		return ConnProperty{Pattern: UnknownPattern}, fmt.Errorf("failed to read conn property header")
	}
	pattern := Pattern(headerBuf[0])
	port := binary.BigEndian.Uint16(headerBuf[1:3])
	hostLen := int(headerBuf[3])

	hostBuf := make([]byte, hostLen)
	if _, err := io.ReadFull(conn, hostBuf); err != nil {
		return ConnProperty{Pattern: UnknownPattern}, errors.New("failed to read host name")
	}
	host := string(hostBuf)

	return ConnProperty{
		Pattern: pattern,
		Host:    host,
		Port:    port,
	}, nil
}

func InsertConnProperty(conn net.Conn, property ConnProperty) (net.Conn, error) {
	if len(property.Host) > 255 {
		return nil, fmt.Errorf("host too long: %s", property.Host)
	}
	headerBuf := make([]byte, 0, 4+len(property.Host))
	headerBuf = append(headerBuf, byte(property.Pattern))
	headerBuf = binary.BigEndian.AppendUint16(headerBuf, property.Port)
	headerBuf = append(headerBuf, byte(len(property.Host)))
	headerBuf = append(headerBuf, []byte(property.Host)...)

	return &PrependConn{Conn: conn, Buffer: headerBuf}, nil
}
