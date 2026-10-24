package hub

// 21 号票回归：conn.send × conn.close 的数据竞争。
//
// 缺陷：close() 曾同时关闭 closed 信号与 outbox 数据通道——send 的 select
// 发送分支与 closechan 之间无 happens-before，-race 必报 DATA RACE，交错
// 不利时 send on closed channel panic 直接崩掉整个测试进程。生产触发结构：
// readLoop 断连收尾的 close（hub 锁外）× Bot 定时器 goroutine 的广播 send
// （hub 锁内）——hub 互斥锁无法排除该交错，因为 close 在锁外调用。
//
// 本测试把该交错压成纯并发压测（500 轮 × 4 发送者对 1 个 close，缓冲 1
// 强制其余发送者走 closed 分支）：修复前 -race 下必红；修复后（outbox
// 永不关闭，全部通道操作内部自同步）绿。
import (
	"sync"
	"testing"
)

func TestConnConcurrentSendClose(t *testing.T) {
	for i := 0; i < 500; i++ {
		c := &conn{outbox: make(chan []byte, 1), closed: make(chan struct{})}
		var wg sync.WaitGroup
		for j := 0; j < 4; j++ {
			wg.Add(1)
			go func() { defer wg.Done(); c.send([]byte("x")) }()
		}
		c.close()
		wg.Wait()
	}
}
