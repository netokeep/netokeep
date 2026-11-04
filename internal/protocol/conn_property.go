package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

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

func EncodeConnProperty(property ConnProperty) ([]byte, error) {
	if len(property.Host) > 255 {
		return nil, fmt.Errorf("host too long: %s", property.Host)
	}
	headerBuf := make([]byte, 0, 4+len(property.Host))
	headerBuf = append(headerBuf, byte(property.Pattern))
	headerBuf = binary.BigEndian.AppendUint16(headerBuf, property.Port)
	headerBuf = append(headerBuf, byte(len(property.Host)))
	headerBuf = append(headerBuf, []byte(property.Host)...)

	return headerBuf, nil
}
