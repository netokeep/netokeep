package transport

import (
	"io"
	"net"
	"sync"
)

/*
Relay helps to forward data between two net.Conn connections in both directions.
*/
func Relay(left net.Conn, right net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	// Read the left connection and write to the right connection
	go func() {
		defer wg.Done()
		_, _ = io.Copy(right, left)
		right.Close()
	}()

	// Read the right connection and write to the left connection
	go func() {
		defer wg.Done()
		_, _ = io.Copy(left, right)
		left.Close()
	}()

	// Wait for both directions to finish
	wg.Wait()
}
