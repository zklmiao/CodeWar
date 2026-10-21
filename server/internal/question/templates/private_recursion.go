// 私有行动题·递归与简单 DP 主题（12 号票批 6b：move/defend/skill × L2-L4；
// 递归题型 L1/L5 为语义空格白名单，见 coverage_test.go）。
// 形式均为 complete；题型标注进入矩阵格。
package templates

import (
	"fmt"
	"math/bits"
	"strings"

	"codewar/server/internal/question"
)

// 移动 L2·递归：斐波那契步法（递归基线 + 递归步，参考解 3 行）。
func moveFibL2() question.Template {
	meta := question.Meta{ID: "move-fib-L2-move", Topic: question.TopicRecursion,
		Difficulty: question.L2, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"base-zero"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		n := rng.Int(0, 10)
		kind := rng.Scenario("dungeon")
		sig := "def fib(n: int) -> int:"
		answer := "def fib(n: int) -> int:\n    if n < 2:\n        return n\n    return fib(n - 1) + fib(n - 2)\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("fib(%d)", n), expected: fmt.Sprintf("%d", fibGo(n))},
			caseAlias{name: "example-2", call: "fib(5)", expected: "5"},
		)
		hidden := cases(
			caseAlias{name: "base-zero", call: "fib(0)", expected: "0"},
			caseAlias{name: "base-one", call: "fib(1)", expected: "1"},
			caseAlias{name: "two", call: "fib(2)", expected: "1"},
			caseAlias{name: "five", call: "fib(5)", expected: "5"},
			caseAlias{name: "ten", call: "fib(10)", expected: "55"},
		)
		stmt := fmt.Sprintf("%s步法数列：第 n 项满足 F(0)=0、F(1)=1、F(n)=F(n−1)+F(n−2)。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L3·递归：递归求和（嵌套辅助函数 + 切片递归，参考解 6 行）。
func moveRecSumL3() question.Template {
	meta := question.Meta{ID: "move-recsum-L3-move", Topic: question.TopicRecursion,
		Difficulty: question.L3, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-element"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(2, 5), -5, 9, question.ListOpts{})
		kind := rng.Scenario("dungeon")
		sig := "def rec_sum(nums: list) -> int:"
		answer := "def rec_sum(nums: list) -> int:\n" +
			"    def helper(items, total):\n" +
			"        if not items:\n" +
			"            return total\n" +
			"        head = items[0]\n" +
			"        rest = items[1:]\n" +
			"        return helper(rest, total + head)\n" +
			"    return helper(nums, 0)\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("rec_sum(%s)", pyList(nums)), expected: fmt.Sprintf("%d", sumGo(nums))},
			caseAlias{name: "example-2", call: "rec_sum([1, 2, 3])", expected: "6"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "rec_sum([])", expected: "0"},
			caseAlias{name: "single-element", call: "rec_sum([5])", expected: "5"},
			caseAlias{name: "mixed", call: "rec_sum([-1, 1])", expected: "0"},
			caseAlias{name: "four", call: "rec_sum([10, 20, 30, 40])", expected: "100"},
			caseAlias{name: "negatives", call: "rec_sum([-2, -3])", expected: "-5"},
		)
		stmt := fmt.Sprintf("%s递归点兵：用递归方式求列表元素之和（空列表为 0；可借助累加器辅助函数）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L4·递归：嵌套深度（空列表守卫 + 子列表递归 + 最大值，参考解 9 行）。
func moveDepthL4() question.Template {
	meta := question.Meta{ID: "move-depth-L4-move", Topic: question.TopicRecursion,
		Difficulty: question.L4, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "one-level", "deep-nesting"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nested, depth := genNestedList(rng, 3)
		kind := rng.Scenario("dungeon")
		sig := "def depth_of(nested: list) -> int:"
		answer := "def depth_of(nested: list) -> int:\n" +
			"    if not nested:\n" +
			"        return 0\n" +
			"    best = 0\n" +
			"    for item in nested:\n" +
			"        if isinstance(item, list):\n" +
			"            d = depth_of(item)\n" +
			"            if d > best:\n" +
			"                best = d\n" +
			"    return best + 1\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("depth_of(%s)", nested), expected: fmt.Sprintf("%d", depth)},
			caseAlias{name: "example-2", call: "depth_of([1, [2, [3]]])", expected: "3"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "depth_of([])", expected: "0"},
			caseAlias{name: "empty-child", call: "depth_of([[]])", expected: "1"},
			caseAlias{name: "one-level", call: "depth_of([1, 2])", expected: "1"},
			caseAlias{name: "two-level", call: "depth_of([[1], [2]])", expected: "2"},
			caseAlias{name: "deep-nesting", call: "depth_of([[[1]]])", expected: "3"},
		)
		stmt := fmt.Sprintf("%s测深渊：嵌套列表的深度定义为「空列表为 0，否则为 1 + 子列表深度的最大值」（非列表元素不计深度）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L2·递归：减伤链（递归层数 + 逐层衰减，参考解 4 行）。
func defendChainL2() question.Template {
	meta := question.Meta{ID: "defend-chain-L2-defend", Topic: question.TopicRecursion,
		Difficulty: question.L2, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"zero-layers"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		hp := rng.Int(20, 120)
		layers := rng.Int(0, 4)
		kind := rng.Scenario("dungeon")
		sig := "def chain_reduce(hp: int, layers: int) -> int:"
		answer := "def chain_reduce(hp: int, layers: int) -> int:\n    if layers == 0:\n        return hp\n    reduced = hp * 3 // 4\n    return chain_reduce(reduced, layers - 1)\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("chain_reduce(%d, %d)", hp, layers), expected: fmt.Sprintf("%d", chainReduceGo(hp, layers))},
			caseAlias{name: "example-2", call: "chain_reduce(100, 2)", expected: "56"},
		)
		hidden := cases(
			caseAlias{name: "zero-layers", call: "chain_reduce(100, 0)", expected: "100"},
			caseAlias{name: "one-layer", call: "chain_reduce(100, 1)", expected: "75"},
			caseAlias{name: "three-layers", call: "chain_reduce(100, 3)", expected: "42"},
			caseAlias{name: "small", call: "chain_reduce(7, 1)", expected: "5"},
			caseAlias{name: "many-layers", call: "chain_reduce(80, 4)", expected: fmt.Sprintf("%d", chainReduceGo(80, 4))},
		)
		stmt := fmt.Sprintf("%s减伤链：每层把当前承伤降为 3/4（向下取整），递归结算 layers 层后返回剩余承伤。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L3·递归：伤害位计数（取模分解 + 递归，参考解 5 行）。
func defendOnesL3() question.Template {
	meta := question.Meta{ID: "defend-ones-L3-defend", Topic: question.TopicRecursion,
		Difficulty: question.L3, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"zero-input", "single-bit"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		n := rng.Int(0, 255)
		kind := rng.Scenario("dungeon")
		sig := "def count_ones(n: int) -> int:"
		answer := "def count_ones(n: int) -> int:\n    if n == 0:\n        return 0\n    last = n % 2\n    rest = n // 2\n    return last + count_ones(rest)\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("count_ones(%d)", n), expected: fmt.Sprintf("%d", countOnesGo(n))},
			caseAlias{name: "example-2", call: "count_ones(7)", expected: "3"},
		)
		hidden := cases(
			caseAlias{name: "zero-input", call: "count_ones(0)", expected: "0"},
			caseAlias{name: "single-bit", call: "count_ones(1)", expected: "1"},
			caseAlias{name: "two", call: "count_ones(2)", expected: "1"},
			caseAlias{name: "ten", call: "count_ones(10)", expected: "2"},
			caseAlias{name: "full-byte", call: "count_ones(255)", expected: "8"},
		)
		stmt := fmt.Sprintf("%s能量位计数：返回整数二进制表示中 1 的个数（用递归分解，0 为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L4·递归：记忆化斐波那契（缓存命中 + 递归分支 + 负输入守卫，参考解 8 行）。
func defendMemoL4() question.Template {
	meta := question.Meta{ID: "defend-memo-L4-defend", Topic: question.TopicRecursion,
		Difficulty: question.L4, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"zero-input", "negative-input", "memo-hit"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		n := rng.Int(0, 20)
		kind := rng.Scenario("dungeon")
		sig := "def fib_memo(n: int, memo: dict) -> int:"
		answer := "def fib_memo(n: int, memo: dict) -> int:\n" +
			"    if n < 0:\n" +
			"        return 0\n" +
			"    if n < 2:\n" +
			"        return n\n" +
			"    if n in memo:\n" +
			"        return memo[n]\n" +
			"    value = fib_memo(n - 1, memo) + fib_memo(n - 2, memo)\n" +
			"    memo[n] = value\n" +
			"    return value\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("fib_memo(%d, {})", n), expected: fmt.Sprintf("%d", fibGo(n))},
			caseAlias{name: "example-2", call: "fib_memo(10, {})", expected: "55"},
		)
		hidden := cases(
			caseAlias{name: "zero-input", call: "fib_memo(0, {})", expected: "0"},
			caseAlias{name: "negative-input", call: "fib_memo(-1, {})", expected: "0"},
			caseAlias{name: "memo-hit", call: "fib_memo(2, {})", expected: "1"},
			caseAlias{name: "twenty", call: "fib_memo(20, {})", expected: "6765"},
			caseAlias{name: "ten", call: "fib_memo(10, {})", expected: "55"},
		)
		stmt := fmt.Sprintf("%s记忆回廊：用带缓存（memo）的递归计算斐波那契第 n 项（负输入返回 0；memo 由调用方传入空字典）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L2·递归：连击递减（递归基例 + 递减步长，参考解 3 行）。
func skillComboL2() question.Template {
	meta := question.Meta{ID: "skill-combo-L2-skill", Topic: question.TopicRecursion,
		Difficulty: question.L2, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"zero-power"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		power := rng.Int(0, 12)
		kind := rng.Scenario("dungeon")
		sig := "def combo_damage(power: int) -> int:"
		answer := "def combo_damage(power: int) -> int:\n    if power <= 0:\n        return 0\n    return power + combo_damage(power - 2)\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("combo_damage(%d)", power), expected: fmt.Sprintf("%d", comboGo(power))},
			caseAlias{name: "example-2", call: "combo_damage(5)", expected: "9"},
		)
		hidden := cases(
			caseAlias{name: "zero-power", call: "combo_damage(0)", expected: "0"},
			caseAlias{name: "negative", call: "combo_damage(-3)", expected: "0"},
			caseAlias{name: "one", call: "combo_damage(1)", expected: "1"},
			caseAlias{name: "even", call: "combo_damage(6)", expected: "12"},
			caseAlias{name: "five", call: "combo_damage(5)", expected: "9"},
		)
		stmt := fmt.Sprintf("%s连击递减：每段伤害为当前威力，之后威力减少 2 继续追击，返回总伤害（威力 ≤0 时停止）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L3·递归：数字根（逐位求和 + 尾递归，参考解 7 行）。
func skillDrootL3() question.Template {
	meta := question.Meta{ID: "skill-droot-L3-skill", Topic: question.TopicRecursion,
		Difficulty: question.L3, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"single-digit", "zero-input"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		n := rng.Int(0, 9999)
		kind := rng.Scenario("dungeon")
		sig := "def digit_root(n: int) -> int:"
		answer := "def digit_root(n: int) -> int:\n    if n < 10:\n        return n\n    total = 0\n    while n > 0:\n        total += n % 10\n        n = n // 10\n    return digit_root(total)\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("digit_root(%d)", n), expected: fmt.Sprintf("%d", digitRootGo(n))},
			caseAlias{name: "example-2", call: "digit_root(38)", expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "single-digit", call: "digit_root(9)", expected: "9"},
			caseAlias{name: "zero-input", call: "digit_root(0)", expected: "0"},
			caseAlias{name: "three-nines", call: "digit_root(999)", expected: "9"},
			caseAlias{name: "five-digits", call: "digit_root(12345)", expected: "6"},
			caseAlias{name: "two-digits", call: "digit_root(56)", expected: "2"},
		)
		stmt := fmt.Sprintf("%s数字根：反复求各位数字之和直到只剩一位（0 的数字根为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L4·递归：组合数（递归 + 元组键缓存 + 边界，参考解 9 行）。
func skillChooseL4() question.Template {
	meta := question.Meta{ID: "skill-choose-L4-skill", Topic: question.TopicRecursion,
		Difficulty: question.L4, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"k-zero", "k-greater-than-n", "full-k"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		n := rng.Int(0, 10)
		k := rng.Int(0, n)
		kind := rng.Scenario("dungeon")
		sig := "def choose(n: int, k: int, memo: dict) -> int:"
		answer := "def choose(n: int, k: int, memo: dict) -> int:\n" +
			"    if k < 0 or k > n:\n" +
			"        return 0\n" +
			"    if k == 0 or k == n:\n" +
			"        return 1\n" +
			"    key = (n, k)\n" +
			"    if key in memo:\n" +
			"        return memo[key]\n" +
			"    value = choose(n - 1, k - 1, memo) + choose(n - 1, k, memo)\n" +
			"    memo[key] = value\n" +
			"    return value\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("choose(%d, %d, {})", n, k), expected: fmt.Sprintf("%d", chooseGo(n, k))},
			caseAlias{name: "example-2", call: "choose(5, 2, {})", expected: "10"},
		)
		hidden := cases(
			caseAlias{name: "k-zero", call: "choose(5, 0, {})", expected: "1"},
			caseAlias{name: "k-greater-than-n", call: "choose(5, 6, {})", expected: "0"},
			caseAlias{name: "full-k", call: "choose(4, 4, {})", expected: "1"},
			caseAlias{name: "zero-zero", call: "choose(0, 0, {})", expected: "1"},
			caseAlias{name: "ten-five", call: "choose(10, 5, {})", expected: "252"},
		)
		stmt := fmt.Sprintf("%s命运组合：用带缓存的递归计算组合数 C(n, k)（k<0 或 k>n 为 0；memo 由调用方传入空字典）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// ---- 期望值计算（与参考解同语义） ----

func fibGo(n int) int {
	if n < 2 {
		return n
	}
	return fibGo(n-1) + fibGo(n-2)
}

// genNestedList：生成嵌套列表的 repr 与其深度（题面语义：空列表 0，否则 1+子列表最大深度）。
func genNestedList(rng *question.RNG, level int) (string, int) {
	n := rng.Int(0, 2)
	if n == 0 {
		return "[]", 0
	}
	parts := make([]string, 0, n)
	bestSub := -1
	for i := 0; i < n; i++ {
		if level > 1 && rng.Bool(0.5) {
			rep, d := genNestedList(rng, level-1)
			parts = append(parts, rep)
			if d > bestSub {
				bestSub = d
			}
		} else {
			parts = append(parts, fmt.Sprintf("%d", rng.Int(1, 9)))
		}
	}
	depth := 1
	if bestSub >= 0 {
		depth = 1 + bestSub
	}
	return "[" + strings.Join(parts, ", ") + "]", depth
}

func chainReduceGo(hp, layers int) int {
	for i := 0; i < layers; i++ {
		hp = hp * 3 / 4
	}
	return hp
}

func countOnesGo(n int) int {
	return bits.OnesCount(uint(n))
}

func comboGo(power int) int {
	if power <= 0 {
		return 0
	}
	return power + comboGo(power-2)
}

func digitRootGo(n int) int {
	for n >= 10 {
		total := 0
		for n > 0 {
			total += n % 10
			n /= 10
		}
		n = total
	}
	return n
}

func chooseGo(n, k int) int {
	if k < 0 || k > n {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}
	return chooseGo(n-1, k-1) + chooseGo(n-1, k)
}
