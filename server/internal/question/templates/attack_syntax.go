// 攻击题·语法与运算（L1/L2/L3 黄金样板 + L4 十二号票扩容）。
package templates

import (
	"fmt"

	"codewar/server/internal/question"
)

// 语法 L1：两数相加（单语法点，参考解 1 行）。
func syntaxAddL1() question.Template {
	meta := question.Meta{ID: "syntax-add-L1-attack", Topic: question.TopicSyntax,
		Difficulty: question.L1, Action: question.ActionAttack, Form: question.FormBoth}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		a, b := rng.Int(-20, 50), rng.Int(-20, 50)
		scene := rng.Scenario("words")
		sig := "def add(a: int, b: int) -> int:"
		answer := "def add(a: int, b: int) -> int:\n    return a + b\n"
		body := []string{"    return a + b"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("add(%d, %d)", a, b), expected: fmt.Sprintf("%d", a+b)},
			caseAlias{name: "example-2", call: "add(-3, 7)", expected: "4"},
		)
		hidden := cases(
			caseAlias{name: "zero-add", call: "add(0, 0)", expected: "0"},
			caseAlias{name: "negative", call: "add(-5, -8)", expected: "-13"},
			caseAlias{name: "mixed-sign", call: "add(-10, 25)", expected: "15"},
			caseAlias{name: "large", call: "add(100, 200)", expected: "300"},
			caseAlias{name: "carry", call: fmt.Sprintf("add(%d, %d)", 99, -100), expected: "-1"},
		)
		stmt := fmt.Sprintf("竞技场%s试炼：实现两整数相加。\n%s", scene, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{0: true}, visible, hidden, answer)
	}}
}

// 语法 L2：偶数求和（语法组合，参考解 5 行）。
func syntaxSumEvensL2() question.Template {
	meta := question.Meta{ID: "syntax-sum-evens-L2-attack", Topic: question.TopicSyntax,
		Difficulty: question.L2, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), -9, 9, question.ListOpts{})
		sc := rng.Scenario("words")
		sig := "def sum_evens(nums: list) -> int:"
		answer := "def sum_evens(nums: list) -> int:\n    total = 0\n    for x in nums:\n        if x % 2 == 0:\n            total += x\n    return total\n"
		body := []string{"    total = 0", "    for x in nums:", "        if x % 2 == 0:", "            total += x", "    return total"}
		sum := 0
		for _, v := range nums {
			if v%2 == 0 {
				sum += v
			}
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("sum_evens(%s)", pyList(nums)), expected: fmt.Sprintf("%d", sum)},
			caseAlias{name: "example-2", call: "sum_evens([1, 2, 3, 4])", expected: "6"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "sum_evens([])", expected: "0"},
			caseAlias{name: "single-even", call: "sum_evens([4])", expected: "4"},
			caseAlias{name: "single-odd", call: "sum_evens([7])", expected: "0"},
			caseAlias{name: "all-odd", call: "sum_evens([1, 3, 5])", expected: "0"},
			caseAlias{name: "negatives", call: "sum_evens([-4, -3, -2])", expected: "-6"},
			caseAlias{name: "zeros", call: "sum_evens([0, 0, 1])", expected: "0"},
		)
		stmt := fmt.Sprintf("%s能量池：求列表中所有偶数之和（空列表返回 0）。\n%s", sc, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{2: true, 3: true}, visible, hidden, answer)
	}}
}

// 语法 L3：整除计数（循环 + 分支 + 边界，参考解 8 行）。
func syntaxCountDivL3() question.Template {
	meta := question.Meta{ID: "syntax-count-div-L3-attack", Topic: question.TopicSyntax,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "no-match"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		k := rng.Int(2, 9)
		nums := rng.IntList(rng.Int(4, 7), -15, 15, question.ListOpts{})
		sig := "def count_divisible(nums: list, k: int) -> int:"
		answer := "def count_divisible(nums: list, k: int) -> int:\n    count = 0\n    for x in nums:\n        if x % k != 0:\n            continue\n        count += 1\n    return count\n"
		body := []string{"    count = 0", "    for x in nums:", "        if x % k != 0:", "            continue", "        count += 1", "    return count"}
		c := 0
		for _, v := range nums {
			if v%k == 0 {
				c++
			}
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("count_divisible(%s, %d)", pyList(nums), k), expected: fmt.Sprintf("%d", c)},
			caseAlias{name: "example-2", call: "count_divisible([2, 3, 4, 5], 2)", expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: fmt.Sprintf("count_divisible([], %d)", k), expected: "0"},
			caseAlias{name: "no-match", call: fmt.Sprintf("count_divisible([1, 3, 5], %d)", k), expected: fmt.Sprintf("%d", countDiv([]int{1, 3, 5}, k))},
			caseAlias{name: "all-match", call: fmt.Sprintf("count_divisible([%d, %d], %d)", k, 2*k, k), expected: "2"},
			caseAlias{name: "negative-div", call: fmt.Sprintf("count_divisible([-6, -4, 3], %d)", k), expected: fmt.Sprintf("%d", countDiv([]int{-6, -4, 3}, k))},
			caseAlias{name: "with-zero", call: fmt.Sprintf("count_divisible([0, 1, 2], %d)", k), expected: fmt.Sprintf("%d", countDiv([]int{0, 1, 2}, k))},
		)
		stmt := fmt.Sprintf("统计列表中能被 k 整除的元素个数（空列表返回 0）。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{2: true, 4: true}, visible, hidden, answer)
	}}
}

func countDiv(nums []int, k int) int {
	c := 0
	for _, v := range nums {
		if v%k == 0 {
			c++
		}
	}
	return c
}

// 语法 L4：符文串括号匹配（栈模拟 + 边界，参考解 9 行；12 号票批 1 高档示范，
// 算法内核参考 codetop59 有效的括号，题面经情境池重包装）。
func syntaxBracketsL4() question.Template {
	meta := question.Meta{ID: "syntax-brackets-L4-attack", Topic: question.TopicSyntax,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "wrong-order", "unmatched-open", "unmatched-close"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		kind := rng.Scenario("rune")
		wellFormed := rng.Pick([]string{"{[()]}", "([]{})", "{[()()]}"})
		sig := "def matched_brackets(s: str) -> bool:"
		answer := "def matched_brackets(s: str) -> bool:\n" +
			"    pairs = {')': '(', ']': '[', '}': '{'}\n" +
			"    stack = []\n" +
			"    for ch in s:\n" +
			"        if ch in '([{':\n" +
			"            stack.append(ch)\n" +
			"            continue\n" +
			"        if not stack or stack.pop() != pairs[ch]:\n" +
			"            return False\n" +
			"    return not stack\n"
		body := []string{
			"    pairs = {')': '(', ']': '[', '}': '{'}",
			"    stack = []",
			"    for ch in s:",
			"        if ch in '([{':",
			"            stack.append(ch)",
			"            continue",
			"        if not stack or stack.pop() != pairs[ch]:",
			"            return False",
			"    return not stack",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("matched_brackets(%s)", pyStr(wellFormed)), expected: "True"},
			caseAlias{name: "example-2", call: "matched_brackets('([)]')", expected: "False"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "matched_brackets('')", expected: "True"},
			caseAlias{name: "unmatched-close", call: "matched_brackets(')(')", expected: "False"},
			caseAlias{name: "unmatched-open", call: "matched_brackets('((')", expected: "False"},
			caseAlias{name: "wrong-order", call: "matched_brackets('{(})')", expected: "False"},
			caseAlias{name: "nested-ok", call: "matched_brackets('{[()()]}')", expected: "True"},
			caseAlias{name: "sequential-ok", call: "matched_brackets('()[]{}')", expected: "True"},
		)
		stmt := fmt.Sprintf("%s串校验：字符串只含 ()[]{} 四种括号，必须以正确顺序闭合（空串视为合法）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{3: true, 6: true}, visible, hidden, answer)
	}}
}
