package bot

// 引擎 seam 集成测试：脚本化人类 + 真实 Controller + 假判题器（事件流断言外部行为）。
// 手动时钟让整局在毫秒内跑完，同时保留真实的 solveMs 语义（引擎用同一时钟测量）。

import (
	"context"
	"fmt"
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

type keyJudge struct {
	keys map[string]bool
}

func newKeyJudge(f *question.Factory) *keyJudge {
	j := &keyJudge{keys: map[string]bool{}}
	add := func(q match.Question) {
		if q.AnswerKey != "" {
			j.keys[q.AnswerKey] = true
		}
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

func (j *keyJudge) Judge(_ context.Context, code string, _ []match.TestCase) match.JudgeResult {
	if j.keys[code] {
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
	const matches = 12
	stats := map[Difficulty]*difficultyStats{}
	for _, d := range []Difficulty{Easy, Normal, Hard} {
		s := &difficultyStats{}
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
					s.solves = append(s.solves, int64(j.GetSolveMs()))
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
	if hard.wins <= easy.wins {
		t.Fatalf("hard 胜场（%d）应高于 easy（%d）", hard.wins, easy.wins)
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
	}
	sort.Strings(hits)
	if len(hits) > 0 {
		t.Fatalf("match 包出现 Bot 引用（引擎必须对 Bot 零感知）：\n%s", strings.Join(hits, "\n"))
	}
}
