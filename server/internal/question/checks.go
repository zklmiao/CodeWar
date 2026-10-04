// 不变量校验（checks.invariants，设计 002 §1/§3/§4）：
// 生成时强制执行——陷阱无对应隐藏用例、参考解行数与档位不符、四维标注缺失均为错误。
package question

import (
	"fmt"
	"strings"
)

// Rubric 参考解行数区间（设计 002 §4 的 S 维，机械复核用）。
// 行为参考解函数体逻辑行数（去空行/注释，不含 def 行）。
func answerLinesRange(d Difficulty) (min, max int) {
	switch d {
	case L1:
		return 1, 3
	case L2:
		return 3, 5
	case L3:
		return 5, 8
	case L4:
		return 8, 12
	case L5:
		return 12, 18
	default:
		return 0, 0
	}
}

// ValidateMeta：四维标注齐全合法。
func ValidateMeta(m Meta) error {
	if m.ID == "" {
		return fmt.Errorf("meta 缺少 id")
	}
	if !validTopic(m.Topic) {
		return fmt.Errorf("模板 %s 题型非法：%q", m.ID, m.Topic)
	}
	if m.Difficulty < L1 || m.Difficulty > L5 {
		return fmt.Errorf("模板 %s 难度非法：%v", m.ID, m.Difficulty)
	}
	if !validAction(m.Action) {
		return fmt.Errorf("模板 %s 动作非法：%q", m.ID, m.Action)
	}
	if !validForm(m.Form) {
		return fmt.Errorf("模板 %s 形式非法：%q", m.ID, m.Form)
	}
	return nil
}

// ValidateInstance：生成后校验（陷阱追溯 + 行数 + 用例数 + 双形式）。
func ValidateInstance(in Instance) error {
	if err := ValidateMeta(in.Meta); err != nil {
		return err
	}
	if len(in.VisibleTests) != 2 {
		return fmt.Errorf("模板 %s 可见样例必须恰为 2 个，实际 %d", in.Meta.ID, len(in.VisibleTests))
	}
	if n := len(in.HiddenTests); n < 5 || n > 8 {
		return fmt.Errorf("模板 %s 隐藏用例必须 5-8 个，实际 %d", in.Meta.ID, n)
	}
	if in.AnswerKey == "" {
		return fmt.Errorf("模板 %s 缺少参考解", in.Meta.ID)
	}
	// S 维：参考解行数与档位相符。
	lines := CountAnswerLines(in.AnswerKey)
	lo, hi := answerLinesRange(in.Meta.Difficulty)
	if lines < lo || lines > hi {
		return fmt.Errorf("模板 %s 参考解 %d 行与档位 %v 期望 [%d,%d] 不符",
			in.Meta.ID, lines, in.Meta.Difficulty, lo, hi)
	}
	// 陷阱追溯：每个声明陷阱至少映射一个隐藏用例（按用例名归一化包含判定）。
	names := make([]string, 0, len(in.HiddenTests))
	for _, tc := range in.HiddenTests {
		names = append(names, tc.Name)
	}
	for _, trap := range in.Meta.Traps {
		if !trapCoveredCases(trap, names) {
			return fmt.Errorf("模板 %s 陷阱 %q 无对应隐藏用例", in.Meta.ID, trap)
		}
	}
	// 双形式：both 模板必须提供骨架，挖空 1-3 行。
	if in.Meta.Form == FormBoth {
		if in.Skeleton == "" || in.BlankLines < 1 || in.BlankLines > 3 {
			return fmt.Errorf("模板 %s 双渲染骨架挖空必须 1-3 行，实际 %d", in.Meta.ID, in.BlankLines)
		}
		if !in.BlankIsCore {
			return fmt.Errorf("模板 %s 挖空行必须为逻辑核心行", in.Meta.ID)
		}
	}
	if in.Statement == "" || in.Signature == "" || in.CompleteStarter == "" {
		return fmt.Errorf("模板 %s 题面/签名/起始代码缺失", in.Meta.ID)
	}
	return nil
}

// CountAnswerLines：参考解逻辑行数（去空行/注释/装饰器，不含 def 行）。
func CountAnswerLines(answer string) int {
	n := 0
	for _, ln := range strings.Split(answer, "\n") {
		t := strings.TrimSpace(ln)
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "@") {
			continue
		}
		if strings.HasPrefix(t, "def ") {
			continue
		}
		n++
	}
	return n
}

// trapCoveredCases：对隐藏用例名切片的陷阱覆盖判定。
func trapCoveredCases(trap string, names []string) bool {
	norm := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, "_", "-")
		return s
	}
	t := norm(trap)
	for _, nm := range names {
		if strings.Contains(norm(nm), t) {
			return true
		}
	}
	return false
}

func validTopic(t Topic) bool {
	for _, a := range AllTopics {
		if a == t {
			return true
		}
	}
	return false
}

func validAction(a Action) bool {
	switch a {
	case ActionAttack, ActionMove, ActionDefend, ActionSkill:
		return true
	}
	return false
}

func validForm(f Form) bool {
	switch f {
	case FormComplete, FormFillBlank, FormBoth:
		return true
	}
	return false
}
