package bot

// 引擎 seam 集成测试：脚本化人类 + 真实 Controller + 假判题器（事件流断言外部行为）。
// 手动时钟让整局在毫秒内跑完，同时保留真实的 solveMs 语义（引擎用同一时钟测量）。

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/match"
	"codewar/server/internal/question"
	"codewar/server/internal/question/templates"
)

// ===== 手动时钟（同时实现 match.Clock 与 bot.Clock） =====

type manualJob struct {
	id      int
	at      time.Time
	fn      func()
	stopped bool
	fired   bool
}

type manualClock struct {
	mu   sync.Mutex
	now  time.Time
	jobs []*manualJob
	next int
}

func newManualClock() *manualClock {
	return &manualClock{now: time.Unix(1700000000, 0)}
}

func (c *manualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *manualClock) AfterFunc(d time.Duration, fn func()) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	j := &manualJob{id: c.next, at: c.now.Add(d), fn: fn}
	c.next++
	c.jobs = append(c.jobs, j)
	return manualTimer{clock: c, job: j}
}

// NewTimer：引擎计时泵在本测试中不启动（回合超时由 HandleRoundTimeout 显式驱动）。
func (c *manualClock) NewTimer(d time.Duration) match.Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	j := &manualJob{id: c.next, at: c.now.Add(d), fn: func() {}}
	c.next++
	c.jobs = append(c.jobs, j)
	return manualTimer{clock: c, job: j}
}

type manualTimer struct {
	clock *manualClock
	job   *manualJob
}

func (t manualTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	if t.job.stopped || t.job.fired {
		return false
	}
	t.job.stopped = true
	return true
}

func (t manualTimer) C() <-chan time.Time { return nil }

// Advance：推进时钟并依序执行所有到期回调；回调中新注册的到期任务同批执行，
// 保证"回合开始 → Bot 调度 → 下一回合开始"的因果链完整。
func (c *manualClock) Advance(d time.Duration) {
	target := c.now.Add(d)
	for {
		c.mu.Lock()
		var next *manualJob
		for _, j := range c.jobs {
			if j.stopped || j.fired || j.at.After(target) {
				continue
			}
			if next == nil || j.at.Before(next.at) || (j.at.Equal(next.at) && j.id < next.id) {
				next = j
			}
		}
		if next == nil {
			if target.After(c.now) {
				c.now = target
			}
			c.mu.Unlock()
			return
		}
		if next.at.After(c.now) {
			c.now = next.at
		}
		next.fired = true
		fn := next.fn
		c.mu.Unlock()
		fn()
	}
}

// ===== 假判题器：只把参考解判过（starter 必然失败） =====

// testsFingerprint：判题用例集指纹。keyJudge 必须把"提交的代码"和"本次实际
// 判的题"绑定起来——只按参考解全集判过会让"拿另一槽/另一回合的答案"也 pass，
// 掩盖"提交了错题"的实现 bug。
func testsFingerprint(tests []match.TestCase) string {
	var b strings.Builder
	for _, tc := range tests {
		b.WriteString(tc.Name)
		b.WriteByte('\x1f')
		b.WriteString(tc.Call)
		b.WriteByte('\x1f')
		b.WriteString(tc.Expected)
		b.WriteByte('\x1f')
		b.WriteString(tc.Setup)
		b.WriteByte('\x1f')
		b.WriteString(tc.Compare)
		b.WriteByte('\n')
	}
	return b.String()
}

type keyJudge struct {
	answers map[string]string // 用例集指纹 → 该题参考解
}

func newKeyJudge(f *question.Factory) *keyJudge {
	j := &keyJudge{answers: map[string]string{}}
	add := func(q match.Question) {
		if q.AnswerKey == "" {
			return
		}
		k := testsFingerprint(q.Tests)
		if prev, ok := j.answers[k]; ok && prev != q.AnswerKey {
			// 用例集相同但参考解不同：判题器无法区分提交对应哪道题，测试前提不成立。
			panic("keyJudge 用例指纹冲突：同一用例集对应不同参考解")
		}
		j.answers[k] = q.AnswerKey
	}
	for round := 1; round <= match.MaxRounds; round++ {
		add(f.AttackQuestion(round))
		for _, p := range []match.Player{match.P1, match.P2} {
			add(f.MoveQuestion(round, p))
			add(f.DefendQuestion(round, p))
			add(f.SkillQuestion(round, p))
		}
	}
	return j
}

func (j *keyJudge) Judge(_ context.Context, code string, tests []match.TestCase) match.JudgeResult {
	if answer, ok := j.answers[testsFingerprint(tests)]; ok && code == answer {
		return match.JudgeResult{Passed: true, Category: match.CategoryOK}
	}
	return match.JudgeResult{Passed: false, Category: match.CategoryWrongAnswer}
}

// ===== 引擎包装：统计 Bot 的 Submit 次数（与判题事件对账） =====

type countingEngine struct {
	*match.Engine
	submits map[match.Player]int
}

func (e *countingEngine) Submit(p match.Player, action, code string, moveTo match.Pos) {
	e.submits[p]++
	e.Engine.Submit(p, action, code, moveTo)
}

// ===== 整局 harness =====

type matchHarness struct {
	clk      *manualClock
	eng      *match.Engine
	count    *countingEngine
	ctl      *Controller
	factory  *question.Factory
	events   []*matchpb.Envelope
	ended    *matchpb.MatchEnded
	botPass  int
	botFail  int
	locked   int
	rejected int
}

func newMatchHarness(t *testing.T, seed string, d Difficulty) *matchHarness {
	t.Helper()
	return newMatchHarnessWithJudge(t, seed, d, nil)
}

// newMatchHarnessWithJudge：judge=nil 时用 keyJudge（只认参考解）。
func newMatchHarnessWithJudge(t *testing.T, seed string, d Difficulty, judge match.JudgePort) *matchHarness {
	t.Helper()
	clk := newManualClock()
	factory := question.NewFactory(question.Config{
		MaxDifficulty: question.L3,
		Strategy:      question.StrategyClimb,
		MatchSeed:     seed,
	})
	factory.RegisterAll(templates.All()...)
	botClass, botSkill := Loadout(seed)
	if judge == nil {
		judge = newKeyJudge(factory)
	}

	h := &matchHarness{clk: clk, factory: factory}
	h.eng = match.NewEngineWithSetup(clk, factory, judge, h.broadcast, match.Setup{
		P1Class: "ranger", // 人类：点狙射程 3 + 移动 3，便于脚本追击
		P2Class: botClass, P2Skill: botSkill,
		RiskTier: 3,
	})
	h.count = &countingEngine{Engine: h.eng, submits: map[match.Player]int{}}
	ctl, err := New(Config{
		Seed: seed, Difficulty: d,
		ClassID: botClass, SkillID: botSkill, Player: match.P2,
		Questions: factory, Engine: h.count, Clock: clk,
	})
	if err != nil {
		t.Fatalf("构造 Bot 失败：%v", err)
	}
	h.ctl = ctl
	return h
}

// newConfiguredHarness：指定双方职业/技能与 Bot Params 的 harness（冷却/无技能
// 等确定性边界用；生产路径仍由 newMatchHarness 覆盖）。
func newConfiguredHarness(t *testing.T, seed, p1Class, p1Skill, p2Class, p2Skill string, params Params) *matchHarness {
	t.Helper()
	clk := newManualClock()
	factory := question.NewFactory(question.Config{
		MaxDifficulty: question.L3,
		Strategy:      question.StrategyClimb,
		MatchSeed:     seed,
	})
	factory.RegisterAll(templates.All()...)
	h := &matchHarness{clk: clk, factory: factory}
	h.eng = match.NewEngineWithSetup(clk, factory, newKeyJudge(factory), h.broadcast, match.Setup{
		P1Class: p1Class, P1Skill: p1Skill,
		P2Class: p2Class, P2Skill: p2Skill,
		RiskTier: 3,
	})
	h.count = &countingEngine{Engine: h.eng, submits: map[match.Player]int{}}
	ctl, err := New(Config{
		Seed: seed, Difficulty: Normal, Params: params,
		ClassID: p2Class, SkillID: p2Skill, Player: match.P2,
		Questions: factory, Engine: h.count, Clock: clk,
	})
	if err != nil {
		t.Fatalf("构造 Bot 失败：%v", err)
	}
	h.ctl = ctl
	return h
}

// broadcast：与 hub 同构——先记录/观察，再交给 Bot 调度（引擎锁内只调度不回调）。
func (h *matchHarness) broadcast(env *matchpb.Envelope) {
	h.events = append(h.events, env)
	if h.ctl != nil {
		h.ctl.OnEvent(env)
	}
	if j := env.GetSubmissionJudged(); j != nil && j.GetPlayer() == match.P2 {
		if j.GetPassed() {
			h.botPass++
		} else {
			h.botFail++
		}
		if j.GetCategory() == match.CategoryLocked {
			h.locked++
		}
	}
	if env.GetSubmissionRejected() != nil {
		h.rejected++
	}
	if me := env.GetMatchEnded(); me != nil {
		h.ended = me
	}
}

func (h *matchHarness) round() int {
	return int(h.eng.Snapshot(match.P1).GetRound())
}

func (h *matchHarness) active() bool {
	return h.eng.Snapshot(match.P1).GetPhase() == matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE
}

// humanAct：脚本化人类——对齐且射程内就点狙；否则沿主轴追击（移动题也走真判题）。
func (h *matchHarness) humanAct(round int) {
	snap := h.eng.Snapshot(match.P1)
	me, ok1 := posOf(snap, match.P1)
	opp, ok2 := posOf(snap, match.P2)
	if !ok1 || !ok2 {
		return
	}
	if dir, dist, ok := lineOfSight(me, opp, 3); ok {
		h.eng.SetTargeting(match.P1, match.Targeting{Direction: dir, Distance: dist})
		h.count.Submit(match.P1, match.ActionAttack, h.factory.AttackQuestion(round).AnswerKey, match.Pos{})
		return
	}
	to := chase(me, opp, 3)
	h.count.Submit(match.P1, match.ActionMove, h.factory.MoveQuestion(round, match.P1).AnswerKey, to)
}

// play：推进整局。人类每回合在 humanDelay 时行动；Bot 由注入时钟驱动。
func (h *matchHarness) play(humanDelay time.Duration) {
	h.eng.StartMatch()
	for i := 0; i < 30 && h.ended == nil; i++ {
		if !h.active() {
			return
		}
		round := h.round()
		// 人类解题耗时（期间 Bot 可能先提交）
		for elapsed := time.Duration(0); elapsed < humanDelay && h.ended == nil; elapsed += time.Second {
			h.clk.Advance(time.Second)
		}
		if h.ended != nil || !h.active() || h.round() != round {
			continue
		}
		h.humanAct(round)
		// 等 Bot（或超时）结束本回合；一旦换回合立即停，避免推进到下一回合深处
		for elapsed := time.Duration(0); elapsed < match.RoundDuration+5*time.Second; elapsed += time.Second {
			if h.ended != nil || !h.active() || h.round() != round {
				break
			}
			h.clk.Advance(time.Second)
		}
		if h.ended == nil && h.active() && h.round() == round {
			h.eng.HandleRoundTimeout()
		}
	}
	h.ctl.Stop()
}

// lineOfSight：同行/同列且距离 1..max 时可点狙。
func lineOfSight(from, opp match.Pos, max int) (string, int, bool) {
	if from.Row == opp.Row {
		d := opp.Col - from.Col
		if d >= 1 && d <= max {
			return "right", d, true
		}
		if d <= -1 && -d <= max {
			return "left", -d, true
		}
	}
	if from.Col == opp.Col {
		d := opp.Row - from.Row
		if d >= 1 && d <= max {
			return "down", d, true
		}
		if d <= -1 && -d <= max {
			return "up", -d, true
		}
	}
	return "", 0, false
}

// chase：沿主轴朝对手移动（最多 moveMax 格，不落到对手格）。
func chase(me, opp match.Pos, moveMax int) match.Pos {
	dr, dc := opp.Row-me.Row, opp.Col-me.Col
	if abs(dr) >= abs(dc) && dr != 0 {
		step := abs(dr)
		if step > moveMax {
			step = moveMax
		}
		return match.Pos{Row: me.Row + sign(dr)*step, Col: me.Col}
	}
	step := abs(dc)
	if step > moveMax {
		step = moveMax
	}
	return match.Pos{Row: me.Row, Col: me.Col + sign(dc)*step}
}

func sign(x int) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}

// ===== 验收 1：可单人完成整局并分出胜负 =====

func TestFullPveMatchReachesDecision(t *testing.T) {
	h := newMatchHarness(t, "pve-full-match", Hard)
	h.play(30 * time.Second)

	if h.ended == nil {
		t.Fatalf("整局未终局（events=%d）", len(h.events))
	}
	if h.ended.GetWinner() == "" {
		t.Fatalf("应分出胜负，实际平局（reason=%s）", h.ended.GetReason())
	}
	if h.botPass == 0 {
		t.Fatal("Bot 从未通过判题（参考解没有走通 SubmissionPort）")
	}
	// 正确分支必须提交"当前所选行动"那道题的参考解：只统计总通过数时，
	// 拿移动题答案去解攻击题会被其他槽位的通过掩盖（假判题器按用例集绑定）。
	botAttackPass := 0
	for _, env := range h.events {
		j := env.GetSubmissionJudged()
		if j != nil && j.GetPlayer() == match.P2 && j.GetAction() == match.ActionAttack && j.GetPassed() {
			botAttackPass++
		}
	}
	if botAttackPass == 0 {
		t.Fatal("Bot 从未通过攻击题（正确分支可能提交了错误槽位/回合的答案）")
	}
	// 每次 Bot 提交都产生判题事件：Submit 次数 = passed + failed（含 locked 以外的正常判定）。
	// locked/judge-error 会破坏"无跳过路径"的语义，单独断言为零。
	if h.count.submits[match.P2] == 0 {
		t.Fatal("Bot 未提交任何行动")
	}
	if got := h.botPass + h.botFail; got != h.count.submits[match.P2] {
		t.Fatalf("Bot 提交 %d 次但只有 %d 条判题事件（存在跳过路径）",
			h.count.submits[match.P2], got)
	}
	if h.locked != 0 {
		t.Fatalf("Bot 出现 %d 次锁定期提交（应等锁过期再行动）", h.locked)
	}
	// 逐条检查判题类别：locked/judge-error 等都不是"正常判题"；只断言 locked
	// 会漏掉 judge-error（它会计入 botFail，从而满足上面的提交/事件对账）。
	for _, env := range h.events {
		j := env.GetSubmissionJudged()
		if j == nil || j.GetPlayer() != match.P2 {
			continue
		}
		switch j.GetCategory() {
		case match.CategoryOK, match.CategoryWrongAnswer:
		default:
			t.Fatalf("Bot 提交产生了非正常判题类别 %q（locked/judge-error 等）", j.GetCategory())
		}
	}
	if h.rejected != 0 {
		t.Fatalf("Bot 出现 %d 次参数拒绝（规划的目标/移动必须合法）", h.rejected)
	}
	if h.count.submits[match.P1] == 0 {
		t.Fatal("脚本化人类未提交")
	}
}

// ===== 验收 2：难度可感知（统计断言，非单局） =====

type difficultyStats struct {
	matches int
	wins    int
	solves  []int64
}

func (s *difficultyStats) avgSolveMs() float64 {
	if len(s.solves) == 0 {
		return 0
	}
	var sum int64
	for _, v := range s.solves {
		sum += v
	}
	return float64(sum) / float64(len(s.solves))
}

// 同 seed 配对：easy 与 hard 使用完全相同的 matchSeed（同职业/同题目/同人类脚本），
// 唯一变量是 Bot 难度。断言 hard 平均 solveMs 至少比 easy 快 2 倍，且胜率更高。
func TestHardBotFasterAndWinsMoreThanEasy(t *testing.T) {
	const matches = 30
	stats := map[Difficulty]*difficultyStats{}
	for _, d := range []Difficulty{Easy, Normal, Hard} {
		s := &difficultyStats{}
		p := ParamsFor(d)
		for i := 0; i < matches; i++ {
			seed := fmt.Sprintf("paired-%02d", i)
			h := newMatchHarness(t, seed, d)
			h.play(30 * time.Second)
			if h.ended == nil {
				t.Fatalf("%s seed %s 未终局", d, seed)
			}
			s.matches++
			if h.ended.GetWinner() == string(match.P2) {
				s.wins++
			}
			for _, env := range h.events {
				if j := env.GetSubmissionJudged(); j != nil && j.GetPlayer() == match.P2 && j.GetPassed() {
					ms := int64(j.GetSolveMs())
					// 每条实测 solveMs 都必须落在该难度窗内（允许毫秒级调度抖动）。
					// 只断言平均值会让"整体慢半拍"的调度 bug 逃逸。
					if ms < p.MinSolveMs-500 || ms > p.MaxSolveMs+2000 {
						t.Fatalf("%s 实测 solveMs=%d 越出难度窗 [%d,%d]（seed=%s）",
							d, ms, p.MinSolveMs, p.MaxSolveMs, seed)
					}
					s.solves = append(s.solves, ms)
				}
			}
		}
		stats[d] = s
	}
	easy, normal, hard := stats[Easy], stats[Normal], stats[Hard]
	eAvg, nAvg, hAvg := easy.avgSolveMs(), normal.avgSolveMs(), hard.avgSolveMs()
	t.Logf("Bot 统计（%d 局/档）：easy avg=%.0fms win=%d/%d | normal avg=%.0fms win=%d/%d | hard avg=%.0fms win=%d/%d",
		matches, eAvg, easy.wins, easy.matches, nAvg, normal.wins, normal.matches, hAvg, hard.wins, hard.matches)
	if len(hard.solves) == 0 || len(easy.solves) == 0 {
		t.Fatal("样本为空")
	}
	if !(eAvg > nAvg && nAvg > hAvg) {
		t.Fatalf("平均 solveMs 未随难度单调下降：easy=%.0f normal=%.0f hard=%.0f", eAvg, nAvg, hAvg)
	}
	if eAvg < 2*hAvg {
		t.Fatalf("easy 平均 solveMs（%.0f）应至少为 hard（%.0f）的 2 倍", eAvg, hAvg)
	}
	// 胜率断言要有可辨别的差距，而不是 1 胜 0 也过关：30 局配对样本里
	// hard 至少比 easy 多赢 3 局（本测试固定 seed，结果可复现）。
	if hard.wins < easy.wins+3 {
		t.Fatalf("hard 胜场（%d）应至少比 easy（%d）多 3 局", hard.wins, easy.wins)
	}
}

// ===== 错误路径：Bot 错误分支走真实自损，最终可 self-ko =====

// Bot 全程提交 starter（正确率 0、0ms、只攻击）：每回合自损 3；人类法师
// 用可控解题耗时打出 11/11/11/11/11/15 伤害，把 Bot 压到 2 HP，
// 下一回合 Bot 自损致死 → self-ko（与人同一套引擎规则，无跳过路径）。
func TestBotSelfKO(t *testing.T) {
	clk := newManualClock()
	seed := "self-ko-seed"
	factory := question.NewFactory(question.Config{
		MaxDifficulty: question.L3, Strategy: question.StrategyClimb, MatchSeed: seed,
	})
	factory.RegisterAll(templates.All()...)
	h := &matchHarness{clk: clk, factory: factory}
	h.eng = match.NewEngineWithSetup(clk, factory, newKeyJudge(factory), h.broadcast, match.Setup{
		P1Class: "mage", P2Class: "mage", RiskTier: 3,
	})
	h.count = &countingEngine{Engine: h.eng, submits: map[match.Player]int{}}
	ctl, err := New(Config{
		Seed: seed, Difficulty: Normal,
		Params: Params{MinSolveMs: 0, MaxSolveMs: 0, Accuracy: 0,
			Weights: Weights{Attack: 100}},
		ClassID: "mage", Player: match.P2,
		Questions: factory, Engine: h.count, Clock: clk,
	})
	if err != nil {
		t.Fatalf("构造 Bot 失败：%v", err)
	}
	h.ctl = ctl
	h.eng.StartMatch()

	// 第 6 回合的 15 伤害 = 31.25s；其余 11 伤害 = 73s。
	humanSolve := []time.Duration{
		73 * time.Second, 73 * time.Second, 73 * time.Second,
		73 * time.Second, 73 * time.Second, 31 * time.Second,
	}
	for i, dt := range humanSolve {
		if h.ended != nil {
			t.Fatalf("第 %d 回合前对局已结束（预期第 7 回合 self-ko）", i+1)
		}
		round := h.round()
		h.clk.Advance(time.Millisecond) // 触发 Bot 的 0ms 错误提交（自损 3）
		h.clk.Advance(dt - time.Millisecond)
		if !h.active() || h.round() != round {
			t.Fatalf("第 %d 回合未按预期保持活跃（round=%d want=%d）", i+1, h.round(), round)
		}
		h.count.Submit(match.P1, match.ActionAttack, factory.AttackQuestion(round).AnswerKey, match.Pos{})
		// 错误提交不算已行动：本回合要等超时结算（Bot 已锁定/无 pending）。
		h.clk.Advance(match.RoundDuration)
		h.eng.HandleRoundTimeout()
	}
	if h.ended == nil {
		// 第 7 回合：Bot 0ms 自损应致死。
		h.clk.Advance(time.Millisecond)
	}
	if h.ended == nil {
		t.Fatal("Bot 未 self-ko（预期 2 HP 下再自损 3）")
	}
	if h.ended.GetReason() != match.ReasonSelfKO || h.ended.GetWinner() != string(match.P1) {
		t.Fatalf("应为 P1 胜 self-ko，实际 winner=%q reason=%q",
			h.ended.GetWinner(), h.ended.GetReason())
	}
	if h.botFail != 7 {
		t.Fatalf("Bot 应有 7 次失败提交（6 次自损 + 1 次致死），实际 %d", h.botFail)
	}
	if h.locked != 0 {
		t.Fatalf("Bot 不应撞上锁定期，实际 %d 次", h.locked)
	}
}

// ===== 技能冷却 / 无技能回退（确定性边界，不靠概率） =====

// 强制每回合都选技能（Skill 权重 100）：战士求生本能 CD3，第 1、4 回合可用；
// 其余回合冷却中，权重并入攻击。若控制器漏登记冷却，第 2/3 回合会提交技能
// 并被引擎以 skill-cooldown 拒绝，h.rejected 立刻非零。
func TestSkillCooldownPreventsRepeatedSkillSubmissions(t *testing.T) {
	h := newConfiguredHarness(t, "skill-cd", "ranger", "", "warrior", "survive",
		Params{MinSolveMs: 0, MaxSolveMs: 0, Accuracy: 1, Weights: Weights{Skill: 100}})
	h.eng.StartMatch()
	for i := 0; i < 5; i++ {
		if h.ended != nil || !h.active() {
			break
		}
		round := h.round()
		// 人类先行动、Bot 后提交：Bot 的技能提交会同时结算本回合，引擎同步
		// 开启下一回合——这正是"冷却登记被 c.round 推进跳过"的触发现场。
		h.count.Submit(match.P1, match.ActionDefend, h.factory.DefendQuestion(round, match.P1).AnswerKey, match.Pos{})
		if h.ended != nil || !h.active() {
			break
		}
		h.clk.Advance(time.Millisecond) // 触发 Bot 的 0ms 提交并结算
	}
	h.ctl.Stop()
	h.eng.Stop()

	skills := 0
	for _, env := range h.events {
		j := env.GetSubmissionJudged()
		if j != nil && j.GetPlayer() == match.P2 && j.GetAction() == match.ActionSkill && j.GetPassed() {
			skills++
		}
	}
	if skills != 2 {
		t.Fatalf("5 回合内应恰好 2 次技能（CD3：第 1、4 回合），实际 %d", skills)
	}
	if h.rejected != 0 {
		t.Fatalf("Bot 出现 %d 次拒绝（冷却中仍提交技能，或目标参数非法）", h.rejected)
	}
	if h.locked != 0 {
		t.Fatalf("Bot 出现 %d 次 locked 判定", h.locked)
	}
}

// 未装备技能时技能权重并入攻击：hasSkill 门控若被删掉，Bot 会选 skill，
// 引擎以 no-skill-equipped 拒绝（无判题事件）。
func TestNoSkillBotNeverSubmitsSkill(t *testing.T) {
	h := newConfiguredHarness(t, "no-skill", "ranger", "", "warrior", "",
		Params{MinSolveMs: 0, MaxSolveMs: 0, Accuracy: 1, Weights: Weights{Skill: 100}})
	h.eng.StartMatch()
	h.clk.Advance(time.Millisecond)
	h.ctl.Stop()
	h.eng.Stop()

	if h.rejected != 0 {
		t.Fatalf("未装备技能时出现拒绝：%d", h.rejected)
	}
	var judged *matchpb.SubmissionJudged
	for _, env := range h.events {
		if j := env.GetSubmissionJudged(); j != nil && j.GetPlayer() == match.P2 {
			judged = j
		}
	}
	if judged == nil {
		t.Fatal("Bot 未产生判题事件（提交被 no-skill-equipped 拒绝？）")
	}
	if judged.GetAction() != match.ActionAttack {
		t.Fatalf("无技能应回退为攻击，实际 %q", judged.GetAction())
	}
}

// ===== 验收 3：引擎包无 Bot 引用（结构断言） =====

func TestMatchPackageHasNoBotReferences(t *testing.T) {
	dir := filepath.Join("..", "match")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取 match 包目录失败：%v", err)
	}
	// 词边界匹配：避免 "both" 这类正常单词误报。
	re := regexp.MustCompile(`(?i)\bbot\b`)
	var hits []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("读取 %s 失败：%v", e.Name(), err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if re.MatchString(line) {
				hits = append(hits, fmt.Sprintf("%s:%d: %s", e.Name(), i+1, strings.TrimSpace(line)))
			}
		}
		// 词边界 regex 抓不到 isBot/pveBot/botSeat 这类驼峰标识符；
		// 用 AST 再扫一遍所有标识符（大小写不敏感包含 "bot"）。
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, e.Name(), b, parser.ParseComments)
		if err != nil {
			t.Fatalf("解析 %s 失败：%v", e.Name(), err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if ok && strings.Contains(strings.ToLower(id.Name), "bot") {
				hits = append(hits, fmt.Sprintf("%s: identifier %q", e.Name(), id.Name))
			}
			return true
		})
	}
	sort.Strings(hits)
	if len(hits) > 0 {
		t.Fatalf("match 包出现 Bot 引用（引擎必须对 Bot 零感知）：\n%s", strings.Join(hits, "\n"))
	}
}
