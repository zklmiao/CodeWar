package bot

import (
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/classes"
	"codewar/server/internal/match"
	"codewar/server/internal/question"
)

// Timer：Bot 调度用的一次性定时器（只关心取消）。
type Timer interface {
	Stop() bool
}

// Clock：Bot 的计时/调度注入点（生产 = 真实时钟；测试 = 手动推进时钟）。
// 与引擎 ClockPort 分开：Bot 不需要为推进对局创建 timer，只需要"延迟执行"。
type Clock interface {
	Now() time.Time
	AfterFunc(d time.Duration, fn func()) Timer
}

// RealClock：生产时钟。
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }
func (RealClock) AfterFunc(d time.Duration, fn func()) Timer {
	return realTimer{t: time.AfterFunc(d, fn)}
}

type realTimer struct{ t *time.Timer }

func (r realTimer) Stop() bool { return r.t.Stop() }

// EnginePort：Bot 对引擎的全部接入面——行动只经 Submit（SubmissionPort 的
// 同一实现路径），目标选择是真人客户端同样使用的 SetTargeting（免费操作），
// Snapshot 为只读观测（规划 targeting/走位）。引擎不知道 Bot 的存在。
type EnginePort interface {
	Submit(player match.Player, action, code string, moveTo match.Pos)
	SetTargeting(player match.Player, t match.Targeting)
	Snapshot(viewer match.Player) *matchpb.Snapshot
}

// Config：一个 Bot 控制器的装配参数（每局新建，终局/断线 Stop）。
type Config struct {
	Seed       string     // 本场 matchSeed（与出题工厂同源，决定出题再生与决策随机流）
	Difficulty Difficulty // easy | normal | hard
	// Params 非零时覆盖 ParamsFor(Difficulty)（测试/未来调参注入用）。
	Params    Params
	ClassID   string
	SkillID   string // 空 = 不装备技能位（技能权重并入攻击）
	Player    match.Player
	Questions match.QuestionPort
	Engine    EnginePort
	Clock     Clock
	// RNG 非空时直接使用（测试固定随机流）；否则由 Seed 派生。
	RNG    *rand.Rand
	Logger *slog.Logger
}

// Controller：模拟 Bot 的回合驱动器。
//
// 事件接入：hub 在广播路径追加 observer 调用 OnEvent（不侵入引擎事件路径）。
// OnEvent 只做"采样 + 调度"，真正的 Snapshot/Submit 在定时回调里执行——
// 广播发生在引擎锁内，同步回调引擎会自死锁。
type Controller struct {
	cfg      Config
	params   Params
	rng      *rand.Rand
	clock    Clock
	player   match.Player
	class    classes.Class
	skill    classes.Skill
	hasSkill bool
	logger   *slog.Logger

	mu              sync.Mutex
	stopped         bool
	round           int
	skillReadyFrom  int // 技能再度可用回合（第 N 回合释放 → N+CD）
	lastSubmitRound int // 最近一次提交所在回合（技能冷却以引擎判题结果为准）
	pending         Timer
}

// Loadout：按 matchSeed 确定性随机三职业之一 + 该职业的一个技能（开局前盲选，
// 开局由引擎快照揭示；与 06 号票的盲选机制复用，不引入新的隐藏规则）。
func Loadout(seed string) (classID, skillID string) {
	all := classes.All()
	if len(all) == 0 {
		return "", ""
	}
	r := rand.New(rand.NewSource(question.DeriveSeed(seed, "bot-class")))
	c := all[r.Intn(len(all))]
	if len(c.Skills) == 0 {
		return c.ID, ""
	}
	return c.ID, c.Skills[r.Intn(len(c.Skills))].ID
}

// New：构造控制器。非法职业/技能组合 fail-closed（hub 侧 Loadout 只产出合法组合，
// 此处是纵深防御；错误不会静默降级成另一种对局）。
func New(cfg Config) (*Controller, error) {
	if cfg.Questions == nil {
		return nil, fmt.Errorf("bot 缺少 QuestionPort")
	}
	if cfg.Engine == nil {
		return nil, fmt.Errorf("bot 缺少 EnginePort")
	}
	if cfg.Difficulty == "" {
		cfg.Difficulty = DefaultDifficulty
	}
	if _, ok := ParseDifficulty(string(cfg.Difficulty)); !ok {
		return nil, fmt.Errorf("未知 Bot 难度 %q", cfg.Difficulty)
	}
	cls, ok := classes.ByID(cfg.ClassID)
	if !ok {
		return nil, fmt.Errorf("未知 Bot 职业 %q", cfg.ClassID)
	}
	sk, hasSkill, err := classes.SkillByID(cfg.ClassID, cfg.SkillID)
	if err != nil {
		return nil, err
	}
	p := cfg.Params
	if p.isZero() {
		p = ParamsFor(cfg.Difficulty)
	}
	if p.MaxSolveMs < p.MinSolveMs {
		return nil, fmt.Errorf("Bot 耗时窗非法：[%d, %d]", p.MinSolveMs, p.MaxSolveMs)
	}
	if p.Accuracy < 0 || p.Accuracy > 1 {
		return nil, fmt.Errorf("Bot 正确率非法：%v", p.Accuracy)
	}
	if cfg.Player == "" {
		cfg.Player = match.P2
	}
	if cfg.Player != match.P1 && cfg.Player != match.P2 {
		return nil, fmt.Errorf("Bot 席位非法 %q", cfg.Player)
	}
	if cfg.Clock == nil {
		cfg.Clock = RealClock{}
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	rng := cfg.RNG
	if rng == nil {
		rng = rand.New(rand.NewSource(question.DeriveSeed(cfg.Seed, "bot-decisions")))
	}
	return &Controller{
		cfg:      cfg,
		params:   p,
		rng:      rng,
		clock:    cfg.Clock,
		player:   cfg.Player,
		class:    cls,
		skill:    sk,
		hasSkill: hasSkill,
		logger:   cfg.Logger,
	}, nil
}

// OnEvent：hub 广播 observer——消费 RoundStarted（调度本回合行动）与
// SubmissionJudged（以引擎结果为权威同步技能冷却）。其余事件忽略。
// 新回合会取消上一回合尚未触发的行动（对局提前结算时防止陈旧提交）。
func (c *Controller) OnEvent(env *matchpb.Envelope) {
	// 16 号票：技能冷却必须跟随引擎的判题结果。Bot 自己的 Correct 只是预测——
	// starter 也可能通过判题（假判题器/边界题），此时引擎已进入冷却而 Bot
	// 仍以为可用，下一回合重复提交会被 skill-cooldown 拒绝，回合无人行动直到超时。
	if j := env.GetSubmissionJudged(); j != nil {
		c.observeJudged(j)
		return
	}
	rs := env.GetRoundStarted()
	if rs == nil {
		return
	}
	round := int(rs.GetRound())

	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return
	}
	if c.pending != nil {
		c.pending.Stop()
		c.pending = nil
	}
	c.round = round
	skillReady := c.hasSkill && round >= c.skillReadyFrom
	d := Decide(c.params, c.rng, skillReady)
	c.mu.Unlock()

	t := c.clock.AfterFunc(time.Duration(d.SolveMs)*time.Millisecond, func() {
		c.act(round, d)
	})

	c.mu.Lock()
	defer c.mu.Unlock()
	// 回调可能在调度瞬间就已执行（0ms 测试时钟）：只在仍是当前回合时登记。
	if c.stopped || c.round != round {
		t.Stop()
		return
	}
	c.pending = t
}

// Stop：终局/断线时停止——取消未触发的行动，之后的一切回调都变成 no-op。
// 幂等。
func (c *Controller) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped {
		return
	}
	c.stopped = true
	if c.pending != nil {
		c.pending.Stop()
		c.pending = nil
	}
}

// act：定时回调——再生本回合同题（与引擎共享工厂/seed），规划目标后经
// engine.Submit 提交。所有参数/回合校验都在提交前完成；引擎侧无任何特判。
func (c *Controller) act(round int, d Decision) {
	c.mu.Lock()
	if c.stopped || c.round != round {
		c.mu.Unlock()
		return
	}
	c.pending = nil
	c.mu.Unlock()

	snap := c.cfg.Engine.Snapshot(c.player)
	if snap.GetPhase() != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE || int(snap.GetRound()) != round {
		return // 对局已结束或回合已切换：丢弃陈旧行动
	}
	me, ok := posOf(snap, c.player)
	if !ok {
		return
	}
	opp, ok := posOf(snap, otherPlayer(c.player))
	if !ok {
		return
	}

	var (
		q      match.Question
		moveTo match.Pos
	)
	switch d.Action {
	case match.ActionAttack:
		q = c.cfg.Questions.AttackQuestion(round)
		if t, ok := PlanTargeting(me, opp, c.class.AttackShape, c.class.AttackRange); ok {
			c.cfg.Engine.SetTargeting(c.player, t)
		}
	case match.ActionMove:
		q = c.cfg.Questions.MoveQuestion(round, c.player)
		c.mu.Lock()
		if c.stopped || c.round != round {
			c.mu.Unlock()
			return
		}
		to, ok := PickMove(c.rng, me, opp, int(c.class.MoveMax))
		c.mu.Unlock()
		if !ok {
			return
		}
		moveTo = to
	case match.ActionDefend:
		q = c.cfg.Questions.DefendQuestion(round, c.player)
	case match.ActionSkill:
		q = c.cfg.Questions.SkillQuestion(round, c.player)
		if t, ok := PlanTargeting(me, opp, c.skill.Shape, c.skill.RangeMax); ok {
			c.cfg.Engine.SetTargeting(c.player, t)
		}
	default:
		return
	}

	code := q.StarterCode
	if d.Correct && q.AnswerKey != "" {
		code = q.AnswerKey
	}
	// 记录本次提交所在回合：SubmissionJudged 在 Submit 内同步到达（早于
	// resolveRound/下一回合的 RoundStarted），handler 用它登记冷却。
	c.mu.Lock()
	if !c.stopped {
		c.lastSubmitRound = round
	}
	c.mu.Unlock()
	c.logger.Debug("bot 提交", "round", round, "action", d.Action,
		"correct", d.Correct, "solve_ms", d.SolveMs, "class", c.class.ID)
	c.cfg.Engine.Submit(c.player, d.Action, code, moveTo)
}

// observeJudged：技能真正通过判题后，按引擎口径登记冷却（第 N 回合释放 →
// 第 N+CD 回合起可用）。失败/未装备/冷却拒绝不登记。
func (c *Controller) observeJudged(j *matchpb.SubmissionJudged) {
	if j.GetPlayer() != c.player || !j.GetPassed() || j.GetAction() != match.ActionSkill {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped || c.lastSubmitRound <= 0 {
		return
	}
	c.skillReadyFrom = c.lastSubmitRound + int(c.skill.Cooldown)
}

func posOf(snap *matchpb.Snapshot, p match.Player) (match.Pos, bool) {
	for _, v := range snap.GetPlayers() {
		if v.GetId() != p || v.GetPos() == nil {
			continue
		}
		return match.Pos{Row: int(v.GetPos().GetRow()), Col: int(v.GetPos().GetCol())}, true
	}
	return match.Pos{}, false
}

func otherPlayer(p match.Player) match.Player {
	if p == match.P1 {
		return match.P2
	}
	return match.P1
}
