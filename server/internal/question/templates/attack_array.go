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
