// 不变量负路径测试（同包白盒）：故意违规的实例必须被校验函数拦截，
// 且错误信息可定位到违规项（陷阱名 / 档位 / 挖空）。
package question

import (
	"strings"
	"testing"

	"codewar/server/internal/match"
)

// baseValid：构造校验通过的基准实例（调用方再注入单项违规）。
func baseValid() Instance {
	vis := []match.TestCase{{Name: "v1", Call: "f(0)", Expected: "0"}, {Name: "v2", Call: "f(1)", Expected: "1"}}
	hidden := []match.TestCase{
		{Name: "empty-input", Call: "f([])", Expected: "[]"},
		{Name: "h2", Call: "f([1])", Expected: "[1]"},
		{Name: "h3", Call: "f([1, 2])", Expected: "[1, 2]"},
		{Name: "h4", Call: "f([2, 1])", Expected: "[2, 1]"},
		{Name: "h5", Call: "f([3, 3])", Expected: "[3]"},
	}
	return Instance{
		Meta:            Meta{ID: "neg-case", Topic: TopicSyntax, Difficulty: L2, Action: ActionAttack, Form: FormComplete},
		Statement:       "题面",
		Signature:       "def f(x: list) -> list:",
		CompleteStarter: RenderComplete("def f(x: list) -> list:"),
		VisibleTests:    vis,
		HiddenTests:     hidden,
		AnswerKey:       "def f(x: list) -> list:\n    out = []\n    for v in x:\n        out.append(v)\n    return out\n",
	}
}

// a. 陷阱无覆盖：声明陷阱在隐藏用例中无映射 → 报错且指明陷阱名。
func TestRejectTrapWithoutCoverage(t *testing.T) {
	in := baseValid()
	in.Meta.Traps = []string{"ghost-trap"}
	err := ValidateInstance(in)
	if err == nil {
		t.Fatal("声明陷阱无对应隐藏用例，校验应报错")
	}
	if !strings.Contains(err.Error(), "ghost-trap") {
		t.Fatalf("错误信息应指明陷阱名，实际：%v", err)
	}
}

// b. 行数越档：3 行参考解标注 L4（L4 要求 8-12 行）→ rubric 机械复核报错。
func TestRejectAnswerLinesMismatch(t *testing.T) {
	in := baseValid()
	in.Meta.Difficulty = L4
	in.AnswerKey = "def f(x: list) -> list:\n    out = []\n    out += x\n    return out\n"
	err := ValidateInstance(in)
	if err == nil {
		t.Fatal("参考解行数与档位不符，校验应报错")
	}
	if !strings.Contains(err.Error(), "L4") {
		t.Fatalf("错误信息应指明档位 L4，实际：%v", err)
	}
}

// c. 挖空越界：both 模板挖空 4 个核心行 → 校验报错（上限 3）。
func TestRejectTooManyBlanks(t *testing.T) {
	in := baseValid()
	in.Meta.Form = FormBoth
	in.Meta.Difficulty = L3
	in.AnswerKey = "def f(x: list) -> list:\n    out = []\n    for v in x:\n        out.append(v)\n        out.append(v)\n    if len(out) == 0:\n        return []\n    return out\n"
	in.Skeleton = "def f(x: list) -> list:\n    # ____ 填空 ____\n"
	in.BlankLines = 4
	in.BlankIsCore = true
	err := ValidateInstance(in)
	if err == nil {
		t.Fatal("挖空 4 行超出上限 3，校验应报错")
	}
	if !strings.Contains(err.Error(), "1-3") {
		t.Fatalf("错误信息应指明挖空上限，实际：%v", err)
	}
}
