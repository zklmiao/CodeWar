// 模板共享构造器（减少重复，保持用例命名/陷阱追溯一致）。
package templates

import (
	"fmt"
	"strings"

	"codewar/server/internal/match"
	"codewar/server/internal/question"
)

// funcTemplate：闭包式模板（meta 声明 + generate 纯函数）。
type funcTemplate struct {
	meta question.Meta
	gen  func(rng *question.RNG) (question.Instance, error)
}

func (t funcTemplate) Meta() question.Meta { return t.meta }

func (t funcTemplate) Generate(rng *question.RNG) (question.Instance, error) {
	return t.gen(rng)
}

// caseOf：构造判题用例（call = Python 表达式，expected = repr）。
func caseOf(name, call, expected string) match.TestCase {
	return match.TestCase{Name: name, Call: call, Expected: expected}
}

// caseAlias：用例简写（toCases 转 match.TestCase）。
type caseAlias struct {
	name     string
	call     string
	expected string
}

// cases：用例简写转 match.TestCase（变参，避免复合字面量选择器的括号问题）。
func cases(cs ...caseAlias) []match.TestCase {
	out := make([]match.TestCase, len(cs))
	for i, c := range cs {
		out[i] = caseOf(c.name, c.call, c.expected)
	}
	return out
}

// pyStr：Python 字符串 repr（单引号风格）。
func pyStr(s string) string {
	r := strings.ReplaceAll(s, "\\", "\\\\")
	r = strings.ReplaceAll(r, "'", "\\'")
	return "'" + r + "'"
}

// pyList：整数列表 Python repr。
func pyList(v []int) string {
	var b strings.Builder
	b.WriteString("[")
	for i, x := range v {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%d", x)
	}
	b.WriteString("]")
	return b.String()
}

// pyStrList：字符串列表 Python repr。
func pyStrList(v []string) string {
	var b strings.Builder
	b.WriteString("[")
	for i, x := range v {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(pyStr(x))
	}
	b.WriteString("]")
	return b.String()
}

// bareInstance：单形式模板（complete-only，如移动/防御/技能题）的 Instance 组装。
// 双渲染字段留空；ValidateInstance 仅对 both 模板校验挖空。
func bareInstance(meta question.Meta, statement, signature, answer string, visible, hidden []match.TestCase) (question.Instance, error) {
	return question.Instance{
		Meta:            meta,
		Statement:       statement,
		Signature:       signature,
		CompleteStarter: question.RenderComplete(signature),
		VisibleTests:    visible,
		HiddenTests:     hidden,
		AnswerKey:       answer,
	}, nil
}

// examplesText：题面可见样例段（2 个，用例文本融入 statement，proto 不动）。
func examplesText(visible []match.TestCase) string {
	var b strings.Builder
	for _, tc := range visible {
		fmt.Fprintf(&b, "示例：%s → %s；", tc.Call, tc.Expected)
	}
	return strings.TrimSuffix(b.String(), "；")
}

// assemble：组装 Instance（含双渲染骨架与核心行标记）。
// bodyLines 为参考解函数体行（含缩进）；blankIdx 为挖空下标（必须核心行且 ≤3）。
func assemble(meta question.Meta, seedNote, statement, signature string, bodyLines []string, blankIdx map[int]bool, visible, hidden []match.TestCase, answer string) (question.Instance, error) {
	// 校验挖空合法性（与 checks.ValidateInstance 一致，早失败）。
	if meta.Form == question.FormBoth {
		if len(blankIdx) < 1 || len(blankIdx) > 3 {
			return question.Instance{}, fmt.Errorf("模板 %s 挖空必须 1-3 行", meta.ID)
		}
		for i := range blankIdx {
			if i < 0 || i >= len(bodyLines) || !question.IsCoreLine(bodyLines[i]) {
				return question.Instance{}, fmt.Errorf("模板 %s 挖空行 %d 非核心行", meta.ID, i)
			}
		}
	}
	skel, n := question.RenderFillBlank(signature, bodyLines, blankIdx)
	core := true
	for i := range blankIdx {
		if !question.IsCoreLine(bodyLines[i]) {
			core = false
		}
	}
	return question.Instance{
		Meta:            meta,
		Seed:            seedNote,
		Statement:       statement,
		Signature:       signature,
		CompleteStarter: question.RenderComplete(signature),
		Skeleton:        skel,
		BlankLines:      n,
		BlankIsCore:     core,
		VisibleTests:    visible,
		HiddenTests:     hidden,
		AnswerKey:       answer,
	}, nil
}
