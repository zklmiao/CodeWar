// Package tutorial：新手教程（10 号票，spec 用户故事 5-6）。
//
// 教程是 PvE 管线的一种模式：复用 MatchEngine + 09 号票的 Bot 骨架，
// 本包只提供三样东西——
//   - 固定题包（每关 1-2 道手工题，含服务端内部 AnswerKey，供脚本 Bot 使用）；
//   - 脚本化 Bot 的回合计划（确定性行动，不走难度分布）；
//   - 目标判定器（观察对局事件流，达成后由 hub 写进度）。
//
// 引擎零改动：本包实现 match.QuestionPort，只经 Submit/事件流/Snapshot 交互。
package tutorial

import (
	"fmt"

	"codewar/server/internal/bot"
	"codewar/server/internal/match"
)

// MaxLevel：当前教程关数（第 5 关综合关为可选项，本票未实现）。
const MaxLevel = 4

// Goal：一关的达成条件（服务端判定；客户端只渲染 checkbox）。
type Goal struct {
	ID    string
	Label string
}

// Level：一关的完整定义（题包 + Bot 脚本 + 目标）。
type Level struct {
	N        int
	Title    string
	Mechanic string // 本关机制小结（完成页展示）
	// Slots：本关客户端展示的行动槽（教程只教一个机制，其余槽不渲染）。
	Slots []string
	Goals []Goal
	// 玩家/Bot 的固定职业与技能：教程不引入盲选变量（数值固定才好教学）。
	PlayerClass, PlayerSkill string
	BotClass, BotSkill       string
	Script                   bot.Script
	// Questions：按行动取固定题（四个行动都必须有，引擎每回合会组装整块面板）。
	Questions map[string]match.Question
}

// Levels：四关定义（顺序即解锁顺序）。
var Levels = []Level{
	level1(), level2(), level3(), level4(),
}

// LevelByN：按 1 起的关号取定义。
func LevelByN(n int) (Level, bool) {
	for _, l := range Levels {
		if l.N == n {
			return l, true
		}
	}
	return Level{}, false
}

// Bank：固定题包适配器（实现 match.QuestionPort——引擎只按回合/玩家取题）。
type Bank struct {
	q map[string]match.Question
}

// NewBank：按关号构造题包；未知关号 fail-closed。
func NewBank(level int) (*Bank, error) {
	l, ok := LevelByN(level)
	if !ok {
		return nil, fmt.Errorf("未知教程关号 %d", level)
	}
	return &Bank{q: l.Questions}, nil
}

// AttackQuestion：本关攻击槽固定题（镜像题的教程特例：始终同一实例）。
func (b *Bank) AttackQuestion(int) match.Question { return b.q[match.ActionAttack] }

// MoveQuestion：本关移动槽固定题。
func (b *Bank) MoveQuestion(int, match.Player) match.Question { return b.q[match.ActionMove] }

// DefendQuestion：本关防御槽固定题。
func (b *Bank) DefendQuestion(int, match.Player) match.Question { return b.q[match.ActionDefend] }

// SkillQuestion：本关技能槽固定题（未装备技能的玩家不会消费）。
func (b *Bank) SkillQuestion(int, match.Player) match.Question { return b.q[match.ActionSkill] }

// q：手工题构造器（格式与 question 模板一致：Call 是 Python 表达式，
// Expected 是 repr；AnswerKey 只在服务端内部使用）。
func q(id, statement, signature, starter, answer string, tests ...match.TestCase) match.Question {
	return match.Question{
		ID:          id,
		Statement:   statement,
		Signature:   signature,
		StarterCode: starter,
		Tests:       tests,
		AnswerKey:   answer,
	}
}

// ---------------------------------------------------------------------------
// 第 1 关：读题作答 → 攻击
// ---------------------------------------------------------------------------

func level1() Level {
	double := q(
		"tutorial-L1-attack",
		"欢迎来到 CodeWar！每回合你从四道题里选一道作答，答对才会执行对应行动。\n本关只需解出攻击题：返回输入的两倍。\n示例：double(21) → 42；double(0) → 0",
		"def double(n: int) -> int:",
		"def double(n: int) -> int:\n    # TODO: 返回 n 的两倍\n    return 0\n",
		"def double(n: int) -> int:\n    return n * 2\n",
		match.TestCase{Name: "example-1", Call: "double(21)", Expected: "42"},
		match.TestCase{Name: "example-2", Call: "double(0)", Expected: "0"},
		match.TestCase{Name: "negative", Call: "double(-5)", Expected: "-10"},
	)
	questions := map[string]match.Question{
		match.ActionAttack: double,
		match.ActionMove:   double, // 本关不展示移动槽；Bot 也不使用
		match.ActionDefend: double, // Bot 每回合防御，让结算得以推进（木桩不出手）
		match.ActionSkill:  double, // 玩家未装备技能，引擎不会消费
	}
	return Level{
		N:           1,
		Title:       "第 1 关 · 读题作答 → 攻击",
		Mechanic:    "四个行动槽对应四种行动，一回合只能生效一个。解出攻击题后角色才会出手；解错会自损并锁定 5 秒。",
		Slots:       []string{match.ActionAttack},
		Goals:       []Goal{{ID: "damage", Label: "解出攻击题，产生 1 次伤害"}},
		PlayerClass: "warrior",
		BotClass:    "warrior",
		// 木桩：全程防御，不出手（防御让双方行动齐备、回合正常结算）。
		Script:    bot.Script{Steps: []bot.Step{{Action: match.ActionDefend, Correct: true, SolveMs: 500}}},
		Questions: questions,
	}
}

// ---------------------------------------------------------------------------
// 第 2 关：走位与射程闪避
// ---------------------------------------------------------------------------

func level2() Level {
	neighbor := q(
		"tutorial-L2-move",
		"走位是本作的防御手段：解出移动题后，在左侧网格点选目标格，角色会移动 1-2 格。\n"+
			"本关对手是法师形态，每回合朝正下方打出一条直线。移出这条线即可闪避，本回合不受伤害。\n"+
			"先解出这道移动题（返回朝某方向走一格后的坐标）。\n示例：neighbor(3, 2, 'L') → [3, 1]",
		"def neighbor(row: int, col: int, d: str) -> list:",
		"def neighbor(row: int, col: int, d: str) -> list:\n    # TODO: 按 U/D/L/R 返回 [row, col] 走一格后的坐标\n    return [row, col]\n",
		"def neighbor(row: int, col: int, d: str) -> list:\n"+
			"    if d == 'U':\n        return [row - 1, col]\n"+
			"    if d == 'D':\n        return [row + 1, col]\n"+
			"    if d == 'L':\n        return [row, col - 1]\n"+
			"    return [row, col + 1]\n",
		match.TestCase{Name: "left", Call: "neighbor(3, 2, 'L')", Expected: "[3, 1]"},
		match.TestCase{Name: "right", Call: "neighbor(3, 2, 'R')", Expected: "[3, 3]"},
		match.TestCase{Name: "up", Call: "neighbor(1, 0, 'U')", Expected: "[0, 0]"},
	)
	questions := map[string]match.Question{
		match.ActionMove:   neighbor,
		match.ActionAttack: neighbor,
		match.ActionDefend: neighbor,
		match.ActionSkill:  neighbor,
	}
	return Level{
		N:           2,
		Title:       "第 2 关 · 走位与射程闪避",
		Mechanic:    "攻击有形状与射程：法师的直线攻击只覆盖选定方向上的格子。结算时目标不在覆盖格内即为闪避（无伤害）。",
		Slots:       []string{match.ActionMove},
		Goals:       []Goal{{ID: "dodge", Label: "让 Bot 的直线攻击落空，且本回合未受伤"}},
		PlayerClass: "warrior",
		BotClass:    "mage",
		// 固定朝下出手：玩家必须走出这一列，闪避才可复现、可教学。
		Script: bot.Script{Steps: []bot.Step{{
			Action: match.ActionAttack, Correct: true, SolveMs: 250,
			Target: &match.Targeting{Direction: "down"},
		}}},
		Questions: questions,
	}
}

// ---------------------------------------------------------------------------
// 第 3 关：防御与技能冷却
// ---------------------------------------------------------------------------

func level3() Level {
	defend := q(
		"tutorial-L3-defend",
		"防御题让你在本回合获得 60% 减伤。本关对手先普攻、再释放一次技能。\n"+
			"解出防御题并在对手出手的回合挂盾，观察受伤数值被折减。\n示例：shielded_damage(26) → 10",
		"def shielded_damage(raw: int) -> int:",
		"def shielded_damage(raw: int) -> int:\n    # TODO: 返回 60% 减伤后的承伤（四舍五入）\n    return raw\n",
		"def shielded_damage(raw: int) -> int:\n    return round(raw * 0.4)\n",
		match.TestCase{Name: "example-1", Call: "shielded_damage(26)", Expected: "10"},
		match.TestCase{Name: "example-2", Call: "shielded_damage(20)", Expected: "8"},
		match.TestCase{Name: "skill-hit", Call: "shielded_damage(55)", Expected: "22"},
	)
	skill := q(
		"tutorial-L3-skill",
		"技能槽是高风险高回报的爆发：解出技能题即可释放，释放后进入冷却，面板灰置。\n"+
			"本关请解出技能题并释放一次。冷却规则：第 N 回合释放 → 第 N+CD 回合起才可再次使用。\n"+
			"示例：cooldown_left(3, 1, 3) → 1",
		"def cooldown_left(current_round: int, used_round: int, cd: int) -> int:",
		"def cooldown_left(current_round: int, used_round: int, cd: int) -> int:\n"+
			"    # TODO: 返回还需要等几回合（已就绪返回 0）\n    return 0\n",
		"def cooldown_left(current_round: int, used_round: int, cd: int) -> int:\n"+
			"    left = used_round + cd - current_round\n"+
			"    return left if left > 0 else 0\n",
		match.TestCase{Name: "waiting", Call: "cooldown_left(3, 1, 3)", Expected: "1"},
		match.TestCase{Name: "ready", Call: "cooldown_left(4, 1, 3)", Expected: "0"},
		match.TestCase{Name: "fresh", Call: "cooldown_left(2, 1, 4)", Expected: "3"},
	)
	questions := map[string]match.Question{
		match.ActionDefend: defend,
		match.ActionSkill:  skill,
		// Bot 的普攻/移动使用防御题的参考解：判题只看用例，行动语义由引擎裁决。
		match.ActionAttack: defend,
		match.ActionMove:   defend,
	}
	return Level{
		N:        3,
		Title:    "第 3 关 · 防御与技能冷却",
		Mechanic: "防御行动在本回合结算批次内提供 60% 减伤；技能释放后进入冷却（第 N 回合释放 → 第 N+CD 回合起可用），冷却期间技能槽灰置不可选。",
		Slots:    []string{match.ActionDefend, match.ActionSkill},
		Goals: []Goal{
			{ID: "shielded", Label: "挂盾吃下 Bot 的一次攻击（伤害被护盾折减）"},
			{ID: "skill", Label: "释放一次自己的技能（观察技能槽进入冷却）"},
		},
		PlayerClass: "warrior",
		PlayerSkill: "cleave",
		BotClass:    "warrior",
		BotSkill:    "cleave",
		// 对手先普攻、再技能：给玩家两回合各完成一个目标。
		Script: bot.Script{Steps: []bot.Step{
			{Action: match.ActionAttack, Correct: true, SolveMs: 300},
			{Action: match.ActionSkill, Correct: true, SolveMs: 300},
			{Action: match.ActionAttack, Correct: true, SolveMs: 300},
		}},
		Questions: questions,
	}
}

// ---------------------------------------------------------------------------
// 第 4 关：难度选择与伤害收益
// ---------------------------------------------------------------------------

func level4() Level {
	base := q(
		"tutorial-L4-attack",
		"难度上限不只是题目难：所选档位越高，普攻伤害系数越高（L1 ×0.85 → L5 ×1.20），ELO 结算奖励同理。\n"+
			"本关入口可选 L3 或 L5，命中一次观察伤害差异。先解出这道攻击题。\n示例：total_damage(22, 5) → 27",
		"def total_damage(base: int, bonus: int) -> int:",
		"def total_damage(base: int, bonus: int) -> int:\n    # TODO: 返回 base + bonus\n    return 0\n",
		"def total_damage(base: int, bonus: int) -> int:\n    return base + bonus\n",
		match.TestCase{Name: "example-1", Call: "total_damage(22, 5)", Expected: "27"},
		match.TestCase{Name: "zero", Call: "total_damage(0, 0)", Expected: "0"},
		match.TestCase{Name: "negative", Call: "total_damage(10, -3)", Expected: "7"},
	)
	questions := map[string]match.Question{
		match.ActionAttack: base,
		match.ActionMove:   base,
		match.ActionDefend: base,
		match.ActionSkill:  base,
	}
	return Level{
		N:           4,
		Title:       "第 4 关 · 难度选择与伤害收益",
		Mechanic:    "难度上限同时决定题目区间与风险系数：L3 = ×1.00，L5 = ×1.20（只乘普攻伤害）。选更高难度意味着更难的题，也意味着更疼的拳。",
		Slots:       []string{match.ActionAttack},
		Goals:       []Goal{{ID: "damage", Label: "用攻击题命中 1 次，观察所选难度上限的伤害系数"}},
		PlayerClass: "warrior",
		BotClass:    "warrior",
		Script:      bot.Script{Steps: []bot.Step{{Action: match.ActionDefend, Correct: true, SolveMs: 500}}},
		Questions:   questions,
	}
}
