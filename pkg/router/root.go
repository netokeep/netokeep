package router

import "net"

func HandleLogicStream(stream net.Conn, sshPort uint16) {
	print(stream)
}
