package match

import (
	"context"
	"testing"
	"time"

	"codewar/server/gen/matchpb"
)

// ===== 测试基建：假钟 / 假判题 / 事件收集器 =====
// 遵循 spec「测试决策」：只经公共边界（Submit / HandleRoundTimeout / Snapshot）
// 断言事件流，不触碰引擎内部状态。

var baseTime = time.Unix(1767225600, 0).UTC() // 固定基准时刻，保证断言可复现

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time                 { return c.now }
func (c *fakeClock) Advance(d time.Duration)        { c.now = c.now.Add(d) }
func (c *fakeClock) NewTimer(d time.Duration) Timer { return noopTimer{} }

type noopTimer struct{}

func (noopTimer) C() <-chan time.Time { return nil } // 测试直接调 HandleRoundTimeout，不经通道
func (noopTimer) Stop() bool          { return true }

// scriptJudge：按调用次序回放判题结果；超出脚本时从头循环；记录每次判题的
// 用例名集合（供"判题用对题"断言：移动/防御桩题带专属探针用例）。
type scriptJudge struct {
	results  []JudgeResult
	calls    int
	seenTest [][]string // 每次判题收到的用例名
}

func (s *scriptJudge) Judge(_ context.Context, _ string, tests []TestCase) JudgeResult {
	names := make([]string, 0, len(tests))
	for _, tc := range tests {
		names = append(names, tc.Name)
	}
	s.seenTest = append(s.seenTest, names)
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

// ===== 行动提交助手（覆盖 Submit 新签名的三个便利封装） =====

func (h *harness) attack(p Player, code string) { h.eng.Submit(p, ActionAttack, code, Pos{}) }
func (h *harness) defend(p Player, code string) { h.eng.Submit(p, ActionDefend, code, Pos{}) }
func (h *harness) moveTo(p Player, code string, to Pos) {
	h.eng.Submit(p, ActionMove, code, to)
}

// timeoutRound：推进假钟越过本回合 deadline 再触发超时——与生产语义一致
// （HandleRoundTimeout 对未到期的陈旧触发会忽略，见 pump_test 的残响防护）。
func (h *harness) timeoutRound() {
	h.clk.Advance(RoundDuration + time.Second)
	h.eng.HandleRoundTimeout()
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

func panels(envs []*matchpb.Envelope) []*matchpb.PanelReady {
	var out []*matchpb.PanelReady
	for _, e := range envs {
		if x, ok := e.Event.(*matchpb.Envelope_PanelReady); ok {
			out = append(out, x.PanelReady)
		}
	}
	return out
}

func rejections(envs []*matchpb.Envelope) []*matchpb.SubmissionRejected {
	var out []*matchpb.SubmissionRejected
	for _, e := range envs {
		if x, ok := e.Event.(*matchpb.Envelope_SubmissionRejected); ok {
			out = append(out, x.SubmissionRejected)
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

// 空回合：双方都未行动 → 回合照常翻页，无出手、无掉血。
func TestEmptyRound(t *testing.T) {
	h := newHarness(t)
	h.timeoutRound()

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

	h.attack(P1, "bad code")
	js := judged(h.events())
	last := js[len(js)-1]
	if last.Player != P1 || last.Passed || last.Category != CategoryWrongAnswer || !last.SelfDamageApplied || last.Action != ActionAttack {
		t.Fatalf("错误提交事件不符：%+v", last)
	}
	hp := hpChanges(h.events())
	if len(hp) != 1 || hp[0].Player != P1 || hp[0].Hp != 97 || hp[0].Delta != -3 || hp[0].Cause != "self-damage" {
		t.Fatalf("自损事件不符：%+v", hp)
	}

	// 锁内再交：locked，无额外掉血
	h.attack(P1, "bad again")
	js = judged(h.events())
	if last = js[len(js)-1]; last.Category != CategoryLocked || last.SelfDamageApplied {
		t.Fatalf("锁内提交应判 locked 不罚：%+v", last)
	}
	if len(hpChanges(h.events())) != 1 {
		t.Fatal("锁内提交不应掉血")
	}

	// 时钟拨过 5 秒：恢复惩罚
	h.clk.Advance(LockDuration)
	h.attack(P1, "bad third")
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

	h.attack(P1, "code")
	js := judged(h.events())
	if js[0].Category != CategoryJudgeError || js[0].SelfDamageApplied {
		t.Fatalf("judge-error 不应带自损：%+v", js[0])
	}
	if len(hpChanges(h.events())) != 0 {
		t.Fatal("judge-error 不应掉血")
	}
	// 不锁：直接重交成功
	h.attack(P1, "good code")
	js = judged(h.events())
	if js[1].Passed != true || js[1].SolveMs < 0 {
		t.Fatalf("重交应通过：%+v", js[1])
	}
	checkSeq(t, h.events())
}

// 同题竞速：耗时越短伤害越高（时间系数），结算按完成耗时排序（先解出者先手）。
func TestRaceTimeCoefficientAndOrder(t *testing.T) {
	h := newHarness(t, passOK())

	h.attack(P1, "good") // t=0，solveMs=0 → 系数 1.0 → 伤害 22
	h.clk.Advance(37500 * time.Millisecond)
	h.attack(P2, "good") // solveMs=37500 → 系数 0.8 → 伤害 17.6 → 18

	acts := actions(h.events())
	if len(acts) != 2 {
		t.Fatalf("双方通过应立即结算两笔出手，实际 %d", len(acts))
	}
	if acts[0].Attacker != P1 || acts[0].Action != ActionAttack || acts[0].Damage != 22 || acts[0].SolveMs != 0 {
		t.Fatalf("先手应为 P1 攻击伤害 22：%+v", acts[0])
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

// 提前结算：双方都已行动 → 立即结算，不等回合超时。
func TestEarlyResolution(t *testing.T) {
	h := newHarness(t, passOK())
	h.attack(P1, "good")
	h.clk.Advance(10 * time.Second)
	h.attack(P2, "good")

	if got := len(actions(h.events())); got != 2 {
		t.Fatalf("双方通过应提前结算，出手数=%d", got)
	}
	// 下一回合应已开始
	if got := roundsStarted(h.events()); len(got) != 2 || got[1].Round != 2 {
		t.Fatalf("结算后应进入第 2 回合：%+v", got)
	}
}

// 回合超时：只有一方行动 → 仅该方出手。
func TestRoundTimeoutSingleSolve(t *testing.T) {
	h := newHarness(t, passOK())
	h.attack(P1, "good")
	h.timeoutRound()

	acts := actions(h.events())
	if len(acts) != 1 || acts[0].Attacker != P1 || acts[0].Damage != 22 {
		t.Fatalf("超时结算应只含 P1 出手：%+v", acts)
	}
	if got := roundsStarted(h.events()); len(got) != 2 {
		t.Fatalf("结算后应进入第 2 回合")
	}
}

// 已行动者再交：静默丢弃（一回合一个行动，用户故事 17）。
func TestSolvedPlayerCannotResubmit(t *testing.T) {
	h := newHarness(t, passOK())
	h.attack(P1, "good")
	n := len(h.events())
	h.attack(P1, "good again")
	if len(h.events()) != n {
		t.Fatal("已行动者再交应被静默丢弃")
	}
}

// 对局结束后提交：静默丢弃（迟到提交判无效，ADR-0002）。
func TestLateSubmissionIgnored(t *testing.T) {
	h := newHarness(t, failWrong())
	for i := 0; i < 40 && len(endings(h.events())) == 0; i++ {
		h.clk.Advance(LockDuration + time.Millisecond)
		h.attack(P1, "bad")
	}
	if len(endings(h.events())) == 0 {
		t.Fatal("前置：P1 应已自损致死")
	}
	n := len(h.events())
	h.attack(P1, "late")
	h.attack(P2, "late")
	if len(h.events()) != n {
		t.Fatal("对局结束后的提交应被静默丢弃")
	}
}

// 自损致死：错误提交把 HP 磨到 0 → 对方胜（self-ko）。
func TestSelfKO(t *testing.T) {
	h := newHarness(t, failWrong())
	for i := 0; i < 40 && len(endings(h.events())) == 0; i++ {
		h.clk.Advance(LockDuration + time.Millisecond)
		h.attack(P1, "bad")
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
// 5 回合双方都秒解攻击：每回合互扣 22，第 5 回合双双归零。
func TestMutualKODraw(t *testing.T) {
	h := newHarness(t, passOK())
	for r := 1; r <= 5 && len(endings(h.events())) == 0; r++ {
		h.attack(P1, "good")
		h.attack(P2, "good")
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
		h.attack(P1, "good") // 秒解，伤害 22
		h.clk.Advance(LockDuration + time.Millisecond)
		h.attack(P2, "bad") // P2 自损 3（错答后不再交）
		h.timeoutRound()
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
		h.timeoutRound()
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
	h.attack(P1, "good")
	for i := 0; i < MaxRounds; i++ {
		h.timeoutRound()
	}
	ends := endings(h.events())
	if len(ends) != 1 || ends[0].Winner != P1 || ends[0].Reason != ReasonRoundsCap {
		t.Fatalf("回合上限按剩余 HP 应 P1 胜：%+v", ends)
	}
}

// 快照：中回合视图（三题面板按查看者过滤私有题 + 双方位置 + 私有锁视图）。
func TestSnapshotMidRound(t *testing.T) {
	h := newHarness(t, failWrong())
	h.defend(P2, "bad") // P2 自损并锁定

	s1 := h.eng.Snapshot(P1)
	if s1.Phase != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE || s1.Round != 1 {
		t.Fatalf("快照回合态不符：%+v", s1.Phase)
	}
	if s1.DeadlineUnixMs == 0 || s1.Panel == nil || s1.Panel.Attack == nil || s1.Panel.Move == nil || s1.Panel.Defend == nil {
		t.Fatal("活跃回合快照应含三题面板")
	}
	if s1.Panel.Move.Id != "move-stub-p1" || s1.Panel.Defend.Id != "defend-stub-p1" {
		t.Fatalf("P1 快照的私有题应属于 P1：%s / %s", s1.Panel.Move.Id, s1.Panel.Defend.Id)
	}
	if s1.ViewerLockUntilUnixMs != 0 {
		t.Fatal("P1 未锁定，快照不应带锁")
	}
	if s1.Players[0].Id != P1 || s1.Players[0].Hp != 100 || s1.Players[1].Hp != 97 {
		t.Fatalf("快照血量不符：%v", s1.Players)
	}
	// 初始站位：P1(3,2)、P2(2,2)，垂直相邻
	if s1.Players[0].Pos.Row != 3 || s1.Players[0].Pos.Col != 2 ||
		s1.Players[1].Pos.Row != 2 || s1.Players[1].Pos.Col != 2 {
		t.Fatalf("快照位置不符：%v", s1.Players)
	}
	s2 := h.eng.Snapshot(P2)
	if s2.ViewerLockUntilUnixMs == 0 {
		t.Fatal("P2 锁定中，其快照应带锁截止")
	}
	if s2.Panel.Move.Id != "move-stub-p2" {
		t.Fatalf("P2 快照的私有移动题应属于 P2：%s", s2.Panel.Move.Id)
	}
}

// 镜像题：双方每回合拿到同一攻击实例，回合间轮换；私有题按玩家区分
// （验收 2：攻击镜像 / 移防私有）。
func TestMirrorProblemRotation(t *testing.T) {
	h := newHarness(t, passOK())
	h.attack(P1, "good")
	h.clk.Advance(time.Second)
	h.attack(P2, "good") // 提前结算 → 进入第 2 回合

	prs := panels(h.events())
	if len(prs) != 2 {
		t.Fatalf("两回合应各有一次 PanelReady，实际 %d", len(prs))
	}
	if prs[1].Attack.Id == prs[0].Attack.Id {
		t.Fatalf("回合间应轮换攻击题：%s 两回合相同", prs[0].Attack.Id)
	}
	// 私有性：同一回合内，P1/P2 的移动题实例不同（桩题 ID 含玩家维度）
	for i, pr := range prs {
		if pr.P1.Move.Id == pr.P2.Move.Id || pr.P1.Defend.Id == pr.P2.Defend.Id {
			t.Fatalf("第 %d 回合私有题应按玩家区分：%s", i+1, pr.P1.Move.Id)
		}
		if pr.P1.Move.Id != "move-stub-p1" || pr.P2.Move.Id != "move-stub-p2" {
			t.Fatalf("私有题 ID 未含玩家维度：%s / %s", pr.P1.Move.Id, pr.P2.Move.Id)
		}
	}
}

// 判题用对题（行动类型甄别）：移动提交用移动题用例判、防御提交用防御题用例判
// ——专属探针出现在判题入参当且仅当用了对应行动题（桩的攻击用例不含探针）。
// 注：探针只甄别行动类型，不甄别私有题"归属"（P1/P2 移动题用例集相同）；
// 归属由 TestSnapshotMidRound / TestMirrorProblemRotation 的面板 ID 断言另行覆盖。
func TestJudgeUsesSelectedPanelQuestion(t *testing.T) {
	h := newHarness(t, passOK())
	h.moveTo(P1, "good", Pos{Row: 3, Col: 0})
	if h.jg.calls != 1 {
		t.Fatalf("移动提交应触发一次判题，实际 %d", h.jg.calls)
	}
	found := false
	for _, name := range h.jg.seenTest[0] {
		if name == "move-only-probe" {
			found = true
		}
	}
	if !found {
		t.Fatalf("移动提交应用移动题用例判题（含 move-only-probe）：%v", h.jg.seenTest[0])
	}

	h.defend(P2, "good")
	found = false
	for _, name := range h.jg.seenTest[1] {
		if name == "defend-only-probe" {
			found = true
		}
	}
	if !found {
		t.Fatalf("防御提交应用防御题用例判题（含 defend-only-probe）：%v", h.jg.seenTest[1])
	}
}

// ===== 03 号票：三动作与走位 =====

// 一回合一个行动：移动后再交攻击 → 静默丢弃，本回合行动不变（验收 1）。
func TestOneActionPerRound(t *testing.T) {
	h := newHarness(t, passOK())
	h.moveTo(P1, "good", Pos{Row: 3, Col: 0})
	n := len(h.events())
	calls := h.jg.calls
	h.attack(P1, "good") // 已行动，静默丢弃
	if len(h.events()) != n {
		t.Fatal("同回合第二个行动应被静默丢弃")
	}
	if h.jg.calls != calls {
		t.Fatal("被丢弃的提交不应触发判题")
	}
	// 非法 action 同样静默丢弃
	h.eng.Submit(P1, "hack", "good", Pos{})
	if len(h.events()) != n || h.jg.calls != calls {
		t.Fatal("非法 action 应被静默丢弃且不判题")
	}
}

// 走位闪避（验收 3）：先手移出相邻 4 格 → 后手攻击闪避，无伤害、无 HpChanged。
func TestMoveAwayDodgesAttack(t *testing.T) {
	h := newHarness(t, passOK())
	h.moveTo(P1, "good", Pos{Row: 3, Col: 0}) // 先手：P1 移到 (3,0)，与 P2(2,2) 距离 3
	h.attack(P2, "good")                      // 后手攻击 → 闪避

	acts := actions(h.events())
	if len(acts) != 2 {
		t.Fatalf("应有两笔行动结算，实际 %d", len(acts))
	}
	if acts[0].Action != ActionMove || acts[0].MoveTo.Row != 3 || acts[0].MoveTo.Col != 0 {
		t.Fatalf("先手应为 P1 移动到 (3,0)：%+v", acts[0])
	}
	if acts[1].Attacker != P2 || acts[1].Action != ActionAttack || !acts[1].Dodged || acts[1].Damage != 0 {
		t.Fatalf("后手攻击应闪避无伤害：%+v", acts[1])
	}
	if got := len(hpChanges(h.events())); got != 0 {
		t.Fatalf("闪避不应产生任何掉血事件，实际 %d", got)
	}
	// 位置跨回合持续（快照为权威视图）
	if snap := h.eng.Snapshot(P1); snap.Players[0].Pos.Row != 3 || snap.Players[0].Pos.Col != 0 {
		t.Fatalf("P1 结算位置应为 (3,0)：%v", snap.Players[0].Pos)
	}
	checkSeq(t, h.events())
}

// 后手移动救不了已落地的攻击：攻击先结算时按当时位置命中（裁决 5 时序语义）。
func TestLateMoveCannotSaveLandedAttack(t *testing.T) {
	h := newHarness(t, passOK())
	h.attack(P1, "good")                      // 先手攻击：P2(2,2) 相邻 → 命中
	h.moveTo(P2, "good", Pos{Row: 2, Col: 0}) // 后手移动 → 救不了这一击

	acts := actions(h.events())
	if len(acts) != 2 || acts[0].Action != ActionAttack || acts[0].Damage != 22 || acts[0].Dodged {
		t.Fatalf("先手攻击应命中 22：%+v", acts)
	}
	hp := hpChanges(h.events())
	if len(hp) != 1 || hp[0].Player != P2 || hp[0].Hp != 78 {
		t.Fatalf("P2 应承受 22 点：%+v", hp)
	}
	if acts[1].Action != ActionMove || acts[1].MoveTo.Col != 0 {
		t.Fatalf("后手移动照常结算：%+v", acts[1])
	}
	checkSeq(t, h.events())
}

// 护盾减伤（验收 4）：防御方受到的该回合命中伤害 ×0.4（22 → 9）。
func TestDefendReducesDamage(t *testing.T) {
	h := newHarness(t, passOK())
	h.defend(P1, "good") // P1 护盾
	h.attack(P2, "good") // P2 攻击相邻的 P1 → 命中且有盾

	acts := actions(h.events())
	if len(acts) != 2 || acts[1].Action != ActionAttack {
		t.Fatalf("应有一笔攻击结算：%+v", acts)
	}
	atk := acts[1]
	if atk.Dodged || !atk.Shielded || atk.Damage != 9 {
		t.Fatalf("护盾命中应 ×0.4（22→9）：%+v", atk)
	}
	hp := hpChanges(h.events())
	if len(hp) != 1 || hp[0].Player != P1 || hp[0].Hp != 91 {
		t.Fatalf("P1 应净受 9 点：%+v", hp)
	}
	checkSeq(t, h.events())
}

// 护盾覆盖整回合：先手攻击也吃后手防御的减伤（验收 4"该回合所受伤害"语义）。
// 时序假设钉住：双方 solveMs 均为 0，"先手"依赖引擎的同毫秒按提交序 tie-break
// （solveSeq）；spec 未定义同耗时平局语义，若 tie-break 语义变更，本测试须随之复核。
func TestShieldCoversFasterAttack(t *testing.T) {
	h := newHarness(t, passOK())
	h.attack(P1, "good") // 先手攻击（solveMs=0）
	h.defend(P2, "good") // 后手防御

	acts := actions(h.events())
	if len(acts) != 2 {
		t.Fatalf("两笔行动都应结算：%+v", acts)
	}
	atk := acts[0]
	if atk.Attacker != P1 || !atk.Shielded || atk.Damage != 9 {
		t.Fatalf("后手防御应折减先手攻击（该回合语义）：%+v", atk)
	}
	checkSeq(t, h.events())
}

// 移入覆盖格仍被命中（验收 5 前半）：接近者进相邻格后被攻击命中。
// 时序假设钉住：双方 solveMs 均为 0，行动顺序依赖提交序 tie-break（同上）。
func TestApproachIntoRangeStillHit(t *testing.T) {
	h := newHarness(t, passOK())
	// R1：P1 拉开距离，P2 防御（不追击）→ 攻击闪避
	h.moveTo(P1, "good", Pos{Row: 3, Col: 0})
	h.defend(P2, "good")
	// R2：P2 移动接近到 (2,0)——与 P1(3,0) 相邻；P1 攻击 → 命中
	h.moveTo(P2, "good", Pos{Row: 2, Col: 0})
	h.attack(P1, "good")

	acts := actions(h.events())
	if len(acts) != 4 {
		t.Fatalf("两回合各两笔行动：%+v", acts)
	}
	atk := acts[3]
	if atk.Attacker != P1 || atk.Action != ActionAttack || atk.Dodged || atk.Damage != 22 {
		t.Fatalf("进入相邻格的目标应被命中：%+v", atk)
	}
	hp := hpChanges(h.events())
	if len(hp) != 1 || hp[0].Player != P2 || hp[0].Hp != 78 {
		t.Fatalf("P2 应承受 22 点：%+v", hp)
	}
	checkSeq(t, h.events())
}

// 闪避优先于护盾（验收 5 后半）：有盾但不在射程内 → 只算闪避，无伤害。
// 时序假设钉住：双方 solveMs 均为 0，行动顺序依赖提交序 tie-break（同上）。
func TestDodgeCheckedBeforeShield(t *testing.T) {
	h := newHarness(t, passOK())
	// R1：P1 拉开距离，P2 挂盾
	h.moveTo(P1, "good", Pos{Row: 3, Col: 0})
	h.defend(P2, "good")
	// R2：P1 也挂盾；P2 攻击 → P1 既在盾又在射程外 → 闪避优先
	h.defend(P1, "good")
	h.attack(P2, "good")

	acts := actions(h.events())
	atk := acts[3]
	if atk.Attacker != P2 || !atk.Dodged || atk.Shielded || atk.Damage != 0 {
		t.Fatalf("射程外的有盾目标应判闪避（先闪避后护盾）：%+v", atk)
	}
	if got := len(hpChanges(h.events())); got != 0 {
		t.Fatalf("闪避不应掉血，实际 %d", got)
	}
	checkSeq(t, h.events())
}

// 移动参数校验负路径：出界/斜向/过远/穿人/落点被占 → 拒绝事件，不判题不惩罚。
func TestMoveValidationRejections(t *testing.T) {
	h := newHarness(t, passOK()) // 末段合法移动会判题一次；被拒段断言 calls==0
	cases := []struct {
		to     Pos
		reason string
	}{
		{Pos{Row: 5, Col: 2}, "out-of-bounds"},
		{Pos{Row: -1, Col: 2}, "out-of-bounds"},
		{Pos{Row: 3, Col: 5}, "out-of-bounds"},
		{Pos{Row: 2, Col: 3}, "diagonal"},
		{Pos{Row: 0, Col: 2}, "too-far"},
		{Pos{Row: 1, Col: 2}, "path-blocked"}, // 直线上行 2 格，中间 (2,2) 是 P2
		{Pos{Row: 2, Col: 2}, "dest-occupied"},
	}
	for _, c := range cases {
		h.moveTo(P1, "good", c.to)
		rej := rejections(h.events())
		if len(rej) == 0 || rej[len(rej)-1].Player != P1 || rej[len(rej)-1].Reason != c.reason {
			t.Fatalf("move_to %v 应拒绝为 %s：%+v", c.to, c.reason, rej)
		}
	}
	if h.jg.calls != 0 {
		t.Fatalf("被拒提交不应触发判题，实际 %d", h.jg.calls)
	}
	if got := len(hpChanges(h.events())); got != 0 {
		t.Fatalf("被拒提交不应惩罚，实际 %d 次掉血", got)
	}
	// 被拒不消耗行动：合法移动仍可提交并结算
	h.moveTo(P1, "good", Pos{Row: 3, Col: 1})
	h.attack(P2, "good")
	acts := actions(h.events())
	if len(acts) != 2 || acts[0].Action != ActionMove || acts[0].MoveTo.Col != 1 {
		t.Fatalf("被拒后合法移动应照常生效：%+v", acts)
	}
	checkSeq(t, h.events())
}

// 回归测试：回合计时的单位必须是毫秒（曾经把 75000 当 time.Duration=75µs，
// 导致回合以判题耗时节奏疯狂翻页）。真实时钟下 200ms 内回合不得推进。
func TestRealClockRoundDoesNotAdvanceEarly(t *testing.T) {
	eng := NewEngine(RealClock{}, StubQuestions{}, &scriptJudge{results: []JudgeResult{failWrong()}}, nil)
	eng.StartMatch()
	eng.Submit(P1, ActionAttack, "bad", Pos{})
	eng.Submit(P2, ActionAttack, "bad", Pos{})
	time.Sleep(200 * time.Millisecond)
	if snap := eng.Snapshot(P1); snap.Round != 1 || snap.Phase != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE {
		t.Fatalf("200ms 内回合不得推进：round=%d phase=%s", snap.Round, snap.Phase)
	}
}
