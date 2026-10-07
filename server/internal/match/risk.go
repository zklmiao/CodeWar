// 难度风险系数（05 号票）：所选难度上限档挂钩全局普攻伤害与 ELO 结算奖励——
// 难度选择从 UI 功能升格为风险回报战术决策（spec 001 用户故事 9）。
// 数值来源：spec 001 附注「难度系数 L1 ×0.85 → L5 ×1.20」。
package match

// 风险系数表锚点（spec 附注；调参改这里与 tests/risk 断言同步）。
const (
	RiskCoeffMin = 0.85 // L1（下限）
	RiskCoeffMax = 1.20 // L5（上限）
	// DefaultRiskTier：cap 缺省档 L3。它同时是 04 出题区间锚点（question.Config
	// 默认 cap=L3，不得改默认值）与伤害/结算的 1.0 锚点——无配置对局（匿名
	// 缺省链路）行为与 05 之前逐字节一致（05 裁决 5 的向后兼容硬性）。
	DefaultRiskTier = 3
)

// RiskCoeff：难度上限档 → 风险系数，分段线性插值、L3 锚定 1.0：
//
//	L1..L3：0.85 + 0.075×(tier−1)  →  0.85 / 0.925 / 1.0
//	L3..L5：1.0  + 0.10×(tier−3)   →  1.0 / 1.10 / 1.20
//
// 端点严格取 spec 附注的 0.85/1.20；L3=1.0 是向后兼容锚点（05 裁决 2 写作
// 「0.85 + 0.35×(tier−1)/4, L3=1.0」，但该式在 L3 得 1.025，与同一裁决的
// 「无参数 = cap L3 → 行为与 main 逐字节等价」矛盾——以锚点语义为准取分段线性，
// 两端点与 L3 锚均满足）。
// tier ≤ 0 视为未配置 → L3；越界钳制到 [1,5]。
func RiskCoeff(tier int) float64 {
	if tier <= 0 {
		tier = DefaultRiskTier
	}
	if tier < 1 {
		tier = 1
	}
	if tier > 5 {
		tier = 5
	}
	if tier <= DefaultRiskTier {
		return RiskCoeffMin + (1.0-RiskCoeffMin)*float64(tier-1)/float64(DefaultRiskTier-1)
	}
	return 1.0 + (RiskCoeffMax-1.0)*float64(tier-DefaultRiskTier)/float64(5-DefaultRiskTier)
}
