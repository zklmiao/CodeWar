// Package bot：PvE 模拟对手（09 号票，spec 用户故事 32-33）。
//
// 架构边界（架构师裁定）：Bot 是 hub 层的独立组件，不进入 match 包——
// 对引擎的行动接入面只有 engine.Submit（SubmissionPort 的同一实现路径），
// 观测面只有 engine.Snapshot / engine.SetTargeting（与真人客户端同一套 API）。
// match 包内不存在任何 bot 相关标识符或分支（测试用 grep 钉住）。
package bot

import "strings"

// Difficulty：PvE 难度档（URL ?bot=easy|normal|hard）。
type Difficulty string

const (
	Easy   Difficulty = "easy"
	Normal Difficulty = "normal"
	Hard   Difficulty = "hard"
)

// DefaultDifficulty：bot 参数缺省/非法时的回退档（与 hub 的 WARN 回退语义一致）。
const DefaultDifficulty = Normal

// ALL：三档的稳定顺序（UI/统计用）。
var All = []Difficulty{Easy, Normal, Hard}

// ParseDifficulty：宽松解析（大小写/空白不敏感）；空或非法 → false。
func ParseDifficulty(s string) (Difficulty, bool) {
	switch Difficulty(strings.ToLower(strings.TrimSpace(s))) {
	case Easy:
		return Easy, true
	case Normal:
		return Normal, true
	case Hard:
		return Hard, true
	}
	return "", false
}

// Label：中文展示名（对局内"机器人（难度）"标识）。
func (d Difficulty) Label() string {
	switch d {
	case Easy:
		return "简单"
	case Hard:
		return "困难"
	default:
		return "普通"
	}
}

// Weights：行动权重（攻击/移动/防御/技能；技能冷却中时技能权重并入攻击）。
type Weights struct {
	Attack int
	Move   int
	Defend int
	Skill  int
}

// Params：一档 Bot 的行为参数。
// 初稿数值：spec 开放问题 3「PvE Bot 解题耗时分布标定」——集中在本表，
// 待 playtest 标定后只改这里（禁止散落到 hub/引擎）。
type Params struct {
	MinSolveMs int64   // 解题耗时窗下界（回合开始→提交）
	MaxSolveMs int64   // 上界
	Accuracy   float64 // 正确率（错误分支 = starter 代码，走真实判题失败，自损 3 + 锁 5s）
	Weights    Weights
}

// 三档初稿参数（spec 开放问题 3，待 playtest 标定）。
const (
	easyMinSolveMs   int64 = 45000
	easyMaxSolveMs   int64 = 70000
	easyAccuracy           = 0.55
	easyAttackWeight       = 60
	easyMoveWeight         = 15
	easyDefendWeight       = 15
	easySkillWeight        = 10

	normalMinSolveMs   int64 = 25000
	normalMaxSolveMs   int64 = 45000
	normalAccuracy           = 0.75
	normalAttackWeight       = 70
	normalMoveWeight         = 10
	normalDefendWeight       = 10
	normalSkillWeight        = 10

	hardMinSolveMs   int64 = 12000
	hardMaxSolveMs   int64 = 25000
	hardAccuracy           = 0.90
	hardAttackWeight       = 75
	hardMoveWeight         = 5
	hardDefendWeight       = 10
	hardSkillWeight        = 10
)

// ParamsFor：三档初稿（spec 开放问题 3，待 playtest 标定）。
//
//	easy   45~70s  正确率 55%  攻60/移15/防15/技10
//	normal 25~45s  正确率 75%  攻70/移10/防10/技10
//	hard   12~25s  正确率 90%  攻75/移5/防10/技10
func ParamsFor(d Difficulty) Params {
	switch d {
	case Easy:
		return Params{MinSolveMs: easyMinSolveMs, MaxSolveMs: easyMaxSolveMs, Accuracy: easyAccuracy,
			Weights: Weights{Attack: easyAttackWeight, Move: easyMoveWeight, Defend: easyDefendWeight, Skill: easySkillWeight}}
	case Hard:
		return Params{MinSolveMs: hardMinSolveMs, MaxSolveMs: hardMaxSolveMs, Accuracy: hardAccuracy,
			Weights: Weights{Attack: hardAttackWeight, Move: hardMoveWeight, Defend: hardDefendWeight, Skill: hardSkillWeight}}
	default: // Normal
		return Params{MinSolveMs: normalMinSolveMs, MaxSolveMs: normalMaxSolveMs, Accuracy: normalAccuracy,
			Weights: Weights{Attack: normalAttackWeight, Move: normalMoveWeight, Defend: normalDefendWeight, Skill: normalSkillWeight}}
	}
}

// isZero：Params 零值判定（New 里用 ParamsFor 兜底；测试可注入自定义 Params）。
func (p Params) isZero() bool {
	return p.MinSolveMs == 0 && p.MaxSolveMs == 0 && p.Accuracy == 0 &&
		p.Weights == Weights{}
}
