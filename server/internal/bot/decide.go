package bot

import (
	"codewar/server/internal/classes"
	"codewar/server/internal/match"
)

// decisionRNG：决策随机源（*math/rand.Rand 满足；单测可注入边界序列）。
// 只暴露两个方法，保证决策函数不依赖具体随机实现。
type decisionRNG interface {
	Float64() float64
	Intn(n int) int
}

// Decision：一次回合行动的纯函数产物。
type Decision struct {
	Action  string // attack | move | defend | skill
	Correct bool   // true = 提交参考解；false = 提交 starter（真实判题失败）
	SolveMs int64  // 解题耗时（调度延迟；引擎用同一时钟测量）
	// Target：脚本模式下可覆盖自动瞄准（普通难度 Bot 恒为 nil，语义不变）。
	Target *match.Targeting
}

// Decide：从参数+RNG 采样一次行动（纯函数；调用顺序固定以保证同 seed 可复现：
// 先耗时、再正确率、最后行动权重）。
//
// skillReady=false（冷却中/未装备）时技能权重并入攻击（架构师裁定 3）。
func Decide(p Params, r decisionRNG, skillReady bool) Decision {
	// 显式分行：RNG 调用顺序 = 耗时 → 正确率 → 行动，复用 fixed seed 大样本可复现。
	ms := sampleSolveMs(p, r)
	correct := r.Float64() < p.Accuracy
	action := pickAction(p.Weights, r, skillReady)
	return Decision{Action: action, Correct: correct, SolveMs: ms}
}

// sampleSolveMs：[Min, Max] 均匀采样（含两端；Float64=0 → Min，趋近 1 → Max）。
func sampleSolveMs(p Params, r decisionRNG) int64 {
	lo, hi := p.MinSolveMs, p.MaxSolveMs
	if lo < 0 {
		lo = 0
	}
	if hi < lo {
		hi = lo
	}
	span := hi - lo + 1
	ms := lo + int64(r.Float64()*float64(span))
	if ms > hi {
		ms = hi
	}
	return ms
}

// pickAction：按权重抽取行动。技能不可用时其权重全部并入攻击——
// 保证冷却回合仍保持"攻击为主"的行为轮廓，而不是出现无行动空档。
func pickAction(w Weights, r decisionRNG, skillReady bool) string {
	attackW := w.Attack
	skillW := 0
	if skillReady {
		skillW = w.Skill
	} else {
		attackW += w.Skill
	}
	total := attackW + w.Move + w.Defend + skillW
	if total <= 0 {
		return match.ActionAttack
	}
	n := r.Intn(total)
	switch {
	case n < attackW:
		return match.ActionAttack
	case n < attackW+w.Move:
		return match.ActionMove
	case n < attackW+w.Move+w.Defend:
		return match.ActionDefend
	default:
		return match.ActionSkill
	}
}

// ===== 超距接近（19 号票：PvE Bot 超射程自我放逐的决策层修复） =====
//
// 缺陷：随机"移动"可能走出攻击射程，此后每回合攻击被引擎自动瞄准的距离
// 钳制（min(射程, 曼哈顿距离)）打在错误格上——dodged=true 永久落空，而随机
// 移动从不偏好接近，对局空转到回合上限。修复只在 Bot 决策层：超距回合的
// 移动落点确定性"接近"，引擎的瞄准钳制语义（玩家侧走位博弈）零改动。

// manhattan：棋盘曼哈顿距离（与引擎 coverage/移动规则同量纲，裁决 2）。
func manhattan(a, b match.Pos) int {
	return abs(a.Row-b.Row) + abs(a.Col-b.Col)
}

// attackReach：普攻射程，按职业形状折算（设计 003 §2 字面值口径）：
// 点狙取 AttackRange（游侠 3）、线扫取 AttackDepth（法师 2）、相邻近战 = 1（剑士）。
func attackReach(c classes.Class) int {
	if c.AttackShape == classes.ShapeSnipe && c.AttackRange > 0 {
		return int(c.AttackRange)
	}
	if c.AttackShape == classes.ShapeLine && c.AttackDepth > 0 {
		return int(c.AttackDepth)
	}
	return 1
}

// skillReach：技能攻击射程（点狙取 RangeMax、线扫/横带取 Depth）。
// 邻域/自身形状无距离语义 → 0：治疗与贴身技能不会放大攻击射程。
func skillReach(s classes.Skill) int {
	if s.RangeMax > 0 {
		return int(s.RangeMax)
	}
	return int(s.Depth)
}

// EffectiveReach：本回合可用的最大攻击距离（超距判定基准，裁决 1）。
// 普攻射程按职业；已装备技能且技能就绪时取 max(普攻, 技能)——
// 穿云箭（射程 4）就绪的游侠按 4 判定，风行回复不放大射程。
func EffectiveReach(c classes.Class, s classes.Skill, hasSkill, skillReady bool) int {
	reach := attackReach(c)
	if hasSkill && skillReady {
		if sr := skillReach(s); sr > reach {
			reach = sr
		}
	}
	return reach
}

// ApproachMove：超距回合的接近落点（确定性纯函数，不消耗随机流，裁决 3）。
// 三层取法的语义出处 = ADR-0007（架构师已裁定"跨回合最大进度"读法）。
//
// 方向 = 朝对手的主轴（|Δrow| ≥ |Δcol| 取行轴，否则列轴，与引擎自动瞄准
// 同构，裁决 2）；候选 = LegalMoves 中该方向上的合法落点——边界、终点被占、
// 距离 2 不穿人规则以 LegalMoves 为单一来源（其与引擎 validateMove 的同语义
// 由 TestLegalMovesAcceptedByEngine 钉住），本函数不另抄一份。选择规则：
//   - 存在"移动后与对手距离 ≤ reach"的落点时，取步数最小的一个——进入射程
//     即停在射程边缘，不贴脸（保持设计 003 §5 的走位博弈语义）；
//   - 整段轴都进不了射程（对手太远、一格接近不了）时，取移动后距离最小的
//     落点做最大进度（同距取较小步数，不越过对手）——跨回合收敛进射程；
//   - 轴向被完全封死（无合法落点）时返回 false，调用方退化现行随机移动。
func ApproachMove(from, opp match.Pos, moveMax, reach int) (match.Pos, bool) {
	dir, _ := aimAxis(from, opp)
	dr, dc := delta(dir)
	best, bestDist := match.Pos{}, -1
	var inRange match.Pos
	hasInRange := false
	for _, to := range LegalMoves(from, opp, moveMax) {
		sr, sc := to.Row-from.Row, to.Col-from.Col
		// 只保留主轴方向上的落点（主轴分量同号且另一分量为零）；
		// LegalMoves 按方向升序产出，步数最小者优先的取法在此天然成立。
		if !((dc == 0 && sc == 0 && sr*dr > 0) || (dr == 0 && sr == 0 && sc*dc > 0)) {
			continue
		}
		d := manhattan(to, opp)
		if d <= reach && !hasInRange {
			inRange, hasInRange = to, true
		}
		if bestDist < 0 || d < bestDist {
			best, bestDist = to, d
		}
	}
	if hasInRange {
		return inRange, true
	}
	if bestDist < 0 {
		return match.Pos{}, false
	}
	return best, true
}

// ChooseMove：行动选择处的移动落点决策（19 号票修复点）。
//
// 与对手距离 > 可用攻击射程时，本回合移动升级为"接近"（确定性落点，不消耗
// 随机流——随机流的其余消费序列不变，裁决 3）；轴向被封死无合法接近格、或
// 距离已在射程内时，维持现行随机移动（行为与随机流均不变，裁决 1）。
func ChooseMove(r decisionRNG, from, opp match.Pos, moveMax, reach int) (match.Pos, bool) {
	if manhattan(from, opp) > reach {
		if to, ok := ApproachMove(from, opp, moveMax, reach); ok {
			return to, true
		}
	}
	return PickMove(r, from, opp, moveMax)
}
