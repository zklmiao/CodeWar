package bot

import "codewar/server/internal/match"

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
