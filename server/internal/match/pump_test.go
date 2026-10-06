package match

import (
	"sync"
	"testing"
	"time"

	"codewar/server/gen/matchpb"
)

// ===== 计时泵回归（浏览器验收发现）：提前结算后泵必须存活 =====

type manualTimer struct{ c chan time.Time }

func newManualTimer() *manualTimer         { return &manualTimer{c: make(chan time.Time, 1)} }
func (t *manualTimer) C() <-chan time.Time { return t.c }
func (t *manualTimer) Stop() bool          { return true }
func (t *manualTimer) fire() {
	select {
	case t.c <- time.Time{}:
	default: // 缓冲已有未消费 tick，吞掉本次
	}
}

type pumpClock struct {
	mu    sync.Mutex
	now   time.Time
	timer *manualTimer
}

func (c *pumpClock) Now() time.Time          { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *pumpClock) Advance(d time.Duration) { c.mu.Lock(); c.now = c.now.Add(d); c.mu.Unlock() }
func (c *pumpClock) NewTimer(time.Duration) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := newManualTimer()
	c.timer = t
	return t
}
func (c *pumpClock) current() *manualTimer { c.mu.Lock(); defer c.mu.Unlock(); return c.timer }

// safeCollector：泵在独立 goroutine 里发事件，收集器需并发安全。
type safeCollector struct {
	mu   sync.Mutex
	envs []*matchpb.Envelope
}

func (s *safeCollector) collect(e *matchpb.Envelope) {
	s.mu.Lock()
	s.envs = append(s.envs, e)
	s.mu.Unlock()
}
func (s *safeCollector) all() []*matchpb.Envelope {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*matchpb.Envelope(nil), s.envs...)
}

func maxRound(envs []*matchpb.Envelope) int {
	m := 0
	for _, e := range envs {
		if x, ok := e.Event.(*matchpb.Envelope_RoundStarted); ok && int(x.RoundStarted.Round) > m {
			m = int(x.RoundStarted.Round)
		}
	}
	return m
}

// 回归：R1 被"双方解出"提前结算后，armTimer 换上的 R2 timer 必须仍能被泵消费。
// 修复前：泵阻塞在已 Stop 的 R1 timerC 上，R2 超时永不触发，对局永久停滞。
func TestPumpSurvivesEarlyResolution(t *testing.T) {
	clk := &pumpClock{now: baseTime}
	col := &safeCollector{}
	jg := &scriptJudge{results: []JudgeResult{passOK()}}
	eng := NewEngine(clk, StubQuestions{}, jg, col.collect)
	eng.StartMatch()
	t1 := clk.current()

	// R1 双方解出 → 提前结算 → R2 armed（旧泵在此刻卡死）
	eng.Submit(P1, ActionAttack, "good", Pos{})
	eng.Submit(P2, ActionAttack, "good", Pos{})
	if maxRound(col.all()) != 2 {
		t.Fatalf("双方解出后应提前结算进入回合 2")
	}

	done := make(chan struct{})
	go func() { eng.Run(); close(done) }()

	// 陈旧触发：fire 已被 Stop 的 R1 timer —— deadline 残响防护应忽略，不推进回合
	t1.fire()
	time.Sleep(300 * time.Millisecond)
	if maxRound(col.all()) != 2 {
		t.Fatal("陈旧 timer 触发不应推进回合")
	}

	// R2 到期 → 泵消费 → 回合 3 开始（修复前此步永远挂起）
	clk.Advance(RoundDuration)
	clk.current().fire()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && maxRound(col.all()) < 3 {
		time.Sleep(20 * time.Millisecond)
	}
	if maxRound(col.all()) < 3 {
		t.Fatal("提前结算后泵应消费新回合的超时（回归：泵卡死在旧 timerC）")
	}

	// 收尾：连发超时把对局推进到回合上限结束，让 Run() 退出
	for i := 0; i < 12; i++ {
		select {
		case <-done:
			return
		default:
		}
		clk.Advance(RoundDuration)
		clk.current().fire()
		time.Sleep(50 * time.Millisecond)
	}
}
