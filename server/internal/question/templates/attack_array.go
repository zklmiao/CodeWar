// 攻击题·数组与列表（3 模板：L2/L3/L4，黄金样板）。
package templates

import (
	"fmt"

	"codewar/server/internal/question"
)

// 数组 L2：去重保序（参考解 5 行）。
func arrayDedupL2() question.Template {
	meta := question.Meta{ID: "array-dedup-L2-attack", Topic: question.TopicArray,
		Difficulty: question.L2, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "all-same"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), 1, 5, question.ListOpts{})
		sig := "def dedup(nums: list) -> list:"
		answer := "def dedup(nums: list) -> list:\n    seen = []\n    for x in nums:\n        if x not in seen:\n            seen.append(x)\n    return seen\n"
		body := []string{"    seen = []", "    for x in nums:", "        if x not in seen:", "            seen.append(x)", "    return seen"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("dedup(%s)", pyList(nums)), expected: pyList(dedupGo(nums))},
			caseAlias{name: "example-2", call: "dedup([1, 2, 2, 3, 1])", expected: "[1, 2, 3]"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "dedup([])", expected: "[]"},
			caseAlias{name: "all-same", call: "dedup([5, 5, 5])", expected: "[5]"},
			caseAlias{name: "no-dup", call: "dedup([1, 2, 3])", expected: "[1, 2, 3]"},
			caseAlias{name: "tail-dup", call: "dedup([1, 2, 3, 1])", expected: "[1, 2, 3]"},
			caseAlias{name: "alternating", call: "dedup([2, 1, 2, 1])", expected: "[2, 1]"},
		)
		stmt := fmt.Sprintf("列表去重并保持首次出现的顺序。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{2: true, 3: true}, visible, hidden, answer)
	}}
}

// 数组 L3：两数之和下标（哈希表，参考解 7 行；生成保证恰好一个解）。
func arrayTwoSumL3() question.Template {
	meta := question.Meta{ID: "array-two-sum-L3-attack", Topic: question.TopicArray,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"duplicates", "negative"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums, target := genTwoSum(rng)
		sol := solveTwoSum(nums, target)
		sig := "def two_sum(nums: list, target: int) -> list:"
		answer := "def two_sum(nums: list, target: int) -> list:\n    index = {}\n    for i, x in enumerate(nums):\n        need = target - x\n        if need in index:\n            return [index[need], i]\n        index[x] = i\n    return []\n"
		body := []string{"    index = {}", "    for i, x in enumerate(nums):", "        need = target - x",
			"        if need in index:", "            return [index[need], i]", "        index[x] = i", "    return []"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("two_sum(%s, %d)", pyList(nums), target), expected: pyList(sol)},
			caseAlias{name: "example-2", call: "two_sum([2, 7, 11, 15], 9)", expected: "[0, 1]"},
		)
		hidden := cases(
			caseAlias{name: "middle", call: "two_sum([3, 2, 4], 6)", expected: "[1, 2]"},
			caseAlias{name: "duplicates", call: "two_sum([3, 3], 6)", expected: "[0, 1]"},
			caseAlias{name: "negative", call: "two_sum([-1, 5, 10, -7], 3)", expected: "[2, 3]"},
			caseAlias{name: "tail-pair", call: "two_sum([1, 5, 3, 7], 10)", expected: "[2, 3]"},
			caseAlias{name: "head-pair", call: "two_sum([4, 6, 1, 2], 10)", expected: "[0, 1]"},
		)
		stmt := fmt.Sprintf("在数组中找出和为目标值的两个数的下标（恰好一个解，返回升序下标列表）。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{2: true, 3: true}, visible, hidden, answer)
	}}
}

// 数组 L4：定长窗口最大和（滑动窗口，参考解 11 行；约定 k ≤ len(nums)）。
func arrayWindowSumL4() question.Template {
	meta := question.Meta{ID: "array-window-sum-L4-attack", Topic: question.TopicArray,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "single-element", "all-negative", "full-window"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		n := rng.Int(4, 7)
		nums := rng.IntList(n, -9, 9, question.ListOpts{})
		k := rng.Int(2, n)
		sig := "def max_window_sum(nums: list, k: int) -> int:"
		answer := "def max_window_sum(nums: list, k: int) -> int:\n    if k <= 0 or len(nums) == 0:\n        return 0\n    window = 0\n    for i in range(k):\n        window += nums[i]\n    best = window\n    for i in range(k, len(nums)):\n        window += nums[i] - nums[i - k]\n        if window > best:\n            best = window\n    return best\n"
		body := []string{"    if k <= 0 or len(nums) == 0:", "        return 0", "    window = 0",
			"    for i in range(k):", "        window += nums[i]", "    best = window",
			"    for i in range(k, len(nums)):", "        window += nums[i] - nums[i - k]",
			"        if window > best:", "            best = window", "    return best"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("max_window_sum(%s, %d)", pyList(nums), k), expected: fmt.Sprintf("%d", windowSumGo(nums, k))},
			caseAlias{name: "example-2", call: "max_window_sum([1, 4, 2, 10, 3], 3)", expected: "16"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "max_window_sum([], 3)", expected: "0"},
			caseAlias{name: "single-element", call: "max_window_sum([5], 1)", expected: "5"},
			caseAlias{name: "all-negative", call: "max_window_sum([-5, -2, -8, -1], 2)", expected: "-7"},
			caseAlias{name: "full-window", call: "max_window_sum([1, 2, 3], 3)", expected: "6"},
			caseAlias{name: "tail-best", call: "max_window_sum([1, 2, 10, 1], 2)", expected: "12"},
		)
		stmt := fmt.Sprintf("求长度为 k 的连续子数组的最大和（约定 k ≤ len(nums)；空列表返回 0）。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{7: true, 8: true}, visible, hidden, answer)
	}}
}

func dedupGo(nums []int) []int {
	var out []int
	seen := map[int]bool{}
	for _, v := range nums {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	if out == nil {
		return []int{}
	}
	return out
}

// genTwoSum：生成恰好一个解的实例（重试有界，消耗同一 RNG 流，确定性不变）。
func genTwoSum(rng *question.RNG) ([]int, int) {
	for tries := 0; tries < 50; tries++ {
		n := rng.Int(3, 5)
		nums := rng.IntList(n, -9, 9, question.ListOpts{})
		i, j := rng.Int(0, n-1), rng.Int(0, n-1)
		if i == j {
			continue
		}
		if i > j {
			i, j = j, i
		}
		target := nums[i] + nums[j]
		if countPairs(nums, target) == 1 {
			return nums, target
		}
	}
	// 保底固定实例（仍合法单解）。
	return []int{2, 7, 11, 15}, 9
}

func countPairs(nums []int, target int) int {
	n := 0
	for i := 0; i < len(nums); i++ {
		for j := i + 1; j < len(nums); j++ {
			if nums[i]+nums[j] == target {
				n++
			}
		}
	}
	return n
}

func solveTwoSum(nums []int, target int) []int {
	idx := map[int]int{}
	for i, x := range nums {
		if p, ok := idx[target-x]; ok {
			return []int{p, i}
		}
		idx[x] = i
	}
	return []int{}
}

func windowSumGo(nums []int, k int) int {
	if k <= 0 || len(nums) == 0 {
		return 0
	}
	if k > len(nums) {
		k = len(nums)
	}
	w := 0
	for i := 0; i < k; i++ {
		w += nums[i]
	}
	best := w
	for i := k; i < len(nums); i++ {
		w += nums[i] - nums[i-k]
		if w > best {
			best = w
		}
	}
	return best
}

// 数组 L1：力量总和（单语法点：内置求和，参考解 1 行；12 号票批 4）。
func arraySumL1() question.Template {
	meta := question.Meta{ID: "array-sum-L1-attack", Topic: question.TopicArray,
		Difficulty: question.L1, Action: question.ActionAttack, Form: question.FormBoth}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(2, 5), -9, 15, question.ListOpts{})
		kind := rng.Scenario("words")
		sig := "def total_power(nums: list) -> int:"
		answer := "def total_power(nums: list) -> int:\n    return sum(nums)\n"
		body := []string{"    return sum(nums)"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("total_power(%s)", pyList(nums)), expected: fmt.Sprintf("%d", sumGo(nums))},
			caseAlias{name: "example-2", call: "total_power([1, 2, 3])", expected: "6"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "total_power([])", expected: "0"},
			caseAlias{name: "single", call: "total_power([7])", expected: "7"},
			caseAlias{name: "negatives", call: "total_power([-2, -3])", expected: "-5"},
			caseAlias{name: "zeros", call: "total_power([0, 0, 0])", expected: "0"},
			caseAlias{name: "mixed", call: "total_power([5, -1, 2])", expected: "6"},
		)
		stmt := fmt.Sprintf("%s战力合计：求列表所有元素之和（空列表为 0）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{0: true}, visible, hidden, answer)
	}}
}

// 数组 L3（关键格副本）：最大值下标（循环 + 严格大于保首个，参考解 5 行；12 号票批 4）。
func arrayMaxIndexL3() question.Template {
	meta := question.Meta{ID: "array-max-index-L3-attack", Topic: question.TopicArray,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"duplicates", "single-element", "all-negative"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), -9, 9, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def max_index(nums: list) -> int:"
		answer := "def max_index(nums: list) -> int:\n    best = 0\n    for i in range(len(nums)):\n        if nums[i] > nums[best]:\n            best = i\n    return best\n"
		body := []string{
			"    best = 0",
			"    for i in range(len(nums)):",
			"        if nums[i] > nums[best]:",
			"            best = i",
			"    return best",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("max_index(%s)", pyList(nums)), expected: fmt.Sprintf("%d", maxIndexGo(nums))},
			caseAlias{name: "example-2", call: "max_index([1, 3, 3])", expected: "1"},
		)
		hidden := cases(
			caseAlias{name: "duplicates", call: "max_index([4, 9, 9, 2])", expected: "1"},
			caseAlias{name: "single-element", call: "max_index([6])", expected: "0"},
			caseAlias{name: "all-negative", call: "max_index([-5, -2, -8])", expected: "1"},
			caseAlias{name: "last-max", call: "max_index([1, 2, 5])", expected: "2"},
			caseAlias{name: "first-max", call: "max_index([9, 1, 2])", expected: "0"},
		)
		stmt := fmt.Sprintf("%s定位最强目标：返回列表最大值的下标（并列取最先出现者；约定输入非空）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{2: true, 3: true}, visible, hidden, answer)
	}}
}

// 数组 L4（关键格副本）：循环右移（取模 + 哨兵数组，参考解 8 行；12 号票批 4）。
func arrayRotateL4() question.Template {
	meta := question.Meta{ID: "array-rotate-L4-attack", Topic: question.TopicArray,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "k-larger-than-n", "k-zero"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), 1, 9, question.ListOpts{})
		k := rng.Int(1, 9)
		kind := rng.Scenario("arena")
		sig := "def rotate_right(nums: list, k: int) -> list:"
		answer := "def rotate_right(nums: list, k: int) -> list:\n" +
			"    n = len(nums)\n" +
			"    if n == 0:\n" +
			"        return []\n" +
			"    k = k % n\n" +
			"    result = [0] * n\n" +
			"    for i in range(n):\n" +
			"        result[(i + k) % n] = nums[i]\n" +
			"    return result\n"
		body := []string{
			"    n = len(nums)",
			"    if n == 0:",
			"        return []",
			"    k = k % n",
			"    result = [0] * n",
			"    for i in range(n):",
			"        result[(i + k) % n] = nums[i]",
			"    return result",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("rotate_right(%s, %d)", pyList(nums), k), expected: pyList(rotateRightGo(nums, k))},
			caseAlias{name: "example-2", call: "rotate_right([1, 2, 3, 4, 5], 2)", expected: "[4, 5, 1, 2, 3]"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "rotate_right([], 3)", expected: "[]"},
			caseAlias{name: "k-larger-than-n", call: "rotate_right([1, 2, 3, 4, 5], 7)", expected: "[4, 5, 1, 2, 3]"},
			caseAlias{name: "k-zero", call: "rotate_right([1, 2, 3], 0)", expected: "[1, 2, 3]"},
			caseAlias{name: "k-equals-n", call: "rotate_right([1, 2], 2)", expected: "[1, 2]"},
			caseAlias{name: "single", call: "rotate_right([1], 3)", expected: "[1]"},
		)
		stmt := fmt.Sprintf("%s轮转阵列：把列表循环右移 k 位（k 可超过长度；空列表返回空）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{5: true, 6: true}, visible, hidden, answer)
	}}
}

// 数组 L5：最大子数组和（Kadane + 全负边界，参考解 12 行；12 号票批 4，
// 算法内核参考 codetop59 最大子数组和，题面经情境池重包装）。
func arrayKadaneL5() question.Template {
	meta := question.Meta{ID: "array-kadane-L5-attack", Topic: question.TopicArray,
		Difficulty: question.L5, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "single-negative", "all-negative", "all-positive"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(4, 7), -9, 9, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def max_subarray(nums: list) -> int:"
		answer := "def max_subarray(nums: list) -> int:\n" +
			"    if not nums:\n" +
			"        return 0\n" +
			"    best = nums[0]\n" +
			"    current = nums[0]\n" +
			"    for i in range(1, len(nums)):\n" +
			"        if current < 0:\n" +
			"            current = nums[i]\n" +
			"        else:\n" +
			"            current = current + nums[i]\n" +
			"        if current > best:\n" +
			"            best = current\n" +
			"    return best\n"
		body := []string{
			"    if not nums:",
			"        return 0",
			"    best = nums[0]",
			"    current = nums[0]",
			"    for i in range(1, len(nums)):",
			"        if current < 0:",
			"            current = nums[i]",
			"        else:",
			"            current = current + nums[i]",
			"        if current > best:",
			"            best = current",
			"    return best",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("max_subarray(%s)", pyList(nums)), expected: fmt.Sprintf("%d", kadaneGo(nums))},
			caseAlias{name: "example-2", call: "max_subarray([-2, 1, -3, 4, -1, 2, 1, -5, 4])", expected: "6"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "max_subarray([])", expected: "0"},
			caseAlias{name: "single-negative", call: "max_subarray([-5])", expected: "-5"},
			caseAlias{name: "all-negative", call: "max_subarray([-3, -1, -7])", expected: "-1"},
			caseAlias{name: "all-positive", call: "max_subarray([1, 2, 3])", expected: "6"},
			caseAlias{name: "classic", call: "max_subarray([-2, 1, -3, 4, -1, 2, 1, -5, 4])", expected: "6"},
			caseAlias{name: "restart-better", call: "max_subarray([5, -1, 2])", expected: "6"},
		)
		stmt := fmt.Sprintf("%s能量峰值：求连续子数组的最大和（空列表为 0；元素可全为负）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{6: true, 8: true}, visible, hidden, answer)
	}}
}

// 数组 L5（关键格副本）：子数组和等于 k 的个数（前缀和 + 哈希计数，参考解 13 行；12 号票批 4，
// 算法内核参考 codetop59 和为 K 的子数组，题面经情境池重包装）。
func arraySubarraySumL5() question.Template {
	meta := question.Meta{ID: "array-subarray-sum-L5-attack", Topic: question.TopicArray,
		Difficulty: question.L5, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "negative-numbers", "zero-k", "no-match"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), -5, 5, question.ListOpts{})
		k := rng.Int(-3, 5)
		kind := rng.Scenario("arena")
		sig := "def subarray_sum(nums: list, k: int) -> int:"
		answer := "def subarray_sum(nums: list, k: int) -> int:\n" +
			"    counts = {0: 1}\n" +
			"    total = 0\n" +
			"    result = 0\n" +
			"    for x in nums:\n" +
			"        total += x\n" +
			"        need = total - k\n" +
			"        if need in counts:\n" +
			"            result += counts[need]\n" +
			"        if total in counts:\n" +
			"            counts[total] += 1\n" +
			"        else:\n" +
			"            counts[total] = 1\n" +
			"    return result\n"
		body := []string{
			"    counts = {0: 1}",
			"    total = 0",
			"    result = 0",
			"    for x in nums:",
			"        total += x",
			"        need = total - k",
			"        if need in counts:",
			"            result += counts[need]",
			"        if total in counts:",
			"            counts[total] += 1",
			"        else:",
			"            counts[total] = 1",
			"    return result",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("subarray_sum(%s, %d)", pyList(nums), k), expected: fmt.Sprintf("%d", subarraySumGo(nums, k))},
			caseAlias{name: "example-2", call: "subarray_sum([1, 1, 1], 2)", expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "subarray_sum([], 0)", expected: "0"},
			caseAlias{name: "no-match", call: "subarray_sum([1, 2, 3], 7)", expected: "0"},
			caseAlias{name: "negative-numbers", call: "subarray_sum([1, -1, 0], 0)", expected: "3"},
			caseAlias{name: "zero-k", call: "subarray_sum([0, 0, 0], 0)", expected: "6"},
			caseAlias{name: "classic", call: "subarray_sum([3, 4, 7, 2, -3, 1, 4, 2], 7)", expected: "4"},
		)
		stmt := fmt.Sprintf("%s连续段计数：统计和恰好为 k 的连续子数组个数（元素可含负数）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{5: true, 7: true}, visible, hidden, answer)
	}}
}

// ---- 期望值计算（与参考解同语义） ----

func sumGo(nums []int) int {
	t := 0
	for _, v := range nums {
		t += v
	}
	return t
}

func maxIndexGo(nums []int) int {
	best := 0
	for i := range nums {
		if nums[i] > nums[best] {
			best = i
		}
	}
	return best
}

func rotateRightGo(nums []int, k int) []int {
	n := len(nums)
	if n == 0 {
		return []int{}
	}
	k %= n
	out := make([]int, n)
	for i := 0; i < n; i++ {
		out[(i+k)%n] = nums[i]
	}
	return out
}

func kadaneGo(nums []int) int {
	if len(nums) == 0 {
		return 0
	}
	best, current := nums[0], nums[0]
	for i := 1; i < len(nums); i++ {
		if current < 0 {
			current = nums[i]
		} else {
			current += nums[i]
		}
		if current > best {
			best = current
		}
	}
	return best
}

func subarraySumGo(nums []int, k int) int {
	counts := map[int]int{0: 1}
	total, result := 0, 0
	for _, x := range nums {
		total += x
		need := total - k
		result += counts[need]
		counts[total]++
	}
	return result
}
