// 攻击题·递归与简单 DP（3 模板：L2/L3/L4；L1/L5 无语义格归 12 号票）。
package templates

import (
	"fmt"

	"codewar/server/internal/question"
)

// 递归 L2：阶乘（约定 n≥0，参考解 4 行）。
func recursionFactorialL2() question.Template {
	meta := question.Meta{ID: "recursion-factorial-L2-attack", Topic: question.TopicRecursion,
		Difficulty: question.L2, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"base-zero", "base-one"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		n := rng.Int(2, 7)
		sig := "def factorial(n: int) -> int:"
		answer := "def factorial(n: int) -> int:\n    result = 1\n    for i in range(2, n + 1):\n        result *= i\n    return result\n"
		body := []string{"    result = 1", "    for i in range(2, n + 1):", "        result *= i", "    return result"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("factorial(%d)", n), expected: fmt.Sprintf("%d", factGo(n))},
			caseAlias{name: "example-2", call: "factorial(5)", expected: "120"},
		)
		hidden := cases(
			caseAlias{name: "base-zero", call: "factorial(0)", expected: "1"},
			caseAlias{name: "base-one", call: "factorial(1)", expected: "1"},
			caseAlias{name: "small", call: "factorial(3)", expected: "6"},
			caseAlias{name: "medium", call: "factorial(7)", expected: "5040"},
			caseAlias{name: "ten", call: "factorial(10)", expected: "3628800"},
		)
		stmt := fmt.Sprintf("求 n 的阶乘（约定 n≥0；0! = 1）。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{1: true, 2: true}, visible, hidden, answer)
	}}
}

// 递归 L3：二分查找（有序不重复列表，参考解 8 行；未找到返回 -1）。
func recursionBinarySearchL3() question.Template {
	meta := question.Meta{ID: "recursion-binary-search-L3-attack", Topic: question.TopicRecursion,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "single-element", "not-found"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		n := rng.Int(3, 6)
		nums := rng.IntList(n, 0, 30, question.ListOpts{Distinct: true, Sorted: true})
		target := nums[rng.Int(0, n-1)]
		sig := "def binary_search(nums: list, target: int) -> int:"
		answer := "def binary_search(nums: list, target: int) -> int:\n    lo = 0\n    hi = len(nums) - 1\n    while lo <= hi:\n        mid = (lo + hi) // 2\n        if nums[mid] == target:\n            return mid\n        lo, hi = (mid + 1, hi) if nums[mid] < target else (lo, mid - 1)\n    return -1\n"
		body := []string{"    lo = 0", "    hi = len(nums) - 1", "    while lo <= hi:",
			"        mid = (lo + hi) // 2", "        if nums[mid] == target:", "            return mid",
			"        lo, hi = (mid + 1, hi) if nums[mid] < target else (lo, mid - 1)", "    return -1"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("binary_search(%s, %d)", pyList(nums), target), expected: fmt.Sprintf("%d", bsearchGo(nums, target))},
			caseAlias{name: "example-2", call: "binary_search([1, 3, 5, 7], 5)", expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "binary_search([], 3)", expected: "-1"},
			caseAlias{name: "single-element", call: "binary_search([4], 4)", expected: "0"},
			caseAlias{name: "single-miss", call: "binary_search([4], 5)", expected: "-1"},
			caseAlias{name: "not-found", call: "binary_search([1, 3, 5, 7], 4)", expected: "-1"},
			caseAlias{name: "first-last", call: "binary_search([2, 4, 6, 8, 10], 10)", expected: "4"},
		)
		stmt := fmt.Sprintf("在升序不重复列表中查找 target（找到返回下标，否则 -1）。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{3: true, 6: true}, visible, hidden, answer)
	}}
}

// 递归 L4：杨辉三角第 n 行（DP 递推，参考解 10 行；约定 n≥0）。
func recursionPascalL4() question.Template {
	meta := question.Meta{ID: "recursion-pascal-L4-attack", Topic: question.TopicRecursion,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"row-zero", "row-one"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		n := rng.Int(2, 6)
		sig := "def pascal_row(n: int) -> list:"
		answer := "def pascal_row(n: int) -> list:\n    row = [1]\n    for _ in range(n):\n        nxt = [1]\n        for v in range(len(row) - 1):\n            nxt.append(row[v] + row[v + 1])\n        nxt.append(1)\n        row = nxt\n    if n == 0:\n        return [1]\n    return row\n"
		body := []string{"    row = [1]", "    for _ in range(n):", "        nxt = [1]",
			"        for v in range(len(row) - 1):", "            nxt.append(row[v] + row[v + 1])",
			"        nxt.append(1)", "        row = nxt", "    if n == 0:", "        return [1]", "    return row"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("pascal_row(%d)", n), expected: pyList(pascalGo(n))},
			caseAlias{name: "example-2", call: "pascal_row(3)", expected: "[1, 3, 3, 1]"},
		)
		hidden := cases(
			caseAlias{name: "row-zero", call: "pascal_row(0)", expected: "[1]"},
			caseAlias{name: "row-one", call: "pascal_row(1)", expected: "[1, 1]"},
			caseAlias{name: "row-two", call: "pascal_row(2)", expected: "[1, 2, 1]"},
			caseAlias{name: "row-four", call: "pascal_row(4)", expected: "[1, 4, 6, 4, 1]"},
			caseAlias{name: "row-five", call: "pascal_row(5)", expected: "[1, 5, 10, 10, 5, 1]"},
		)
		stmt := fmt.Sprintf("返回杨辉三角第 n 行（0 起；pascal_row(0) = [1]）。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{4: true, 6: true}, visible, hidden, answer)
	}}
}

func factGo(n int) int {
	r := 1
	for i := 2; i <= n; i++ {
		r *= i
	}
	return r
}

func bsearchGo(nums []int, t int) int {
	lo, hi := 0, len(nums)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		if nums[mid] == t {
			return mid
		}
		if nums[mid] < t {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return -1
}

func pascalGo(n int) []int {
	row := []int{1}
	for i := 0; i < n; i++ {
		nxt := []int{1}
		for v := 0; v < len(row)-1; v++ {
			nxt = append(nxt, row[v]+row[v+1])
		}
		nxt = append(nxt, 1)
		row = nxt
	}
	return row
}

// 递归 L3（关键格副本）：爬楼梯（递归 + 记忆化，参考解 7 行；12 号票批 6b，
// 算法内核参考 codetop59 爬楼梯，题面经情境池重包装）。
func recursionClimbL3() question.Template {
	meta := question.Meta{ID: "recursion-climb-L3-attack", Topic: question.TopicRecursion,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"base-zero", "base-one", "memo-hit"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		n := rng.Int(0, 8)
		kind := rng.Scenario("dungeon")
		sig := "def climb(n: int, memo: dict) -> int:"
		answer := "def climb(n: int, memo: dict) -> int:\n" +
			"    if n <= 2:\n" +
			"        return n\n" +
			"    if n in memo:\n" +
			"        return memo[n]\n" +
			"    value = climb(n - 1, memo) + climb(n - 2, memo)\n" +
			"    memo[n] = value\n" +
			"    return value\n"
		body := []string{
			"    if n <= 2:",
			"        return n",
			"    if n in memo:",
			"        return memo[n]",
			"    value = climb(n - 1, memo) + climb(n - 2, memo)",
			"    memo[n] = value",
			"    return value",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("climb(%d, {})", n), expected: fmt.Sprintf("%d", climbGo(n))},
			caseAlias{name: "example-2", call: "climb(5, {})", expected: "8"},
		)
		hidden := cases(
			caseAlias{name: "base-zero", call: "climb(0, {})", expected: "0"},
			caseAlias{name: "base-one", call: "climb(1, {})", expected: "1"},
			caseAlias{name: "memo-hit", call: "climb(3, {})", expected: "3"},
			caseAlias{name: "stairs-5", call: "climb(5, {})", expected: "8"},
			caseAlias{name: "stairs-10", call: "climb(10, {})", expected: "89"},
		)
		stmt := fmt.Sprintf("%s阶梯步法：每次可跨 1 或 2 级，求到达第 n 级的不同走法数（n≤2 时返回 n；memo 为记忆化缓存）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{4: true, 5: true}, visible, hidden, answer)
	}}
}

// 递归 L4（关键格副本）：快速幂（分治递归 + 奇偶分支，参考解 8 行；12 号票批 6b）。
func recursionFastPowerL4() question.Template {
	meta := question.Meta{ID: "recursion-fast-power-L4-attack", Topic: question.TopicRecursion,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"exp-zero", "negative-exp", "odd-exp"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		base := rng.Int(2, 3)
		exp := rng.Int(0, 6)
		kind := rng.Scenario("dungeon")
		sig := "def fast_power(base: int, exp: int) -> int:"
		answer := "def fast_power(base: int, exp: int) -> int:\n" +
			"    if exp < 0:\n" +
			"        return 0\n" +
			"    if exp == 0:\n" +
			"        return 1\n" +
			"    half = fast_power(base, exp // 2)\n" +
			"    value = half * half\n" +
			"    if exp % 2 == 1:\n" +
			"        value = value * base\n" +
			"    return value\n"
		body := []string{
			"    if exp < 0:",
			"        return 0",
			"    if exp == 0:",
			"        return 1",
			"    half = fast_power(base, exp // 2)",
			"    value = half * half",
			"    if exp % 2 == 1:",
			"        value = value * base",
			"    return value",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("fast_power(%d, %d)", base, exp), expected: fmt.Sprintf("%d", fastPowerGo(base, exp))},
			caseAlias{name: "example-2", call: "fast_power(2, 10)", expected: "1024"},
		)
		hidden := cases(
			caseAlias{name: "exp-zero", call: "fast_power(2, 0)", expected: "1"},
			caseAlias{name: "negative-exp", call: "fast_power(2, -1)", expected: "0"},
			caseAlias{name: "odd-exp", call: "fast_power(3, 3)", expected: "27"},
			caseAlias{name: "exp-one", call: "fast_power(5, 1)", expected: "5"},
			caseAlias{name: "square", call: "fast_power(7, 2)", expected: "49"},
		)
		stmt := fmt.Sprintf("%s奥术乘方：用分治递归计算 base 的 exp 次幂（负指数返回 0；0 次幂为 1）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{4: true, 5: true}, visible, hidden, answer)
	}}
}

// climbGo：期望值计算（与参考解同语义：n≤2 返回 n，其后斐波那契式递推）。
func climbGo(n int) int {
	if n <= 2 {
		return n
	}
	prev2, prev1 := 1, 2
	for i := 3; i <= n; i++ {
		prev2, prev1 = prev1, prev2+prev1
	}
	return prev1
}

// fastPowerGo：期望值计算（与参考解同语义：负指数 0，0 次幂 1）。
func fastPowerGo(base, exp int) int {
	if exp < 0 {
		return 0
	}
	result := 1
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}
