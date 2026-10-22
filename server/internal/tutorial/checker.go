package tutorial

import (
	"fmt"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/match"
)

// GoalState：目标清单的一项（客户端 checkbox 的数据源）。
type GoalState struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Done  bool   `json:"done"`
}

// State：教程对局的当前状态（随 ServerMessage.tutorial 下发）。
type State struct {
	Level        int         `json:"level"`
	Title        string      `json:"title"`
	Mechanic     string      `json:"mechanic"`
	Slots        []string    `json:"slots"`
	Goals        []GoalState `json:"goals"`
	Completed    bool        `json:"completed"`
	NextLevel    int         `json:"next_level"` // 0 = 没有下一关
	AllCompleted bool        `json:"all_completed"`
	Cap          int         `json:"cap"`
	CapCoeff     float64     `json:"cap_coeff"`
}

// Checker：服务端目标判定器（观察事件流，与 Bot 同一 observer 通道）。
//
// 判定只发生在服务端：客户端没有任何上报/推进接口，只渲染 State。
// 非线程安全——hub 在 broadcast（持 h.mu）与 join（持 h.mu）中串行调用。
type Checker struct {
	level  Level
	player match.Player
	bot    match.Player
	cap    int

	goals []GoalState
	done  bool
	// L2 的回合内状态：Bot 的直线攻击是否落空、玩家本回合是否受伤。
	dodgedThisRound bool
	hurtThisRound   bool
}

// NewChecker：按关号与难度上限构造判定器；未知关号 fail-closed。
func NewChecker(level, cap int) (*Checker, error) {
	l, ok := LevelByN(level)
	if !ok {
		return nil, fmt.Errorf("未知教程关号 %d", level)
	}
	if cap <= 0 {
		cap = match.DefaultRiskTier
	}
	goals := make([]GoalState, len(l.Goals))
	for i, g := range l.Goals {
		goals[i] = GoalState{ID: g.ID, Label: g.Label}
	}
	return &Checker{level: l, player: match.P1, bot: match.P2, cap: cap, goals: goals}, nil
}

// Observe：应用一条对局事件并返回最新状态。达成后重复调用幂等。
func (c *Checker) Observe(env *matchpb.Envelope) State {
	if c.done {
		return c.State()
	}
	switch {
	case env.GetRoundStarted() != nil:
		// 回合作用域的判定重置：上一回合的受击/闪避不跨回合累计。
		c.dodgedThisRound = false
		c.hurtThisRound = false
	case env.GetHpChanged() != nil:
		hp := env.GetHpChanged()
		if hp.GetPlayer() == c.player && hp.GetDelta() < 0 {
			c.hurtThisRound = true
		}
	case env.GetActionResolved() != nil:
		c.observeAction(env.GetActionResolved())
	}
	return c.State()
}

func (c *Checker) observeAction(a *matchpb.ActionResolved) {
	switch c.level.N {
	case 1, 4:
		// 目标：玩家自己产生 1 次伤害（攻击命中）。
		if a.GetAttacker() == c.player && a.GetAction() == match.ActionAttack && a.GetDamage() > 0 {
			c.mark(0)
		}
	case 2:
		// 目标：Bot 的直线攻击落空，且该回合玩家没有受伤事件。
		if a.GetAttacker() == c.bot && a.GetAction() == match.ActionAttack && a.GetDodged() {
			c.dodgedThisRound = true
		}
		if c.dodgedThisRound && !c.hurtThisRound {
			c.mark(0)
		}
	case 3:
		// 目标 1：玩家挂盾吃下 Bot 的一次攻击；目标 2：玩家释放一次技能。
		if a.GetAttacker() == c.bot && a.GetAction() == match.ActionAttack &&
			a.GetShielded() && a.GetDamage() > 0 {
			c.mark(0)
		}
		if a.GetAttacker() == c.player && a.GetAction() == match.ActionSkill {
			c.mark(1)
		}
	}
}

func (c *Checker) mark(i int) {
	if i < 0 || i >= len(c.goals) {
		return
	}
	c.goals[i].Done = true
	for _, g := range c.goals {
		if !g.Done {
			return
		}
	}
	c.done = true
}

// State：当前状态的快照（切片深拷贝，调用方修改不影响判定器）。
func (c *Checker) State() State {
	goals := make([]GoalState, len(c.goals))
	copy(goals, c.goals)
	slots := make([]string, len(c.level.Slots))
	copy(slots, c.level.Slots)
	next := 0
	if c.level.N < MaxLevel {
		next = c.level.N + 1
	}
	return State{
		Level:        c.level.N,
		Title:        c.level.Title,
		Mechanic:     c.level.Mechanic,
		Slots:        slots,
		Goals:        goals,
		Completed:    c.done,
		NextLevel:    next,
		AllCompleted: c.done && c.level.N == MaxLevel,
		Cap:          c.cap,
		CapCoeff:     match.RiskCoeff(c.cap),
	}
}
