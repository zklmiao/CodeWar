package match

import (
	"context"
	"time"

	"codewar/server/gen/matchpb"
)

// 本票战斗数值（spec 附注初稿；调参以 spec 表为准）
const (
	BaseDamage = 22  // 普攻基础伤害
	RoundMs    = 75000
	LockMs     = 5000
	SelfDamage = 3
	MaxRounds  = 10
	// 时间系数：solveMs ∈ [0, RoundMs] 线性映射到 [1.0, CoeffMin]
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
type TestCase struct {
	Name     string
	Call     string
	Expected string
}

// Question：镜像题实例。本票由硬编码桩提供；04 号票换成 QuestionPort 背后的出题工厂。
type Question struct {
	ID          string
	Statement   string
	Signature   string
	StarterCode string
	Tests       []TestCase
}

// QuestionPort：按回合取攻击镜像题（本票每轮轮换，双方同实例）。
type QuestionPort interface {
	AttackQuestion(round int) Question
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

var _ = matchpb.MatchPhase_MATCH_PHASE_WAITING // 引用生成包，防 import 空置
