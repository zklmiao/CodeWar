package match

import (
	"context"
	"testing"
	"time"

	"codewar/server/gen/matchpb"
)

// ===== 测试基建：假钟 / 假判题 / 事件收集器 =====
// 遵循 spec「测试决策」：只经公共边界（Submit / HandleRoundTimeout / Events / Snapshot）
// 断言事件流，不触碰引擎内部状态。

var baseTime = time.Unix(1767225600, 0).UTC() // 固定基准时刻，保证断言可复现

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time      { return c.now }
func (c *fakeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }
func (c *fakeClock) NewTimer(d time.Duration) Timer { return noopTimer{} }

type noopTimer struct{}

func (noopTimer) C() <-chan time.Time { return nil } // 测试直接调 HandleRoundTimeout，不经通道
func (noopTimer) Stop() bool          { return true }

// scriptJudge：按调用次序回放判题结果；超出脚本时从头循环。
type scriptJudge struct {
	results []JudgeResult
	calls   int
}

func (s *scriptJudge) Judge(_ context.Context, _ string, _ []TestCase) JudgeResult {
	r := s.results[s.calls%len(s.results)]
	s.calls++
	return r
}

func failWrong() JudgeResult { return JudgeResult{Passed: false, Category: CategoryWrongAnswer} }
func passOK() JudgeResult    { return JudgeResult{Passed: true, Category: CategoryOK} }

type harness struct {
	t    *testing.T
	clk  *fakeClock
	jg   *scriptJudge
	eng  *Engine
	envs *[]*matchpb.Envelope
}

func newHarness(t *testing.T, results ...JudgeResult) *harness {
	clk := &fakeClock{now: baseTime}
	jg := &scriptJudge{results: results}
	var envs []*matchpb.Envelope
	eng := NewEngine(clk, StubQuestions{}, jg, func(e *matchpb.Envelope) { envs = append(envs, e) })
	eng.StartMatch()
	return &harness{t: t, clk: clk, jg: jg, eng: eng, envs: &envs}
}

func (h *harness) events() []*matchpb.Envelope { return *h.envs }

func (h *harness) latest() *matchpb.Envelope {
	envs := *h.envs
	if len(envs) == 0 {
		h.t.Fatal("事件流为空")
	}
	return envs[len(envs)-1]
}

// ===== 事件类型收集器 =====

func roundsStarted(envs []*matchpb.Envelope) []*matchpb.RoundStarted {
	var out []*matchpb.RoundStarted
	for _, e := range envs {
		if x, ok := e.Event.(*matchpb.Envelope_RoundStarted); ok {
			out = append(out, x.RoundStarted)
		}
	}
	return out
}

func judged(envs []*matchpb.Envelope) []*matchpb.SubmissionJudged {
	var out []*matchpb.SubmissionJudged
	for _, e := range envs {
		if x, ok := e.Event.(*matchpb.Envelope_SubmissionJudged); ok {
			out = append(out, x.SubmissionJudged)
		}
	}
	return out
}

func actions(envs []*matchpb.Envelope) []*matchpb.ActionResolved {
	var out []*matchpb.ActionResolved
	for _, e := range envs {
		if x, ok := e.Event.(*matchpb.Envelope_ActionResolved); ok {
			out = append(out, x.ActionResolved)
		}
	}
	return out
}

func hpChanges(envs []*matchpb.Envelope) []*matchpb.HpChanged {
	var out []*matchpb.HpChanged
	for _, e := range envs {
		if x, ok := e.Event.(*matchpb.Envelope_HpChanged); ok {
			out = append(out, x.HpChanged)
		}
	}
	return out
}

func endings(envs []*matchpb.Envelope) []*matchpb.MatchEnded {
	var out []*matchpb.MatchEnded
	for _, e := range envs {
		if x, ok := e.Event.(*matchpb.Envelope_MatchEnded); ok {
			out = append(out, x.MatchEnded)
		}
	}
	return out
}

// checkSeq：事件序号必须严格单调递增（ADR-0002）。
func checkSeq(t *testing.T, envs []*matchpb.Envelope) {
	t.Helper()
	for i, e := range envs {
		if e.Seq != uint32(i+1) {
			t.Fatalf("事件序号断裂：envs[%d].seq=%d，期望 %d", i, e.Seq, i+1)
		}
	}
}

// ===== 测试：错误路径优先 =====

// 空回合：双方都未解出 → 回合照常翻页，无出手、无掉血。
func TestEmptyRound(t *testing.T) {
	h := newHarness(t)
	h.eng.HandleRoundTimeout()

	if got := len(roundsStarted(h.events())); got != 2 {
		t.Fatalf("空回合后应进入第 2 回合，实际 RoundStarted 事件数=%d", got)
	}
	if got := len(actions(h.events())); got != 0 {
		t.Fatalf("空回合不应有出手，实际 %d", got)
	}
	if got := len(hpChanges(h.events())); got != 0 {
		t.Fatalf("空回合不应掉血，实际 %d", got)
	}
	checkSeq(t, h.events())
}

// 错误提交：自损 3 + 锁 5 秒；锁内再交只回 locked 不罚；锁过期恢复惩罚。
func TestWrongSubmissionPenaltyAndLock(t *testing.T) {
	h := newHarness(t, failWrong())

	h.eng.Submit(P1, "bad code")
	js := judged(h.events())
	last := js[len(js)-1]
	if last.Player != P1 || last.Passed || last.Category != CategoryWrongAnswer || !last.SelfDamageApplied {
		t.Fatalf("错误提交事件不符：%+v", last)
	}
	hp := hpChanges(h.events())
	if len(hp) != 1 || hp[0].Player != P1 || hp[0].Hp != 97 || hp[0].Delta != -3 || hp[0].Cause != "self-damage" {
		t.Fatalf("自损事件不符：%+v", hp)
	}

	// 锁内再交：locked，无额外掉血
	h.eng.Submit(P1, "bad again")
	js = judged(h.events())
	if last = js[len(js)-1]; last.Category != CategoryLocked || last.SelfDamageApplied {
		t.Fatalf("锁内提交应判 locked 不罚：%+v", last)
	}
	if len(hpChanges(h.events())) != 1 {
		t.Fatal("锁内提交不应掉血")
	}

	// 时钟拨过 5 秒：恢复惩罚
	h.clk.Advance(LockMs * time.Millisecond)
	h.eng.Submit(P1, "bad third")
	if got := len(hpChanges(h.events())); got != 2 {
		t.Fatalf("锁过期后错误提交应再罚，掉血事件数=%d", got)
	}
	if snap := h.eng.Snapshot(P1); snap.Players[0].Hp != 94 {
		t.Fatalf("P1 血量应为 94，实际 %d", snap.Players[0].Hp)
	}
	checkSeq(t, h.events())
}

// judge-error（基础设施故障）：不罚不锁，可立即重交。
func TestJudgeErrorNoPenalty(t *testing.T) {
	h := newHarness(t, JudgeResult{Passed: false, Category: CategoryJudgeError}, passOK())

	h.eng.Submit(P1, "code")
	js := judged(h.events())
	if js[0].Category != CategoryJudgeError || js[0].SelfDamageApplied {
		t.Fatalf("judge-error 不应带自损：%+v", js[0])
	}
	if len(hpChanges(h.events())) != 0 {
		t.Fatal("judge-error 不应掉血")
	}
	// 不锁：直接重交成功
	h.eng.Submit(P1, "good code")
	js = judged(h.events())
	if js[1].Passed != true || js[1].SolveMs < 0 {
		t.Fatalf("重交应通过：%+v", js[1])
	}
	checkSeq(t, h.events())
}

// 同题竞速：耗时越短伤害越高（时间系数），结算按完成耗时排序（先解出者先手）。
func TestRaceTimeCoefficientAndOrder(t *testing.T) {
	h := newHarness(t, passOK())

	h.eng.Submit(P1, "good") // t=0，solveMs=0 → 系数 1.0 → 伤害 22
	h.clk.Advance(37500 * time.Millisecond)
	h.eng.Submit(P2, "good") // solveMs=37500 → 系数 0.8 → 伤害 17.6 → 18

	acts := actions(h.events())
	if len(acts) != 2 {
		t.Fatalf("双方通过应立即结算两笔出手，实际 %d", len(acts))
	}
	if acts[0].Attacker != P1 || acts[0].Damage != 22 || acts[0].SolveMs != 0 {
		t.Fatalf("先手应为 P1 伤害 22：%+v", acts[0])
	}
	if acts[1].Attacker != P2 || acts[1].Damage != 18 || acts[1].SolveMs != 37500 {
		t.Fatalf("后手应为 P2 伤害 18（22×0.8）： %+v", acts[1])
	}
	hp := hpChanges(h.events())
	if hp[0].Player != P2 || hp[0].Hp != 78 {
		t.Fatalf("P2 应先中 22 点：%+v", hp[0])
	}
	if hp[1].Player != P1 || hp[1].Hp != 82 {
		t.Fatalf("P1 应后中 18 点：%+v", hp[1])
	}
	checkSeq(t, h.events())
}

// 提前结算：双方都通过 → 立即结算，不等回合超时。
func TestEarlyResolution(t *testing.T) {
	h := newHarness(t, passOK())
	h.eng.Submit(P1, "good")
	h.clk.Advance(10 * time.Second)
	h.eng.Submit(P2, "good")

	if got := len(actions(h.events())); got != 2 {
		t.Fatalf("双方通过应提前结算，出手数=%d", got)
	}
	// 下一回合应已开始
	if got := roundsStarted(h.events()); len(got) != 2 || got[1].Round != 2 {
		t.Fatalf("结算后应进入第 2 回合：%+v", got)
	}
}

// 回合超时：只有一方解出 → 仅该方出手。
func TestRoundTimeoutSingleSolve(t *testing.T) {
	h := newHarness(t, passOK())
	h.eng.Submit(P1, "good")
	h.eng.HandleRoundTimeout()

	acts := actions(h.events())
	if len(acts) != 1 || acts[0].Attacker != P1 || acts[0].Damage != 22 {
		t.Fatalf("超时结算应只含 P1 出手：%+v", acts)
	}
	if got := roundsStarted(h.events()); len(got) != 2 {
		t.Fatalf("结算后应进入第 2 回合")
	}
}

// 已通过者再交：静默丢弃（一回合一个行动，用户故事 17）。
func TestSolvedPlayerCannotResubmit(t *testing.T) {
	h := newHarness(t, passOK())
	h.eng.Submit(P1, "good")
	n := len(h.events())
	h.eng.Submit(P1, "good again")
	if len(h.events()) != n {
		t.Fatal("已通过者再交应被静默丢弃")
	}
}

// 对局结束后提交：静默丢弃（迟到提交判无效，ADR-0002）。
func TestLateSubmissionIgnored(t *testing.T) {
	h := newHarness(t, failWrong())
	for i := 0; i < 40 && len(endings(h.events())) == 0; i++ {
		h.clk.Advance(LockMs*time.Millisecond + time.Millisecond)
		h.eng.Submit(P1, "bad")
	}
	if len(endings(h.events())) == 0 {
		t.Fatal("前置：P1 应已自损致死")
	}
	n := len(h.events())
	h.eng.Submit(P1, "late")
	h.eng.Submit(P2, "late")
	if len(h.events()) != n {
		t.Fatal("对局结束后的提交应被静默丢弃")
	}
}

// 自损致死：错误提交把 HP 磨到 0 → 对方胜（self-ko）。
func TestSelfKO(t *testing.T) {
	h := newHarness(t, failWrong())
	for i := 0; i < 40 && len(endings(h.events())) == 0; i++ {
		h.clk.Advance(LockMs*time.Millisecond + time.Millisecond)
		h.eng.Submit(P1, "bad")
	}
	ends := endings(h.events())
	if len(ends) != 1 {
		t.Fatalf("应恰好一次终局事件，实际 %d", len(ends))
	}
	if ends[0].Winner != P2 || ends[0].Reason != ReasonSelfKO {
		t.Fatalf("自损致死应由 P2 获胜：%+v", ends[0])
	}
	if hp := hpChanges(h.events()); hp[len(hp)-1].Hp > 0 {
		t.Fatalf("终局前最后掉血应 ≤0：%+v", hp[len(hp)-1])
	}
}

// 互相击杀 → 平局：双方同刻归零（结算批次内全部伤害生效）。
// 5 回合双方都秒解：每回合互扣 22，第 5 回合双双归零。
func TestMutualKODraw(t *testing.T) {
	h := newHarness(t, passOK())
	for r := 1; r <= 5 && len(endings(h.events())) == 0; r++ {
		h.eng.Submit(P1, "good")
		h.eng.Submit(P2, "good")
	}
	ends := endings(h.events())
	if len(ends) != 1 {
		t.Fatalf("应恰好一次终局事件，实际 %d", len(ends))
	}
	if ends[0].Winner != "" || ends[0].Reason != ReasonKO {
		t.Fatalf("互相击杀应判平局（winner 空、reason=ko）：%+v", ends[0])
	}
}

// 先手致死：P2 每回合自损 3 且被 P1 打 22（-25/回合），第 4 回合归零 → P1 胜。
func TestFirstStrikeKO(t *testing.T) {
	h := newHarness(t, passOK(), failWrong())
	for r := 1; r <= 10 && len(endings(h.events())) == 0; r++ {
		h.eng.Submit(P1, "good") // 秒解，伤害 22
		h.clk.Advance(LockMs*time.Millisecond + time.Millisecond)
		h.eng.Submit(P2, "bad") // P2 自损 3（错答后不再交）
		h.eng.HandleRoundTimeout()
	}
	ends := endings(h.events())
	if len(ends) != 1 {
		t.Fatalf("应恰好一次终局事件，实际 %d", len(ends))
	}
	if ends[0].Winner != P1 || ends[0].Reason != ReasonKO {
		t.Fatalf("先手致死应由 P1 获胜：%+v", ends[0])
	}
}

// 回合上限·平局：10 个空回合 → 按剩余 HP 判（相等 → 平局）。
func TestRoundsCapDraw(t *testing.T) {
	h := newHarness(t)
	for i := 0; i < MaxRounds; i++ {
		h.eng.HandleRoundTimeout()
	}
	ends := endings(h.events())
	if len(ends) != 1 {
		t.Fatalf("10 回合后应恰好一次终局，实际 %d", len(ends))
	}
	if ends[0].Winner != "" || ends[0].Reason != ReasonRoundsCap {
		t.Fatalf("等血应判平局（rounds-cap）：%+v", ends[0])
	}
}

// 回合上限·分胜负：R1 P1 得手（P2 -22），余下空回合 → 按剩余 HP P1 胜。
func TestRoundsCapWinner(t *testing.T) {
	h := newHarness(t, passOK())
	h.eng.Submit(P1, "good")
	for i := 0; i < MaxRounds; i++ {
		h.eng.HandleRoundTimeout()
	}
	ends := endings(h.events())
	if len(ends) != 1 || ends[0].Winner != P1 || ends[0].Reason != ReasonRoundsCap {
		t.Fatalf("回合上限按剩余 HP 应 P1 胜：%+v", ends)
	}
}

// 快照：中回合视图（含查看者私有锁视图）。
func TestSnapshotMidRound(t *testing.T) {
	h := newHarness(t, failWrong())
	h.eng.Submit(P2, "bad") // P2 自损并锁定

	s1 := h.eng.Snapshot(P1)
	if s1.Phase != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE || s1.Round != 1 {
		t.Fatalf("快照回合态不符：%+v", s1.Phase)
	}
	if s1.DeadlineUnixMs == 0 || s1.Problem == nil || s1.Problem.Id == "" {
		t.Fatal("活跃回合快照应含截止时间与题目")
	}
	if s1.ViewerLockUntilUnixMs != 0 {
		t.Fatal("P1 未锁定，快照不应带锁")
	}
	if s1.Players[0].Id != P1 || s1.Players[0].Hp != 100 || s1.Players[1].Hp != 97 {
		t.Fatalf("快照血量不符：%v", s1.Players)
	}
	s2 := h.eng.Snapshot(P2)
	if s2.ViewerLockUntilUnixMs == 0 {
		t.Fatal("P2 锁定中，其快照应带锁截止")
	}
}

// 镜像题：双方每回合拿到同一实例，回合间轮换（用户故事 12、ruling #5）。
func TestMirrorProblemRotation(t *testing.T) {
	h := newHarness(t, passOK())
	h.eng.Submit(P1, "good")
	h.clk.Advance(time.Second)
	h.eng.Submit(P2, "good") // 提前结算 → 进入第 2 回合

	prs := panels(h.events())
	if len(prs) != 2 {
		t.Fatalf("两回合应各有一次 PanelReady，实际 %d", len(prs))
	}
	if prs[1].Problem.Id == prs[0].Problem.Id {
		t.Fatalf("回合间应轮换题目：%s 两回合相同", prs[0].Problem.Id)
	}
}

func panels(envs []*matchpb.Envelope) []*matchpb.PanelReady {
	var out []*matchpb.PanelReady
	for _, e := range envs {
		if x, ok := e.Event.(*matchpb.Envelope_PanelReady); ok {
			out = append(out, x.PanelReady)
		}
	}
	return out
}

// 回归测试：回合计时的单位必须是毫秒（曾经把 75000 当 time.Duration=75µs，
// 导致回合以判题耗时节奏疯狂翻页）。真实时钟下 200ms 内回合不得推进。
func TestRealClockRoundDoesNotAdvanceEarly(t *testing.T) {
	eng := NewEngine(RealClock{}, StubQuestions{}, &scriptJudge{results: []JudgeResult{failWrong()}}, nil)
	eng.StartMatch()
	eng.Submit(P1, "bad")
	eng.Submit(P2, "bad")
	time.Sleep(200 * time.Millisecond)
	if snap := eng.Snapshot(P1); snap.Round != 1 || snap.Phase != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE {
		t.Fatalf("200ms 内回合不得推进：round=%d phase=%s", snap.Round, snap.Phase)
	}
}
