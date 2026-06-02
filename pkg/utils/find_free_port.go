package utils

import (
	"net"
)

func FindFreePort() (uint16, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return uint16(addr.Port), nil
}
