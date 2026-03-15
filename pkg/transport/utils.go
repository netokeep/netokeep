package transport

import (
	"io"
	"net"
	"sync"
)

var bufferPool = sync.Pool{
    New: func() any {
        return make([]byte, 1*1024*1024)
    },
}

/*
Relay helps to forward data between two net.Conn connections in both directions.
*/
func Relay(left net.Conn, right net.Conn) {
    var wg sync.WaitGroup
    wg.Add(2)

    copyDir := func(dst net.Conn, src net.Conn) {
        defer wg.Done()
        defer dst.Close()

        buf := bufferPool.Get().([]byte)
        defer bufferPool.Put(buf)

        _, _ = io.CopyBuffer(dst, src, buf)
    }

    go copyDir(right, left)
    go copyDir(left, right)

    wg.Wait()
}
