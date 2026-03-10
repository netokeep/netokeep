package transport

import (
	"io"
	"net"
	"sync"
)

// Relay 在两个 net.Conn 之间进行双向数据转发
func Relay(left, right net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	// 1. 从左边读，往右边写
	go func() {
		defer wg.Done()
		_, _ = io.Copy(right, left)
		// 一旦这边断了，通知另一边也别读了
		right.Close()
	}()

	// 2. 从右边读，往左边写
	go func() {
		defer wg.Done()
		_, _ = io.Copy(left, right)
		// 一旦这边断了，通知另一边也别读了
		left.Close()
	}()

	// 等待两个方向都结束
	wg.Wait()
}
