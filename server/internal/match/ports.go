package match

import (
	"context"
	"time"
)

// 战斗时长常量（职业/技能数值已迁往 classes 注册表单一来源，review 发现 4）
const (
	// 时长一律用 time.Duration 类型常量，杜绝裸 int 手工换算（review 修复：
	// 散落的 * time.Millisecond 是 timer 单位 bug 的成因）
	RoundDuration = 75 * time.Second // 回合倒计时
	LockDuration  = 5 * time.Second  // 错误提交锁定

	SelfDamage = 3
	MaxRounds  = 10
	// 时间系数：solveMs ∈ [0, RoundDuration] 线性映射到 [1.0, CoeffMin]
	CoeffMin  = 0.6
	CoeffSpan = 0.4
)

// 玩家标识：本票固定 "p1"/"p2"
type Player = string

const (
	P1 Player = "p1"
	P2 Player = "p2"
)

// TestCase：判题用例（call 为 Python 表达式，expected 为期望 repr）。
// 15 号票契约 v2：Setup/Compare 为可选的题作者声明（空串 = 未声明，走现行
// eval/repr 路径）——setup 在玩家代码 import 后、call 求值前 exec 进玩家
// 命名空间（可用沙箱构造辅助库）；compare 是 `def _normalize(got)` 的函数
// 源码，worker 用它把 got 规范化为 repr 字符串，值比对仍在宿主（expected
// 永不进容器）。见 docs/adr/0004。
type TestCase struct {
	Name     string
	Call     string
	Expected string
	Setup    string
	Compare  string
}

// Targeting：攻击/技能的目标选择载荷（06 号票，设计 003 §1 方向选择全局规则）。
// proto SetTargeting/SubmitRequest.targeting 同构；几何解释见 shapes.go。
type Targeting struct {
	Direction string // up | down | left | right；空 = 未指定（结算时自动瞄准）
	Distance  int    // 点狙类在方向上的格距；0 = 未指定
}

// Question：题实例。04 号票起由出题工厂按 (spec, seed) 生成；题面/用例格式
// 与判题器对接契约：call 为 Python 表达式，expected 为期望 repr。
type Question struct {
	ID          string
	Statement   string
	Signature   string
	StarterCode string
	Tests       []TestCase
	// AnswerKey：服务端内部参考解（09 号票，PvE 模拟对手的"正确分支"来源）。
	// 只存在于引擎内部结构：proto Problem 没有对应字段，problemToPb 不下发，
	// 客户端永远不可见。引擎本身忽略该字段（任何玩家都不能通过提交它获得特权）。
	AnswerKey string
}

// QuestionPort：按回合取题（06 号票扩展技能槽）。
//   - AttackQuestion：攻击镜像题（双方同实例，seed = f(matchSeed, round)）。
//   - Move/DefendQuestion：私有行动题（双方独立，seed = f(matchSeed, player, round)）。
//   - SkillQuestion：私有技能题（06 号票；难度 = 所选上限档，不走爬坡/均匀策略——
//     实现见 question.Factory；仅装备了技能位的玩家会被消费）。
type QuestionPort interface {
	AttackQuestion(round int) Question
	MoveQuestion(round int, player Player) Question
	DefendQuestion(round int, player Player) Question
	SkillQuestion(round int, player Player) Question
}

// JudgeResult：判题结果。Category 取值见 proto judge.proto 注释。
type JudgeResult struct {
	Passed     bool
	Category   string
	DurationMs int32
	Detail     string
}

// JudgePort：判题 seam。生产实现走 gRPC（judgeclient）；测试注入 fake。
type JudgePort interface {
	Judge(ctx context.Context, code string, tests []TestCase) JudgeResult
}

// ClockPort：回合计时权威（ADR-0002）。测试注入可拨动的假钟。
type Clock interface {
	Now() time.Time
	NewTimer(d time.Duration) Timer
}

type Timer interface {
	C() <-chan time.Time
	Stop() bool
}

// 结算原因（proto MatchResult.reason）
const (
	ReasonKO        = "ko"
	ReasonSelfKO    = "self-ko"
	ReasonRoundsCap = "rounds-cap"
)

// 判题类别
const (
	CategoryOK          = "ok"
	CategoryWrongAnswer = "wrong-answer"
	CategoryLocked      = "locked"
	CategoryTimeout     = "timeout"
	CategorySyntax      = "syntax"
	CategoryRuntime     = "runtime-error"
	CategoryJudgeError  = "judge-error"
)
