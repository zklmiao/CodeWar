// Package match：MatchEngine——唯一集成 seam（spec「实现决策」）。
// 引擎只发布对局事件流；提交经 Submit 进入（生产 = WS hub，测试 = 脚本回放），
// 计时经 Clock（生产 = 真实时钟，测试 = 可拨动假钟）。
package match

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"codewar/server/gen/matchpb"
)

type Engine struct {
	mu        sync.Mutex
	clock     Clock
	questions QuestionPort
	judge     JudgePort
	broadcast func(*matchpb.Envelope) // 生产由 hub 扇出；测试可注入收集器

	seq        uint32
	phase      matchpb.MatchPhase
	round      int
	roundNum   int // 已完成的结算批次数（用于回合上限裁决）
	hp         map[Player]int32
	lockUntil  map[Player]time.Time
	deadline   time.Time
	roundStart time.Time
	problem    Question
	solved     map[Player]int64 // solveMs
	solveSeq   map[Player]int64 // 同毫秒 tie-break：先来后到
	solveCnt   int64
	timer      Timer
	timerC     <-chan time.Time
	events     []*matchpb.Envelope
	done       chan struct{} // 结束时关闭，停掉计时泵
}

func NewEngine(clock Clock, questions QuestionPort, judge JudgePort, broadcast func(*matchpb.Envelope)) *Engine {
	return &Engine{
		clock:     clock,
		questions: questions,
		judge:     judge,
		broadcast: broadcast,
		phase:     matchpb.MatchPhase_MATCH_PHASE_WAITING,
		hp:        map[Player]int32{P1: 100, P2: 100},
		lockUntil: map[Player]time.Time{},
		solved:    map[Player]int64{},
		solveSeq:  map[Player]int64{},
		done:      make(chan struct{}),
	}
}

// StartMatch：双方就位后开第一回合。hub 在两名玩家都接入时调用一次。
func (e *Engine) StartMatch() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase != matchpb.MatchPhase_MATCH_PHASE_WAITING {
		return
	}
	e.startRound(1)
}

// Run：计时泵。生产由 hub 用 goroutine 启动；测试直接调 HandleRoundTimeout，不启动泵。
func (e *Engine) Run() {
	for {
		select {
		case <-e.done:
			return
		case <-e.currentTimerC():
			e.HandleRoundTimeout()
		}
	}
}

// currentTimerC 只在持锁的 armTimer/resolveRound 里被替换，泵里读取——
// 用小函数配锁避免 data race（泵协程与提交协程并发）。
func (e *Engine) currentTimerC() <-chan time.Time {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.timerC
}

// Submit：玩家提交（spec 用户故事 13-18）。同步走判题；规则：
//   - 锁定中提交：只回 locked 事件，不罚不续锁
//   - 判题失败（wrong-answer/timeout/syntax/runtime-error）：自损 3 + 锁 5 秒
//   - judge-error：基础设施故障，不罚不锁，可重交
//   - 通过：记录解题耗时；双方都通过 → 立即提前结算
//   - 已通过再交 / 回合外提交 / 对局已结束：静默丢弃（迟到提交判无效，ADR-0002）
func (e *Engine) Submit(player Player, code string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.phase != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE {
		return
	}
	if _, already := e.solved[player]; already {
		return
	}
	now := e.clock.Now()
	if now.Before(e.lockUntil[player]) {
		e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_SubmissionJudged{SubmissionJudged: &matchpb.SubmissionJudged{
			Player: player, Passed: false, Category: CategoryLocked,
		}}})
		return
	}

	res := e.judge.Judge(context.Background(), code, e.problem.Tests)
	if res.Category == CategoryJudgeError {
		e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_SubmissionJudged{SubmissionJudged: &matchpb.SubmissionJudged{
			Player: player, Passed: false, Category: CategoryJudgeError,
		}}})
		return
	}

	if res.Passed {
		solveMs := now.Sub(e.roundStart).Milliseconds()
		if solveMs < 0 {
			solveMs = 0
		}
		e.solved[player] = solveMs
		e.solveCnt++
		e.solveSeq[player] = e.solveCnt
		e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_SubmissionJudged{SubmissionJudged: &matchpb.SubmissionJudged{
			Player: player, Passed: true, Category: CategoryOK, SolveMs: int32(solveMs),
		}}})
		if len(e.solved) == 2 {
			e.resolveRound()
		}
		return
	}

	// 错误提交：自损 + 锁定
	e.hp[player] -= SelfDamage
	e.lockUntil[player] = now.Add(LockMs * time.Millisecond)
	e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_SubmissionJudged{SubmissionJudged: &matchpb.SubmissionJudged{
		Player: player, Passed: false, Category: res.Category, SelfDamageApplied: true,
	}}})
	e.emitHp(player, -SelfDamage, "self-damage")
	e.checkSelfKO(player)
}

// HandleRoundTimeout：回合倒计时归零（ADR-0002：计时权威在服务端）。
// 结算仍会发生——已通过的提交照常出手，未通过的本回合无行动。
func (e *Engine) HandleRoundTimeout() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE {
		return
	}
	e.resolveRound()
}

// resolveRound：结算当前回合（调用方持锁，phase=ROUND_ACTIVE）。
// 伤害按解题耗时升序出手（用户故事 16：先解出者先手）；
// 结算批次内伤害全部生效后统一判生死——同刻伤害才存在"双方同时归零"（spec 用户故事 25），
// 本票无防御/闪避，先手只体现在事件顺序与平局判定的先后上。
func (e *Engine) resolveRound() {
	e.stopTimer()

	order := make([]Player, 0, len(e.solved))
	for p := range e.solved {
		order = append(order, p)
	}
	sort.Slice(order, func(i, j int) bool {
		if e.solved[order[i]] != e.solved[order[j]] {
			return e.solved[order[i]] < e.solved[order[j]]
		}
		return e.solveSeq[order[i]] < e.solveSeq[order[j]]
	})

	damages := make(map[Player]int32, len(order))
	for _, p := range order {
		damages[p] = damageFor(e.solved[p])
	}
	for _, p := range order {
		dmg := damages[p]
		e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_ActionResolved{ActionResolved: &matchpb.ActionResolved{
			Attacker: p, Damage: dmg, SolveMs: int32(e.solved[p]),
		}}})
		target := other(p)
		e.hp[target] -= dmg
		e.emitHp(target, -dmg, "attack")
	}

	e.roundNum++
	d1, d2 := e.hp[P1] <= 0, e.hp[P2] <= 0
	switch {
	case d1 && d2:
		e.endMatch("", ReasonKO) // 双方同时归零 → 平局（spec 用户故事 25）
	case d2:
		e.endMatch(P1, ReasonKO)
	case d1:
		e.endMatch(P2, ReasonKO)
	case e.roundNum >= MaxRounds:
		// 回合上限：按剩余 HP 裁决（ruling #8）
		switch {
		case e.hp[P1] > e.hp[P2]:
			e.endMatch(P1, ReasonRoundsCap)
		case e.hp[P2] > e.hp[P1]:
			e.endMatch(P2, ReasonRoundsCap)
		default:
			e.endMatch("", ReasonRoundsCap)
		}
	default:
		e.startRound(e.round + 1)
	}
}

// checkSelfKO：自损致死的即时判定（错误提交路径，非结算批次）。
func (e *Engine) checkSelfKO(dead Player) {
	if e.hp[dead] > 0 {
		return
	}
	e.endMatch(other(dead), ReasonSelfKO)
}

func (e *Engine) startRound(n int) {
	e.round = n
	e.phase = matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE
	e.problem = e.questions.AttackQuestion(n)
	now := e.clock.Now()
	e.roundStart = now
	e.deadline = now.Add(RoundMs * time.Millisecond)
	e.solved = map[Player]int64{}
	e.solveSeq = map[Player]int64{}
	e.solveCnt = 0

	e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_RoundStarted{RoundStarted: &matchpb.RoundStarted{
		Round: uint32(n), DeadlineUnixMs: e.deadline.UnixMilli(),
	}}})
	e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_PanelReady{PanelReady: &matchpb.PanelReady{
		Problem: problemToPb(e.problem),
	}}})
	e.armTimer()
}

func (e *Engine) armTimer() {
	e.stopTimer()
	t := e.clock.NewTimer(RoundMs * time.Millisecond)
	e.timer = t
	e.timerC = t.C()
}

func (e *Engine) stopTimer() {
	if e.timer != nil {
		e.timer.Stop()
		e.timer = nil
		e.timerC = nil
	}
}

func (e *Engine) endMatch(winner Player, reason string) {
	e.phase = matchpb.MatchPhase_MATCH_PHASE_ENDED
	e.stopTimer()
	close(e.done)
	e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_MatchEnded{MatchEnded: &matchpb.MatchEnded{
		Winner: winner, Reason: reason,
	}}})
}

func (e *Engine) emitHp(p Player, delta int32, cause string) {
	e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_HpChanged{HpChanged: &matchpb.HpChanged{
		Player: p, Hp: e.hp[p], Delta: delta, Cause: cause,
	}}})
}

func (e *Engine) emit(env *matchpb.Envelope) {
	e.seq++
	env.Seq = e.seq
	env.UnixMs = e.clock.Now().UnixMilli()
	e.events = append(e.events, env)
	if e.broadcast != nil {
		e.broadcast(env)
	}
}

// Snapshot：按查看者出快照（含其私有锁定视图），接入/重连先发（ADR-0002）。
func (e *Engine) Snapshot(viewer Player) *matchpb.Snapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := &matchpb.Snapshot{
		Seq:   e.seq,
		Phase: e.phase,
		Round: uint32(e.round),
		Players: []*matchpb.PlayerView{
			{Id: P1, Hp: e.hp[P1]},
			{Id: P2, Hp: e.hp[P2]},
		},
	}
	if e.phase == matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE {
		s.DeadlineUnixMs = e.deadline.UnixMilli()
		s.Problem = problemToPb(e.problem)
	}
	if lock, ok := e.lockUntil[viewer]; ok && e.clock.Now().Before(lock) {
		s.ViewerLockUntilUnixMs = lock.UnixMilli()
	}
	return s
}

// Events：事件日志副本（审计/测试断言用）。
func (e *Engine) Events() []*matchpb.Envelope {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]*matchpb.Envelope, len(e.events))
	copy(out, e.events)
	return out
}

// damageFor：伤害 = 基础 22 × 时间系数（solveMs→[0.6,1.0] 线性）四舍五入。
func damageFor(solveMs int64) int32 {
	ratio := float64(solveMs) / float64(RoundMs)
	if ratio > 1 {
		ratio = 1
	}
	coeff := 1.0 - CoeffSpan*ratio
	if coeff < CoeffMin {
		coeff = CoeffMin
	}
	return int32(math.Round(BaseDamage * coeff))
}

func problemToPb(q Question) *matchpb.Problem {
	return &matchpb.Problem{
		Id: q.ID, Statement: q.Statement, Signature: q.Signature, StarterCode: q.StarterCode,
	}
}

func other(p Player) Player {
	if p == P1 {
		return P2
	}
	return P1
}
