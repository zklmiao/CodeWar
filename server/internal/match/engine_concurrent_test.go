package match

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"codewar/server/gen/matchpb"
)

// ===== 29 号票：判题在锁外——镜像竞速公平性回归 =====
// 断言风格遵守 spec「测试决策」：只经公共边界（Submit / SetTargeting /
// Snapshot / HandleRoundTimeout）断言事件流与判题器调用次数，不触碰引擎内部状态。

// gatedJudge：可阻塞的假判题器。第 n 次调用（1 起）在 gates[n-1]（非 nil 时）
// 上阻塞直到 release(n) 放行；每次调用经 entered 上报序号（带缓冲，不阻塞判题）。
// 并发安全：calls 在锁内递增；gates 构造后只读；result 不可变。
type gatedJudge struct {
	mu      sync.Mutex
	calls   int
	result  JudgeResult
	entered chan int
	gates   []chan struct{}
}

func newGatedJudge(result JudgeResult, gatedCalls int) *gatedJudge {
	g := &gatedJudge{result: result, entered: make(chan int, 16)}
	for i := 0; i < gatedCalls; i++ {
		g.gates = append(g.gates, make(chan struct{}))
	}
	return g
}

func (g *gatedJudge) Judge(_ context.Context, _ string, _ []TestCase) JudgeResult {
	g.mu.Lock()
	g.calls++
	n := g.calls
	var gate chan struct{}
	if n <= len(g.gates) {
		gate = g.gates[n-1]
	}
	g.mu.Unlock()
	g.entered <- n
	if gate != nil {
		<-gate
	}
	return g.result
}

func (g *gatedJudge) callCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.calls
}

func (g *gatedJudge) release(n int) { close(g.gates[n-1]) }

// newConcurrentHarness：与 engine_test.go 的 harness 同构，判题器换为可阻塞的
// gatedJudge（同一 fakeClock 基准时刻，事件收集器逐 envelope 记录）。
func newConcurrentHarness(t *testing.T, jg *gatedJudge) (*fakeClock, *Engine, *[]*matchpb.Envelope) {
	t.Helper()
	clk := &fakeClock{now: baseTime}
	var envs []*matchpb.Envelope
	eng := NewEngine(clk, StubQuestions{}, jg, func(e *matchpb.Envelope) { envs = append(envs, e) })
	eng.StartMatch()
	return clk, eng, &envs
}

// waitDone：等待 Submit goroutine 收尾（超时防挂死）。
func waitDone(t *testing.T, done chan struct{}, what string) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s 未在超时内返回", what)
	}
}

// 公平性回归（修复前必须失败）：P1 判题阻塞期间 P2 提交——P2 的判题必须立即
// 开始（不被引擎锁挡住），且 P2 的 solveMs 按其提交时刻计：排队等待与判题耗时
// 都不计入解题耗时（spec 用户故事 12/15/16 镜像竞速公平）。
func TestJudgeOutsideLockFairness(t *testing.T) {
	jg := newGatedJudge(passOK(), 2) // 前两次判题都阻塞，手动放行
	clk, eng, envsP := newConcurrentHarness(t, jg)

	done1 := make(chan struct{})
	go func() { eng.Submit(P1, ActionAttack, "good", Pos{}); close(done1) }()
	select {
	case <-jg.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("前置：P1 判题未开始")
	}

	clk.Advance(10 * time.Second) // P2 在 P1 判题仍阻塞时提交（提交时刻 = +10s）

	done2 := make(chan struct{})
	go func() { eng.Submit(P2, ActionAttack, "good", Pos{}); close(done2) }()
	select {
	case <-jg.entered: // P2 的判题已开始（P1 判题仍阻塞，未 release）
	case <-time.After(2 * time.Second):
		t.Fatal("P2 的判题未能在 P1 判题阻塞期间开始（引擎在判题期间持锁）")
	}

	clk.Advance(20 * time.Second)
	jg.release(1)
	jg.release(2)
	waitDone(t, done1, "P1 Submit")
	waitDone(t, done2, "P2 Submit")

	// P2 的 solve_ms 必须按提交时刻（+10s）计，判题排队不污染耗时
	var p2 *matchpb.SubmissionJudged
	for _, j := range judged(*envsP) {
		if j.Player == P2 && j.Passed {
			p2 = j
		}
	}
	if p2 == nil || p2.SolveMs != 10000 {
		t.Fatalf("P2 solve_ms 应为 10000（按提交时刻计）：%+v", p2)
	}
	checkSeq(t, *envsP)
}

// 判题阻塞期间 SetTargeting 与 Snapshot 必须及时返回（修复前被引擎锁阻塞，
// 重连快照与目标选择全部卡死在别人的判题上）。
func TestEngineResponsiveDuringJudging(t *testing.T) {
	jg := newGatedJudge(passOK(), 1)
	_, eng, _ := newConcurrentHarness(t, jg)

	done := make(chan struct{})
	go func() { eng.Submit(P1, ActionAttack, "good", Pos{}); close(done) }()
	select {
	case <-jg.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("前置：P1 判题未开始")
	}

	errc := make(chan error, 1)
	go func() {
		start := time.Now()
		eng.SetTargeting(P1, Targeting{Direction: "up", Distance: 1})
		if snap := eng.Snapshot(P1); snap.Phase != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE {
			errc <- fmt.Errorf("快照阶段不符：%s", snap.Phase)
			return
		}
		if d := time.Since(start); d > 100*time.Millisecond {
			errc <- fmt.Errorf("SetTargeting/Snapshot 耗时 %v > 100ms", d)
			return
		}
		errc <- nil
	}()
	select {
	case err := <-errc:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("判题阻塞期间 SetTargeting/Snapshot 未返回（引擎在判题期间持锁）")
	}

	jg.release(1)
	waitDone(t, done, "P1 Submit")
}

// 判题阻塞期间回合超时结算：放行后过期结果必须被丢弃（不发事件不罚——
// 迟到提交判无效，ADR-0002），且在途标记已清，玩家在新回合可正常提交。
func TestStaleJudgementDroppedAfterRoundTimeout(t *testing.T) {
	jg := newGatedJudge(passOK(), 1) // 只给第一次判题上闸
	clk, eng, envsP := newConcurrentHarness(t, jg)

	done := make(chan struct{})
	go func() { eng.Submit(P1, ActionAttack, "good", Pos{}); close(done) }()
	select {
	case <-jg.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("前置：P1 判题未开始")
	}

	// 判题阻塞期间回合超时结算（修复前该调用被引擎锁阻塞，故加守卫防挂死）
	guarded := make(chan struct{})
	go func() {
		clk.Advance(RoundDuration + time.Second)
		eng.HandleRoundTimeout()
		close(guarded)
	}()
	select {
	case <-guarded:
	case <-time.After(2 * time.Second):
		t.Fatal("判题阻塞期间 HandleRoundTimeout 未返回（引擎在判题期间持锁）")
	}
	if got := len(roundsStarted(*envsP)); got != 2 {
		t.Fatalf("前置：超时应结算并进入第 2 回合，RoundStarted 数=%d", got)
	}

	jg.release(1)
	waitDone(t, done, "P1 Submit")

	// 过期结果被丢弃：事件流中不得出现 P1 的 SubmissionJudged
	for _, j := range judged(*envsP) {
		if j.Player == P1 {
			t.Fatalf("过期判题结果应被丢弃（不发事件）：%+v", j)
		}
	}

	// 在途标记已清：P1 在第 2 回合可正常提交并判题
	eng.Submit(P1, ActionAttack, "good", Pos{})
	js := judged(*envsP)
	if len(js) != 1 || js[0].Player != P1 || !js[0].Passed {
		t.Fatalf("第 2 回合应可正常提交判题：%+v", js)
	}
	checkSeq(t, *envsP)
}

// 同一玩家判题在途期间再交：静默丢弃，不触发第二次判题（与"已行动再交"一致）。
// 修复前经引擎锁串行化等价达成（第二次提交在锁上排队、落地后按已行动丢弃）；
// 修复后由 inFlight 在途标记直接丢弃——外部行为相同，判题都只发生一次。
func TestInFlightSubmitDeduplicated(t *testing.T) {
	jg := newGatedJudge(passOK(), 1)
	_, eng, envsP := newConcurrentHarness(t, jg)

	done1 := make(chan struct{})
	go func() { eng.Submit(P1, ActionAttack, "good", Pos{}); close(done1) }()
	select {
	case <-jg.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("前置：P1 判题未开始")
	}

	// 在途期间并发第二次提交（goroutine：修复前会阻塞在引擎锁上直到放行）
	done2 := make(chan struct{})
	go func() { eng.Submit(P1, ActionAttack, "good again", Pos{}); close(done2) }()

	jg.release(1)
	waitDone(t, done1, "第一次 Submit")
	waitDone(t, done2, "第二次 Submit")

	if got := jg.callCount(); got != 1 {
		t.Fatalf("判题应只被调用一次，实际 %d", got)
	}
	js := judged(*envsP)
	if len(js) != 1 || js[0].Player != P1 || !js[0].Passed {
		t.Fatalf("应恰好一次通过事件：%+v", js)
	}
	checkSeq(t, *envsP)
}
