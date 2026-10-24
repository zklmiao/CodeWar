// 攻击题·语法与运算（L1/L2/L3 黄金样板 + L4 十二号票扩容）。
package templates

import (
	"fmt"
	"strings"

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
			caseAlias{name: "no-match", call: "count_divisible([1, 3, 5], 2)", expected: "0"},
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

// 语法 L3（关键格副本）：整数数字反转（循环 + 取模分解，参考解 7 行；12 号票批 2）。
func syntaxReverseDigitsL3() question.Template {
	meta := question.Meta{ID: "syntax-reverse-digits-L3-attack", Topic: question.TopicSyntax,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"trailing-zero", "negative", "single-digit", "zero"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		n := rng.Int(-450, 980)
		kind := rng.Scenario("rune")
		sig := "def reverse_digits(n: int) -> int:"
		answer := "def reverse_digits(n: int) -> int:\n" +
			"    sign = -1 if n < 0 else 1\n" +
			"    n = abs(n)\n" +
			"    result = 0\n" +
			"    while n > 0:\n" +
			"        result = result * 10 + n % 10\n" +
			"        n //= 10\n" +
			"    return sign * result\n"
		body := []string{
			"    sign = -1 if n < 0 else 1",
			"    n = abs(n)",
			"    result = 0",
			"    while n > 0:",
			"        result = result * 10 + n % 10",
			"        n //= 10",
			"    return sign * result",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("reverse_digits(%d)", n), expected: fmt.Sprintf("%d", reverseDigitsGo(n))},
			caseAlias{name: "example-2", call: "reverse_digits(405)", expected: "504"},
		)
		hidden := cases(
			caseAlias{name: "trailing-zero", call: "reverse_digits(120)", expected: "21"},
			caseAlias{name: "negative", call: "reverse_digits(-123)", expected: "-321"},
			caseAlias{name: "single-digit", call: "reverse_digits(7)", expected: "7"},
			caseAlias{name: "zero", call: "reverse_digits(0)", expected: "0"},
			caseAlias{name: "inner-zero", call: "reverse_digits(1003)", expected: "3001"},
		)
		stmt := fmt.Sprintf("%s解密：将整数的数字位反转（负号保留在最前；反转产生的前导零舍弃）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{0: true, 4: true}, visible, hidden, answer)
	}}
}

// 语法 L4（关键格副本）：大数加法（对齐 + 进位链 + 边界，参考解 12 行；12 号票批 2，
// 算法内核参考 codetop59 字符串相加，题面经情境池重包装）。
func syntaxBigAddL4() question.Template {
	meta := question.Meta{ID: "syntax-big-add-L4-attack", Topic: question.TopicSyntax,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"carry-chain", "different-length", "zero-value"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		a := rng.Int(10, 999)
		b := rng.Int(1, 9)
		kind := rng.Scenario("rune")
		sig := "def add_big_numbers(a: str, b: str) -> str:"
		answer := "def add_big_numbers(a: str, b: str) -> str:\n" +
			"    n = max(len(a), len(b))\n" +
			"    a = a.zfill(n)\n" +
			"    b = b.zfill(n)\n" +
			"    carry = 0\n" +
			"    digits = []\n" +
			"    for k in range(n - 1, -1, -1):\n" +
			"        total = int(a[k]) + int(b[k]) + carry\n" +
			"        digits.append(str(total % 10))\n" +
			"        carry = total // 10\n" +
			"    if carry > 0:\n" +
			"        digits.append('1')\n" +
			"    return ''.join(reversed(digits))\n"
		body := []string{
			"    n = max(len(a), len(b))",
			"    a = a.zfill(n)",
			"    b = b.zfill(n)",
			"    carry = 0",
			"    digits = []",
			"    for k in range(n - 1, -1, -1):",
			"        total = int(a[k]) + int(b[k]) + carry",
			"        digits.append(str(total % 10))",
			"        carry = total // 10",
			"    if carry > 0:",
			"        digits.append('1')",
			"    return ''.join(reversed(digits))",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("add_big_numbers('%d', '%d')", a, b), expected: pyStr(addBigGo(fmt.Sprintf("%d", a), fmt.Sprintf("%d", b)))},
			caseAlias{name: "example-2", call: "add_big_numbers('999', '1')", expected: pyStr("1000")},
		)
		hidden := cases(
			caseAlias{name: "different-length", call: "add_big_numbers('123', '7')", expected: pyStr("130")},
			caseAlias{name: "zero-value", call: "add_big_numbers('0', '0')", expected: pyStr("0")},
			caseAlias{name: "simple", call: "add_big_numbers('12', '34')", expected: pyStr("46")},
			caseAlias{name: "carry-chain", call: "add_big_numbers('999999999999999999', '1')", expected: pyStr("1000000000000000000")},
			caseAlias{name: "mid-carry", call: "add_big_numbers('19', '91')", expected: pyStr("110")},
		)
		stmt := fmt.Sprintf("复算远古%s数碑：两个非负整数的十进制串相加，返回和的十进制串（输入无前导零）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{6: true, 8: true}, visible, hidden, answer)
	}}
}

// reverseDigitsGo：期望值计算（与参考解同语义）。
func reverseDigitsGo(n int) int {
	sign := 1
	if n < 0 {
		sign = -1
		n = -n
	}
	result := 0
	for n > 0 {
		result = result*10 + n%10
		n /= 10
	}
	return sign * result
}

// addBigGo：期望值计算（与参考解同语义：左补零对齐 + 逐位进位）。
func addBigGo(a, b string) string {
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	a = strings.Repeat("0", n-len(a)) + a
	b = strings.Repeat("0", n-len(b)) + b
	carry := 0
	digits := make([]byte, 0, n+1)
	for k := n - 1; k >= 0; k-- {
		total := int(a[k]-'0') + int(b[k]-'0') + carry
		digits = append(digits, byte('0'+total%10))
		carry = total / 10
	}
	if carry > 0 {
		digits = append(digits, byte('0'+carry))
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
