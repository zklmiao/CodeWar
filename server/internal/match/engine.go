// Package match：MatchEngine——唯一集成 seam（spec「实现决策」）。
// 引擎只发布对局事件流；提交经 Submit 进入（生产 = WS hub，测试 = 脚本回放），
// 计时经 Clock（生产 = 真实时钟，测试 = 可拨动假钟）。
// 03 号票起：三动作（攻击/移动/防御）+ 5×5 动态站位 + 结算位置命中判定
// （相邻 4 格；走位闪避）+ 本回合护盾（−60%）。
package match

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"codewar/server/gen/matchpb"
)

// 行动类型（proto SubmitRequest.action / ActionResolved.action 同值）
const (
	ActionAttack = "attack"
	ActionMove   = "move"
	ActionDefend = "defend"
)

// GridSize：竞技场边长（5×5，7×7 预留）
const GridSize = 5

// DefendFactor：护盾减伤系数（−60%；spec 用户故事 22）
const DefendFactor = 0.4

// Pos：网格坐标（行列从 0 计）
type Pos struct{ Row, Col int }

// pendingAction：一名玩家本回合已解出的行动（一回合至多一个）。
type pendingAction struct {
	kind    string // attack | move | defend
	solveMs int64
	moveTo  Pos // 仅 move
}

// panel：本回合三题面板（攻击镜像 + 双方私有行动题）。
type panel struct {
	attack Question
	move   map[Player]Question
	defend map[Player]Question
}

// questionFor：该玩家所选行动对应的判题用题。
func (p *panel) questionFor(player Player, action string) (Question, bool) {
	switch action {
	case ActionAttack:
		return p.attack, true
	case ActionMove:
		q, ok := p.move[player]
		return q, ok
	case ActionDefend:
		q, ok := p.defend[player]
		return q, ok
	}
	return Question{}, false
}

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
	panel      panel
	actions    map[Player]*pendingAction // 一回合一人一行动
	solveSeq   map[Player]int64          // 同毫秒 tie-break：先来后到
	solveCnt   int64
	pos        map[Player]Pos  // 结算位置（跨回合持续，移动题改变）
	shield     map[Player]bool // 本回合护盾（行动为 defend 即成立，回合结束消失）
	timer      Timer
	timerC     <-chan time.Time
	resetC     chan struct{} // armTimer 替换 timer 时 close+重建，唤醒阻塞在旧 timerC 上的泵
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
		actions:   map[Player]*pendingAction{},
		solveSeq:  map[Player]int64{},
		// 初始站位：垂直相邻（ruling #7 的坐标实例；03 起随移动题动态变化）
		pos:    map[Player]Pos{P1: {Row: 3, Col: 2}, P2: {Row: 2, Col: 2}},
		shield: map[Player]bool{},
		resetC: make(chan struct{}),
		done:   make(chan struct{}),
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
// resetC：armTimer 每次替换 timer 都会 close+重建——泵在 select 上阻塞期间绑定了
// 旧 timerC，若无此信号，提前结算（Submit 路径）后的新回合超时将永远无人消费。
func (e *Engine) Run() {
	for {
		select {
		case <-e.done:
			return
		case <-e.resetC: // timer 被替换，重新取最新 timerC
			continue
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

// Submit：玩家提交（03 号票：带行动选择）。规则：
//   - 非法 action / 回合外提交 / 已行动再交 / 对局已结束：静默丢弃（ADR-0002 迟到判无效）
//   - 移动参数非法：拒绝反馈事件（不判题不惩罚，改参数可重交）
//   - 锁定中提交：只回 locked 事件，不罚不续锁
//   - 判题失败（wrong-answer/timeout/syntax/runtime-error）：自损 3 + 锁 5 秒
//   - judge-error：基础设施故障，不罚不锁，可重交
//   - 通过：记录行动与解题耗时；双方都已行动 → 立即提前结算
func (e *Engine) Submit(player Player, action, code string, moveTo Pos) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.phase != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE {
		return
	}
	if _, already := e.actions[player]; already {
		return
	}
	if action != ActionAttack && action != ActionMove && action != ActionDefend {
		return // 非法 action：静默丢弃（裁决 2）
	}
	// 移动参数先验（裁决 2：判题成本宝贵，参数先验；拒绝不判题不惩罚）
	if action == ActionMove {
		if reason := e.validateMove(player, moveTo); reason != "" {
			e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_SubmissionRejected{SubmissionRejected: &matchpb.SubmissionRejected{
				Player: player, Reason: reason,
			}}})
			return
		}
	}

	now := e.clock.Now()
	if now.Before(e.lockUntil[player]) {
		e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_SubmissionJudged{SubmissionJudged: &matchpb.SubmissionJudged{
			Player: player, Passed: false, Category: CategoryLocked, Action: action,
		}}})
		return
	}

	q, _ := e.panel.questionFor(player, action)
	res := e.judge.Judge(context.Background(), code, q.Tests)
	if res.Category == CategoryJudgeError {
		e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_SubmissionJudged{SubmissionJudged: &matchpb.SubmissionJudged{
			Player: player, Passed: false, Category: CategoryJudgeError, Action: action,
		}}})
		return
	}

	if res.Passed {
		solveMs := now.Sub(e.roundStart).Milliseconds()
		if solveMs < 0 {
			solveMs = 0
		}
		e.actions[player] = &pendingAction{kind: action, solveMs: solveMs, moveTo: moveTo}
		e.solveCnt++
		e.solveSeq[player] = e.solveCnt
		e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_SubmissionJudged{SubmissionJudged: &matchpb.SubmissionJudged{
			Player: player, Passed: true, Category: CategoryOK, SolveMs: int32(solveMs), Action: action,
		}}})
		if len(e.actions) == 2 {
			e.resolveRound()
		}
		return
	}

	// 错误提交：自损 + 锁定
	e.hp[player] -= SelfDamage
	e.lockUntil[player] = now.Add(LockDuration)
	e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_SubmissionJudged{SubmissionJudged: &matchpb.SubmissionJudged{
		Player: player, Passed: false, Category: res.Category, SelfDamageApplied: true,
		LockUntilUnixMs: e.lockUntil[player].UnixMilli(), Action: action,
	}}})
	e.emitHp(player, -SelfDamage, "self-damage")
	e.checkSelfKO(player)
}

// validateMove：移动参数校验（直线正交距离 1-2，路径不穿对方，终点空格）。
// 返回拒绝原因（proto SubmissionRejected.reason），空串 = 合法。
// 直线约束说明：曼哈顿 1-2 允许 (1,1) 斜向，但"逐步正交走"的 L 形路径存在两种
// 中间格、语义含糊（裁决 3 的"路径中间格"单数语义），故限定同行或同列。
func (e *Engine) validateMove(p Player, to Pos) string {
	if to.Row < 0 || to.Row >= GridSize || to.Col < 0 || to.Col >= GridSize {
		return "out-of-bounds"
	}
	from := e.pos[p]
	dr, dc := to.Row-from.Row, to.Col-from.Col
	if dr != 0 && dc != 0 {
		return "diagonal"
	}
	dist := abs(dr) + abs(dc)
	if dist < 1 || dist > 2 {
		return "too-far"
	}
	opp := e.pos[other(p)]
	if dist == 2 {
		mid := Pos{Row: from.Row + dr/2, Col: from.Col + dc/2}
		if mid == opp {
			return "path-blocked"
		}
	}
	if to == opp {
		return "dest-occupied"
	}
	return ""
}

// HandleRoundTimeout：回合倒计时归零（ADR-0002：计时权威在服务端）。
// 结算仍会发生——已行动的提交照常结算，未行动的本回合无行动。
func (e *Engine) HandleRoundTimeout() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE {
		return
	}
	// 陈旧触发的残响防护：提前结算换新 timer 后，旧 timer 可能留下已入缓冲的
	// tick（Stop 不清缓冲）；未到 deadline 的触发一律忽略。
	if e.clock.Now().Before(e.deadline) {
		return
	}
	e.resolveRound()
}

// resolveRound：逐动作即时结算当前回合（调用方持锁，phase=ROUND_ACTIVE）。
// 按解题耗时升序（用户故事 16：先解出者先手）逐个落地：
//   - 移动：立即改变结算位置——后手攻击按攻击结算时刻的位置判命中，
//     先手移动出相邻 4 格即闪避（走位博弈核心语义，裁决 5）
//   - 攻击：结算时刻位置判定（相邻 4 格）→ 命中则按目标本回合护盾折减；
//     闪避不产生 HpChanged（验收 3）
//   - 防御：护盾覆盖本回合结算批次（票据验收 4"该回合所受伤害乘 0.4"——
//     与位置不同，护盾不是位置量，后手防御也保护先手受到的攻击）
//
// 批次内全部落地后统一判生死——同刻伤害才存在"双方同时归零"（spec 用户故事 25）。
func (e *Engine) resolveRound() {
	e.stopTimer()

	order := make([]Player, 0, len(e.actions))
	for p := range e.actions {
		order = append(order, p)
	}
	sort.Slice(order, func(i, j int) bool {
		if e.actions[order[i]].solveMs != e.actions[order[j]].solveMs {
			return e.actions[order[i]].solveMs < e.actions[order[j]].solveMs
		}
		return e.solveSeq[order[i]] < e.solveSeq[order[j]]
	})

	// 护盾集合：本回合行动为防御的玩家（结算批次开始即生效——验收 4）
	for _, p := range order {
		if e.actions[p].kind == ActionDefend {
			e.shield[p] = true
		}
	}

	for _, p := range order {
		a := e.actions[p]
		switch a.kind {
		case ActionMove:
			e.pos[p] = a.moveTo
			e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_ActionResolved{ActionResolved: &matchpb.ActionResolved{
				Attacker: p, Action: ActionMove, SolveMs: int32(a.solveMs),
				MoveTo: &matchpb.Position{Row: int32(a.moveTo.Row), Col: int32(a.moveTo.Col)},
			}}})
		case ActionDefend:
			e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_ActionResolved{ActionResolved: &matchpb.ActionResolved{
				Attacker: p, Action: ActionDefend, SolveMs: int32(a.solveMs),
			}}})
		case ActionAttack:
			target := other(p)
			base := damageFor(a.solveMs)
			if !isAdjacent(e.pos[p], e.pos[target]) {
				// 走位闪避：无伤害、无 HpChanged（验收 3）
				e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_ActionResolved{ActionResolved: &matchpb.ActionResolved{
					Attacker: p, Action: ActionAttack, SolveMs: int32(a.solveMs), Dodged: true,
				}}})
				continue
			}
			dmg := base
			shielded := e.shield[target]
			if shielded {
				dmg = int32(math.Round(float64(base) * DefendFactor))
			}
			e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_ActionResolved{ActionResolved: &matchpb.ActionResolved{
				Attacker: p, Action: ActionAttack, SolveMs: int32(a.solveMs),
				Damage: dmg, Shielded: shielded,
			}}})
			e.hp[target] -= dmg
			e.emitHp(target, -dmg, "attack")
		}
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
	e.panel = panel{
		attack: e.questions.AttackQuestion(n),
		move: map[Player]Question{
			P1: e.questions.MoveQuestion(n, P1),
			P2: e.questions.MoveQuestion(n, P2),
		},
		defend: map[Player]Question{
			P1: e.questions.DefendQuestion(n, P1),
			P2: e.questions.DefendQuestion(n, P2),
		},
	}
	now := e.clock.Now()
	e.roundStart = now
	e.deadline = now.Add(RoundDuration)
	e.actions = map[Player]*pendingAction{}
	e.solveSeq = map[Player]int64{}
	e.solveCnt = 0
	e.shield = map[Player]bool{} // 护盾不跨回合；位置跨回合持续

	e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_RoundStarted{RoundStarted: &matchpb.RoundStarted{
		Round: uint32(n), DeadlineUnixMs: e.deadline.UnixMilli(),
	}}})
	e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_PanelReady{PanelReady: &matchpb.PanelReady{
		Attack: problemToPb(e.panel.attack),
		P1:     privatePanelToPb(e.panel.move[P1], e.panel.defend[P1]),
		P2:     privatePanelToPb(e.panel.move[P2], e.panel.defend[P2]),
	}}})
	e.armTimer()
}

func (e *Engine) armTimer() {
	e.stopTimer()
	close(e.resetC) // 广播"timer 已更换"：唤醒阻塞在旧 timerC 上的泵
	e.resetC = make(chan struct{})
	t := e.clock.NewTimer(RoundDuration)
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
	if e.broadcast != nil {
		e.broadcast(env)
	}
}

// Snapshot：按查看者出快照（含其私有面板与锁定视图），接入/重连先发（ADR-0002）。
func (e *Engine) Snapshot(viewer Player) *matchpb.Snapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := &matchpb.Snapshot{
		Seq:   e.seq,
		Phase: e.phase,
		Round: uint32(e.round),
		Players: []*matchpb.PlayerView{
			playerView(P1, e.hp[P1], e.pos[P1], e.shield[P1]),
			playerView(P2, e.hp[P2], e.pos[P2], e.shield[P2]),
		},
	}
	if e.phase == matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE {
		s.DeadlineUnixMs = e.deadline.UnixMilli()
		s.Panel = &matchpb.Panel{
			Attack: problemToPb(e.panel.attack),
			Move:   problemToPb(e.panel.move[viewer]),
			Defend: problemToPb(e.panel.defend[viewer]),
		}
	}
	if lock, ok := e.lockUntil[viewer]; ok && e.clock.Now().Before(lock) {
		s.ViewerLockUntilUnixMs = lock.UnixMilli()
	}
	return s
}

// damageFor：伤害 = 基础 22 × 时间系数（solveMs→[0.6,1.0] 线性）四舍五入。
func damageFor(solveMs int64) int32 {
	ratio := float64(solveMs) / float64(RoundDuration.Milliseconds())
	if ratio > 1 {
		ratio = 1
	}
	coeff := 1.0 - CoeffSpan*ratio
	if coeff < CoeffMin {
		coeff = CoeffMin
	}
	return int32(math.Round(BaseDamage * coeff))
}

// isAdjacent：正交相邻（曼哈顿距离恰 1）——本票普攻形状（相邻 4 格）。
func isAdjacent(a, b Pos) bool {
	return abs(a.Row-b.Row)+abs(a.Col-b.Col) == 1
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func problemToPb(q Question) *matchpb.Problem {
	return &matchpb.Problem{
		Id: q.ID, Statement: q.Statement, Signature: q.Signature, StarterCode: q.StarterCode,
	}
}

func privatePanelToPb(move, defend Question) *matchpb.PrivatePanel {
	return &matchpb.PrivatePanel{Move: problemToPb(move), Defend: problemToPb(defend)}
}

func playerView(p Player, hp int32, pos Pos, shielded bool) *matchpb.PlayerView {
	return &matchpb.PlayerView{
		Id: p, Hp: hp, Shielded: shielded,
		Pos: &matchpb.Position{Row: int32(pos.Row), Col: int32(pos.Col)},
	}
}

func other(p Player) Player {
	if p == P1 {
		return P2
	}
	return P1
}
