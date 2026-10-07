// Package question：出题工厂与模板管线（04 号票）。
// 依据：docs/design/002-question-system.md §1-§5；spec 001「实现决策/出题系统」。
// 本包是 QuestionFactory 纯函数 seam：create(spec, seed) → instance。
// 随机唯一来源是种子派生的 RNG 流，禁止使用全局 rand。
package question

import (
	"fmt"

	"codewar/server/internal/match"
)

// 题型：spec 用户故事 7 的 6 大题型（英文 slug，中文标签见 TopicLabel）。
type Topic string

const (
	TopicSyntax    Topic = "syntax"    // 语法与运算
	TopicString    Topic = "string"    // 字符串处理
	TopicArray     Topic = "array"     // 数组与列表
	TopicHash      Topic = "hash"      // 哈希与计数
	TopicSort      Topic = "sort"      // 排序与贪心
	TopicRecursion Topic = "recursion" // 递归与简单 DP
)

// AllTopics：全部 6 题型（默认 Config 不过滤时使用）。
var AllTopics = []Topic{TopicSyntax, TopicString, TopicArray, TopicHash, TopicSort, TopicRecursion}

// TopicLabel：题型中文标签（题面展示用）。
var TopicLabel = map[Topic]string{
	TopicSyntax:    "语法与运算",
	TopicString:    "字符串处理",
	TopicArray:     "数组与列表",
	TopicHash:      "哈希与计数",
	TopicSort:      "排序与贪心",
	TopicRecursion: "递归与简单 DP",
}

// 难度档：L1~L5（设计 002 §4 Rubric）。
type Difficulty int

const (
	L1 Difficulty = 1
	L2 Difficulty = 2
	L3 Difficulty = 3
	L4 Difficulty = 4
	L5 Difficulty = 5
)

// String：L1~L5 文本形式。
func (d Difficulty) String() string { return fmt.Sprintf("L%d", int(d)) }

// 动作类型：attack | move | defend | skill。
type Action string

const (
	ActionAttack Action = "attack"
	ActionMove   Action = "move"
	ActionDefend Action = "defend"
	ActionSkill  Action = "skill"
)

// 题面形式：complete（补全函数体）| fill-blank（填空 1-3 行）| both（攻击模板默认）。
type Form string

const (
	FormComplete  Form = "complete"
	FormFillBlank Form = "fill-blank"
	FormBoth      Form = "both"
)

// Spec：出题请求（题型×难度档×动作类型×题面形式）。
// Topic 为空表示不限制题型（行动题跨题型抽取，spec「出题系统」）。
type Spec struct {
	Topic      Topic
	Difficulty Difficulty
	Action     Action
	Form       Form
}

// Meta：模板声明式元信息（工具可索引、可审计，设计 002 §1）。
type Meta struct {
	ID         string
	Topic      Topic
	Difficulty Difficulty
	Action     Action
	Form       Form
	Traps      []string // 边界陷阱声明，必须与隐藏用例一一对应（checks 强制）
}

// Instance：工厂产物（MatchEngine 与 Judge 的消费单元，设计 002 §1）。
// VisibleTests 固定 2 个（题面自解释），HiddenTests 5-8 个（边界密集）。
type Instance struct {
	Meta            Meta
	Seed            string // 实例种子（派生链可追溯）
	Statement       string // 题面（含 2 个可见样例文本）
	Signature       string // 函数签名
	CompleteStarter string // complete 形式起始代码
	Skeleton        string // fill-blank 形式骨架（挖空 ≤3 行）
	BlankLines      int    // 挖空行数
	BlankIsCore     bool   // 挖空是否为逻辑核心行
	VisibleTests    []match.TestCase
	HiddenTests     []match.TestCase
	AnswerKey       string // 参考解（完整函数体）
}

// Tests：判题用例全集（可见 + 隐藏，全部通过才生效）。
func (in Instance) Tests() []match.TestCase {
	out := make([]match.TestCase, 0, len(in.VisibleTests)+len(in.HiddenTests))
	out = append(out, in.VisibleTests...)
	out = append(out, in.HiddenTests...)
	return out
}

// ToQuestion：转 match.Question（引擎消费格式，与 stub.go 契约一致）。
// form 为实际渲染形式；both 模板按请求任选其一，单形式模板忽略请求。
func (in Instance) ToQuestion(form Form) match.Question {
	starter := in.CompleteStarter
	id := in.Meta.ID + "@" + shortSeed(in.Seed)
	if in.Meta.Form == FormBoth {
		if form != FormFillBlank {
			form = FormComplete
		}
		if form == FormFillBlank {
			starter = in.Skeleton
		}
		id += ":" + string(form)
	}
	return match.Question{
		ID:          id,
		Statement:   in.Statement,
		Signature:   in.Signature,
		StarterCode: starter,
		Tests:       in.Tests(),
	}
}

// Template：模板接口（声明式 meta + 代码生成器，设计 002 §1 混合形态）。
// Generate 为纯函数：同 RNG 流必产同实例。
type Template interface {
	Meta() Meta
	Generate(rng *RNG) (Instance, error)
}

// shortSeed：种子缩写（Question ID 后缀，避免过长）。
func shortSeed(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12]
}
