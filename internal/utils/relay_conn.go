package utils

import (
	"io"
	"net"
	"sync"
)

func Relay(left net.Conn, right net.Conn) {
	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Go(func() {
		_, _ = io.Copy(left, right)
		left.Close()
	})

	wg.Go(func() {
		_, _ = io.Copy(right, left)
		right.Close()
	})
}
