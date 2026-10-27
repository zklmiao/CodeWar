package hub

// 31 号票：hub 连接卫生——死链（不回 pong）超时回收、pong 保活续窗、并发上限
// 拒绝与释放后再入、握手期（不发 join）回收。
//
// 回收/保活走真实墙钟与真实 socket（SetReadDeadline 无法注入假钟），回收与
// 保活两个慢观察合并进一个 t.Parallel 测试与其他用例重叠执行。

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/match"

	"github.com/gorilla/websocket"
)

// enablePong：测试客户端的 pong 保活。gorilla 默认不自动回 pong（生产客户端是
// 浏览器，协议栈自动回）——读窗口（pongTimeout）下的长对局测试必须显式回 pong
// 扮演真实浏览器，否则会被服务端判死。只改 harness 行为，不改任何断言语义。
func enablePong(ws *websocket.Conn) {
	ws.SetPingHandler(func(appData string) error {
		return ws.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(5*time.Second))
	})
}

// TestDeadConnRecycledAndPongKeepsAlive：两条接入成功的连接——
//   - dead：不回 pong 且不上行（断网/休眠死链）→ pongTimeout 内被判死回收
//     （conns 名额释放，断线重连不再被死名额锁住）；
//   - alive：显式回 pong（浏览器行为）→ 跨过 pongTimeout 后仍存活（保活生效）。
//
// 后半断言同时钉住"pong 重置 deadline"本身：若 pong 不续窗（保活被删），
// alive 会在自己的读窗口到期时被误杀。两个观察共用一次墙钟等待（≈pongTimeout）。
func TestDeadConnRecycledAndPongKeepsAlive(t *testing.T) {
	t.Parallel()
	// A：死链——PvE 单连接（idle Bot 保持对局稳定），不设 ping handler
	hA := New(passJudge{}, slog.Default(), nil)
	hA.botClock = idleClock{}
	srvA := httptest.NewServer(http.HandlerFunc(hA.ServeWS))
	defer srvA.Close()
	dead := dialJoin(t, srvA.URL, &matchpb.JoinRequest{Player: "p1", Mode: "pve"})
	readSnapshot(t, dead)

	// B：保活——PvP p1 匿名 WAITING（纯静默等对手，最依赖 pong 保活的场景）。
	// pong 由 ping handler 在 ReadMessage 的控制帧处理中触发——客户端必须
	// 保持读取（浏览器 onmessage 语义），静置连接读不到 ping 也不会回 pong。
	hB := New(passJudge{}, slog.Default(), nil)
	srvB := httptest.NewServer(http.HandlerFunc(hB.ServeWS))
	defer srvB.Close()
	alive := dialJoin(t, srvB.URL, &matchpb.JoinRequest{Player: "p1"})
	readSnapshot(t, alive)
	enablePong(alive)
	go func() {
		// readSnapshot 曾设 5s 读 deadline：保活读循环必须清掉（零值 = 永不
		// 超时），否则 T0+5s 后客户端退出读循环、ping 无人处理、pong 断供。
		_ = alive.SetReadDeadline(time.Time{})
		for {
			if _, _, err := alive.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// 死链应在 pongTimeout（+调度余量）内回收
	deadline := time.Now().Add(pongTimeout + 10*time.Second)
	for {
		hA.mu.Lock()
		empty := len(hA.conns) == 0
		hA.mu.Unlock()
		if empty {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("不回 pong 的死连接应在 pongTimeout 内被回收")
		}
		time.Sleep(100 * time.Millisecond)
	}
	// 保活：alive 已跨过自己的首个读窗口（接入时刻 + pongTimeout）仍不被回收
	hB.mu.Lock()
	n := len(hB.conns)
	hB.mu.Unlock()
	if n != 1 {
		t.Fatalf("回 pong 的连接不得被回收（保活失效），conns=%d", n)
	}
	alive.Close()
}

// TestJoinHandshakeTimeout：升级后不发 join 的连接（占坑/半开）在 joinTimeout
// 内被关回收，而不是无限期占用 goroutine 与读缓冲。慢测试，t.Parallel。
func TestJoinHandshakeTimeout(t *testing.T) {
	t.Parallel()
	h := New(passJudge{}, slog.Default(), nil)
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatalf("拨号失败：%v", err)
	}
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(joinTimeout + 5*time.Second))
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("握手后不发 join 的连接应在 joinTimeout 内被回收（读应失败）")
	}
	if h.engine != nil {
		t.Fatal("被回收的握手连接不得创建引擎")
	}
}

// TestConnLimitRejectsAndReleases：并发上限——满员后新连接走既有 close-frame
// 拒绝路径；真实连接断开后名额即时释放（removeConn），新连接可再入（断线重连
// 不被锁死）。ghost 连接为直接在册的假 conn（上限语义以 conns 计数为准，
// 与既有测试直接布置 hub 状态同构）；释放步骤必须走真实断开路径——否则
// removeConn 丢失释放语义时测试不可红。冲突连接用 PvP 合法 p2 席位——若用
// 单人模式的第二名玩家，席位/模式检查会兜住拒绝，上限失效时测试仍绿。
func TestConnLimitRejectsAndReleases(t *testing.T) {
	h := New(passJudge{}, slog.Default(), nil)
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	// 前置：maxConns-1 个 ghost + 1 个真实 PvP p1（WAITING 大厅）= 恰满上限
	for i := 0; i < maxConns-1; i++ {
		h.mu.Lock()
		h.conns[match.Player(fmt.Sprintf("ghost-%d", i))] =
			&conn{outbox: make(chan []byte, 1), closed: make(chan struct{})}
		h.mu.Unlock()
	}
	real := dialJoin(t, srv.URL, &matchpb.JoinRequest{Player: "p1"})
	readSnapshot(t, real)
	h.mu.Lock()
	n := len(h.conns)
	h.mu.Unlock()
	if n != maxConns {
		t.Fatalf("前置失败：连接数应恰达上限 %d，实际 %d", maxConns, n)
	}

	// 超限：第 maxConns+1 个连接（合法 PvP p2 席位）被拒——唯一的拒绝理由是上限
	ws := dialJoin(t, srv.URL, &matchpb.JoinRequest{Player: "p2"})
	_ = ws.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("超限连接应被拒连")
	}
	ws.Close()

	// 释放：断开真实连接，名额即时可用（上限按活跃连接计，非曾连接数）
	real.Close()
	deadline := time.Now().Add(3 * time.Second)
	for {
		h.mu.Lock()
		n := len(h.conns)
		h.mu.Unlock()
		if n == maxConns-1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("断开后连接名额应即时释放，conns=%d", n)
		}
		time.Sleep(10 * time.Millisecond)
	}
	// 再入：释放后的合法 p2 席位接入成功（收到 WAITING 快照）
	ws2 := dialJoin(t, srv.URL, &matchpb.JoinRequest{Player: "p2"})
	defer ws2.Close()
	snap := readSnapshot(t, ws2)
	if snap.GetPhase() != matchpb.MatchPhase_MATCH_PHASE_WAITING {
		t.Fatalf("释放后再入应进入 WAITING 大厅，实际 %v", snap.GetPhase())
	}
}
