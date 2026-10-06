// Package match：MatchEngine——唯一集成 seam（spec「实现决策」）。
// 引擎只发布对局事件流；提交经 Submit 进入（生产 = WS hub，测试 = 脚本回放），
// 计时经 Clock（生产 = 真实时钟，测试 = 可拨动假钟）。
// 03 号票起：三动作（攻击/移动/防御）+ 5×5 动态站位 + 结算位置命中判定
// （相邻 4 格；走位闪避）+ 本回合护盾（−60%）。
// 06 号票起：四槽面板（+技能位）、3 职业形状与数值（classes 包单一来源）、
// 方向/距离 targeting（SetTargeting，自动瞄准兜底）、技能冷却跨回合。
package match

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/classes"
)

// 行动类型（proto SubmitRequest.action / ActionResolved.action 同值）
const (
	ActionAttack = "attack"
	ActionMove   = "move"
	ActionDefend = "defend"
	ActionSkill  = "skill" // 06 号票：技能位行动（效果由职业表定义）
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

// panel：本回合四题面板（攻击镜像 + 双方私有行动题；06 号票 + 技能槽）。
type panel struct {
	attack Question
	move   map[Player]Question
	defend map[Player]Question
	skill  map[Player]Question // 仅装备了技能位的玩家有题（难度 = 所选上限档）
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
	case ActionSkill:
		q, ok := p.skill[player]
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
	// 06 号票：职业与技能（classes 包单一来源；缺省 = legacy 锚点，
	// HP100/攻22/相邻1格/移动≤2/无技能——既有测试与匿名对局零改动）
	cls            map[Player]classes.Class
	equipped       map[Player]classes.Skill // 已装备技能（局前盲选装备，局内唯一）
	skillReadyFrom map[Player]int           // 技能再度可用回合（第 N 回合释放 → N+CD）
	targeting      map[Player]Targeting     // 攻击/技能目标选择（可随时覆盖，结算时读取）
	timer          Timer
	timerC         <-chan time.Time
	resetC         chan struct{} // armTimer 替换 timer 时 close+重建，唤醒阻塞在旧 timerC 上的泵
	done           chan struct{} // 结束时关闭，停掉计时泵
}

func NewEngine(clock Clock, questions QuestionPort, judge JudgePort, broadcast func(*matchpb.Envelope)) *Engine {
	legacy := classes.Legacy()
	e := &Engine{
		clock:     clock,
		questions: questions,
		judge:     judge,
		broadcast: broadcast,
		phase:     matchpb.MatchPhase_MATCH_PHASE_WAITING,
		hp:        map[Player]int32{P1: legacy.HP, P2: legacy.HP},
		lockUntil: map[Player]time.Time{},
		actions:   map[Player]*pendingAction{},
		solveSeq:  map[Player]int64{},
		// 初始站位：垂直相邻（ruling #7 的坐标实例；03 起随移动题动态变化）
		pos:            map[Player]Pos{P1: {Row: 3, Col: 2}, P2: {Row: 2, Col: 2}},
		shield:         map[Player]bool{},
		cls:            map[Player]classes.Class{P1: legacy, P2: legacy},
		equipped:       map[Player]classes.Skill{},
		skillReadyFrom: map[Player]int{},
		targeting:      map[Player]Targeting{},
		resetC:         make(chan struct{}),
		done:           make(chan struct{}),
	}
	return e
}

// Setup：对局的职业配置（hub 从 JoinRequest.class/skill 装配；缺省 = 双方 legacy）。
type Setup struct {
	P1Class, P1Skill string
	P2Class, P2Skill string
}

// NewEngineWithSetup：带职业配置的构造（hub 用；未知组合由 hub 先行校验，
// 引擎侧对未知值 fail-closed 降级 legacy 并保留错误语义由 hub 报告）。
func NewEngineWithSetup(clock Clock, questions QuestionPort, judge JudgePort,
	broadcast func(*matchpb.Envelope), setup Setup) *Engine {
	e := NewEngine(clock, questions, judge, broadcast)
	_ = e.BindClass(P1, setup.P1Class, setup.P1Skill)
	_ = e.BindClass(P2, setup.P2Class, setup.P2Skill)
	return e
}

// BindClass：为玩家绑定职业与已装备技能（仅 WAITING 阶段可绑——开赛后数值
// 与初始 HP 已冻结）。classID 空 = 保持现状；组合非法 = 报错（hub 已先行校验，
// 此处是纵深防御）。绑定即以职业表覆盖初始 HP。
func (e *Engine) BindClass(player Player, classID, skillID string) error {
	if e.phase != matchpb.MatchPhase_MATCH_PHASE_WAITING {
		return fmt.Errorf("对局已开始，职业绑定被拒")
	}
	c := classes.Legacy()
	if classID != "" {
		got, ok := classes.ByID(classID)
		if !ok {
			return fmt.Errorf("未知职业 %q", classID)
		}
		c = got
	}
	sk, hasSkill, err := classes.SkillByID(c.ID, skillID)
	if err != nil {
		return err
	}
	e.cls[player] = c
	e.hp[player] = c.HP // 职业表初始 HP（仍 WAITING，无事件流出）
	if hasSkill {
		e.equipped[player] = sk
	} else {
		delete(e.equipped, player)
	}
	return nil
}

// SetTargeting：玩家提交/修改攻击/技能的目标选择（免费操作不耗答题，裁决 3）。
// 仅活跃回合接受；结算时读取最新值，未提交的方向/距离走自动瞄准。
func (e *Engine) SetTargeting(player Player, t Targeting) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE {
		return
	}
	e.targeting[player] = t
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

// Submit：玩家提交（03 号票：带行动选择；06 号票：+技能位）。规则：
//   - 非法 action / 回合外提交 / 已行动再交 / 对局已结束：静默丢弃（ADR-0002 迟到判无效）
//   - 技能位未装备 / 冷却中：拒绝反馈事件（不判题不惩罚）
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
	if action != ActionAttack && action != ActionMove && action != ActionDefend && action != ActionSkill {
		return // 非法 action：静默丢弃（裁决 2）
	}
	// 技能位前置校验（裁决 2：判题成本宝贵，参数先验；拒绝不判题不惩罚）：
	// 未装备 / 冷却中 → 拒绝反馈（验收 1"冷却中灰置不可选"的服务端权威面）
	if action == ActionSkill {
		if _, has := e.equipped[player]; !has {
			e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_SubmissionRejected{SubmissionRejected: &matchpb.SubmissionRejected{
				Player: player, Reason: "no-skill-equipped",
			}}})
			return
		}
		if e.round < e.skillReadyFrom[player] {
			e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_SubmissionRejected{SubmissionRejected: &matchpb.SubmissionRejected{
				Player: player, Reason: "skill-cooldown",
			}}})
			return
		}
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

// validateMove：移动参数校验（直线正交 1..职业移动上限，路径不穿对方，终点空格）。
// 返回拒绝原因（proto SubmissionRejected.reason），空串 = 合法。
// 直线约束说明：曼哈顿 1-2 允许 (1,1) 斜向，但"逐步正交走"的 L 形路径存在两种
// 中间格、语义含糊（裁决 3 的"路径中间格"单数语义），故限定同行或同列。
// 06 号票：距离上限来自职业表（游侠 ≤3，其余 ≤2）。
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
	if dist < 1 || dist > int(e.cls[p].MoveMax) {
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
			// 06 号票：普攻形状与数值来自职业表（裁决 1），方向/距离 targeting
			// 结算时读取最新值（未提交走自动瞄准，裁决 3）
			cls := e.cls[p]
			needsDist := cls.AttackShape == classes.ShapeSnipe
			t := e.effectiveTargeting(p, true, needsDist, cls.AttackRange)
			cells := coverageCells(e.pos[p], cls.AttackShape, cls.AttackDepth, cls.AttackRange, t)
			target := other(p)
			if !cells[e.pos[target]] {
				// 走位闪避：无伤害、无 HpChanged（验收 3）
				e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_ActionResolved{ActionResolved: &matchpb.ActionResolved{
					Attacker: p, Action: ActionAttack, SolveMs: int32(a.solveMs), Dodged: true,
				}}})
				continue
			}
			base := damageFor(a.solveMs, cls.AttackDamage)
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
		case ActionSkill:
			// 06 号票：技能结算（设计 003 §4 字面值；伤害受盾减免；治疗不溢出；
			// 未命中（形状覆盖判定）→ 伤害/汲取自疗都不发生，治疗系无目标概念）
			sk := e.equipped[p]
			target := other(p)
			needsDir := sk.Shape != classes.ShapeBurst8 && sk.Shape != classes.ShapeSelf
			needsDist := sk.Shape == classes.ShapeSnipe
			t := e.effectiveTargeting(p, needsDir, needsDist, sk.RangeMax)
			cells := coverageCells(e.pos[p], sk.Shape, sk.Depth, sk.RangeMax, t)
			hit := cells[e.pos[target]]
			switch sk.Kind {
			case classes.KindDamage:
				if !hit {
					e.emitSkillDodged(p, a.solveMs)
					break
				}
				e.applySkillDamage(p, target, a.solveMs, sk.Power)
			case classes.KindHeal:
				// 治疗不溢出（设计 003 §1）：以职业表最大 HP 封顶
				healed := e.clampHeal(p, sk.Heal)
				e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_ActionResolved{ActionResolved: &matchpb.ActionResolved{
					Attacker: p, Action: ActionSkill, SolveMs: int32(a.solveMs), Heal: healed,
				}}})
				if healed > 0 {
					e.hp[p] += healed
					e.emitHp(p, healed, "heal")
				}
			case classes.KindDrain:
				// 秘法汲取：命中 → 伤 25（受盾）+ 自疗 20（溢出截断）；
				// 未命中 → 两段都不发生（伤害没打中就没有汲取的血液可回）
				if !hit {
					e.emitSkillDodged(p, a.solveMs)
					break
				}
				e.applySkillDamage(p, target, a.solveMs, sk.Power)
				if healed := e.clampHeal(p, sk.Heal); healed > 0 {
					e.hp[p] += healed
					e.emitHp(p, healed, "heal")
				}
			}
		}
	}

	// 技能冷却登记（结算批次后统一登记：第 N 回合释放 → 第 N+CD 回合起可用，
	// 票据验收 5"第 N+3 回合前不可再选"对 CD=3 即 N+1/N+2 灰置）
	for _, p := range order {
		if e.actions[p].kind == ActionSkill {
			if sk, has := e.equipped[p]; has {
				e.skillReadyFrom[p] = e.round + int(sk.Cooldown)
			}
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
	// 06 号票：技能槽题仅对装备了技能的玩家出（难度 = 所选上限档，工厂承载）
	e.panel.skill = map[Player]Question{}
	for _, p := range []Player{P1, P2} {
		if _, has := e.equipped[p]; has {
			e.panel.skill[p] = e.questions.SkillQuestion(n, p)
		}
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

// emitSkillDodged：技能被闪避的统一事件（伤害/汲取的未命中分支共用）。
func (e *Engine) emitSkillDodged(p Player, solveMs int64) {
	e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_ActionResolved{ActionResolved: &matchpb.ActionResolved{
		Attacker: p, Action: ActionSkill, SolveMs: int32(solveMs), Dodged: true,
	}}})
}

// applySkillDamage：技能伤害落地（盾减免 → 事件 → 扣血）。
// KindDamage 与 KindDrain 共用（review 发现 3：两段原本逐行重复）。
func (e *Engine) applySkillDamage(attacker, target Player, solveMs int64, power int32) {
	dmg := power
	shielded := e.shield[target]
	if shielded {
		dmg = int32(math.Round(float64(dmg) * DefendFactor))
	}
	e.emit(&matchpb.Envelope{Event: &matchpb.Envelope_ActionResolved{ActionResolved: &matchpb.ActionResolved{
		Attacker: attacker, Action: ActionSkill, SolveMs: int32(solveMs),
		Damage: dmg, Shielded: shielded,
	}}})
	e.hp[target] -= dmg
	e.emitHp(target, -dmg, "skill")
}

// clampHeal：治疗不溢出（设计 003 §1）——以职业表最大 HP 封顶，负值截 0。
func (e *Engine) clampHeal(p Player, amount int32) int32 {
	if room := e.cls[p].HP - e.hp[p]; amount > room {
		amount = room
	}
	if amount < 0 {
		amount = 0
	}
	return amount
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
// 06 号票：PlayerView 携带职业/技能/冷却——盲选隐私（设计 003 §1）：
// 对局开始前不下发对方职业；查看者自己的行始终携带（含技能冷却剩余回合数）。
func (e *Engine) Snapshot(viewer Player) *matchpb.Snapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	revealOpponent := e.phase != matchpb.MatchPhase_MATCH_PHASE_WAITING
	playerRow := func(p Player) *matchpb.PlayerView {
		v := playerView(p, e.hp[p], e.pos[p], e.shield[p])
		v.ClassId = e.cls[p].ID
		if sk, has := e.equipped[p]; has {
			v.SkillId = sk.ID
		}
		if p == viewer {
			if e.round < e.skillReadyFrom[p] {
				v.SkillCdRounds = int32(e.skillReadyFrom[p] - e.round)
			}
		} else if !revealOpponent {
			v.ClassId = ""
			v.SkillId = ""
		}
		return v
	}
	s := &matchpb.Snapshot{
		Seq:   e.seq,
		Phase: e.phase,
		Round: uint32(e.round),
		Players: []*matchpb.PlayerView{
			playerRow(P1),
			playerRow(P2),
		},
	}
	if !revealOpponent {
		// 盲选期不泄露对手职业血量（110/90/95 三档可反推职业，review 发现）：
		// 对手行按初始 100 上报，开局（revealOpponent）即恢复真实值。
		opp := other(viewer)
		for _, row := range s.Players {
			if row.Id == opp {
				row.Hp = 100
			}
		}
	}
	if e.phase == matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE {
		s.DeadlineUnixMs = e.deadline.UnixMilli()
		s.Panel = &matchpb.Panel{
			Attack: problemToPb(e.panel.attack),
			Move:   problemToPb(e.panel.move[viewer]),
			Defend: problemToPb(e.panel.defend[viewer]),
		}
		if q, ok := e.panel.skill[viewer]; ok {
			s.Panel.Skill = problemToPb(q)
		}
	}
	if lock, ok := e.lockUntil[viewer]; ok && e.clock.Now().Before(lock) {
		s.ViewerLockUntilUnixMs = lock.UnixMilli()
	}
	return s
}

// damageFor：伤害 = 职业普攻基础值 × 时间系数（solveMs→[0.6,1.0] 线性）四舍五入。
// 06 号票起基础值来自职业表（legacy 锚点 = 22，与 01~03 号票行为一致）。
func damageFor(solveMs int64, base int32) int32 {
	ratio := float64(solveMs) / float64(RoundDuration.Milliseconds())
	if ratio > 1 {
		ratio = 1
	}
	coeff := 1.0 - CoeffSpan*ratio
	if coeff < CoeffMin {
		coeff = CoeffMin
	}
	return int32(math.Round(float64(base) * coeff))
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
