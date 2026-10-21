// 攻击题·排序与贪心（L2/L3/L4 黄金样板 + 12 号票批 6 扩容：L1/L5 + 关键格副本）。
package templates

import (
	"fmt"
	"sort"
	"strings"

	"codewar/server/internal/question"
)

// 排序 L2：奇偶重排（偶数在前、稳定，参考解 5 行）。
func sortParityL2() question.Template {
	meta := question.Meta{ID: "sort-parity-L2-attack", Topic: question.TopicSort,
		Difficulty: question.L2, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), 0, 9, question.ListOpts{})
		sig := "def sort_by_parity(nums: list) -> list:"
		// 参考解 5 行（L2 S=3-5 上限）：条件表达式处理空列表。
		body5 := []string{"    evens = [x for x in nums if x % 2 == 0]", "    odds = [x for x in nums if x % 2 != 0]",
			"    result = evens + odds", "    result = result if len(nums) > 0 else []", "    return result"}
		answer5 := "def sort_by_parity(nums: list) -> list:\n    evens = [x for x in nums if x % 2 == 0]\n    odds = [x for x in nums if x % 2 != 0]\n    result = evens + odds\n    result = result if len(nums) > 0 else []\n    return result\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("sort_by_parity(%s)", pyList(nums)), expected: pyList(parityGo(nums))},
			caseAlias{name: "example-2", call: "sort_by_parity([3, 1, 2, 4])", expected: "[2, 4, 3, 1]"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "sort_by_parity([])", expected: "[]"},
			caseAlias{name: "all-even", call: "sort_by_parity([2, 4, 6])", expected: "[2, 4, 6]"},
			caseAlias{name: "all-odd", call: "sort_by_parity([1, 3, 5])", expected: "[1, 3, 5]"},
			caseAlias{name: "single-even", call: "sort_by_parity([8])", expected: "[8]"},
			caseAlias{name: "single-odd", call: "sort_by_parity([7])", expected: "[7]"},
		)
		stmt := fmt.Sprintf("偶数在前、奇数在后（各自保持原相对顺序）。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body5, map[int]bool{0: true, 1: true}, visible, hidden, answer5)
	}}
}

// 排序 L3：第 K 小（排序 + 钳制，参考解 6 行）。
func sortKthL3() question.Template {
	meta := question.Meta{ID: "sort-kth-L3-attack", Topic: question.TopicSort,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"single-element", "duplicates"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		n := rng.Int(3, 6)
		nums := rng.IntList(n, 0, 20, question.ListOpts{})
		k := rng.Int(1, n)
		sig := "def kth_smallest(nums: list, k: int) -> int:"
		answer := "def kth_smallest(nums: list, k: int) -> int:\n    ordered = sorted(nums)\n    if k < 1:\n        k = 1\n    if k > len(ordered):\n        k = len(ordered)\n    return ordered[k - 1]\n"
		body := []string{"    ordered = sorted(nums)", "    if k < 1:", "        k = 1",
			"    if k > len(ordered):", "        k = len(ordered)", "    return ordered[k - 1]"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("kth_smallest(%s, %d)", pyList(nums), k), expected: fmt.Sprintf("%d", kthGo(nums, k))},
			caseAlias{name: "example-2", call: "kth_smallest([7, 10, 4, 3], 2)", expected: "4"},
		)
		hidden := cases(
			caseAlias{name: "single-element", call: "kth_smallest([9], 1)", expected: "9"},
			caseAlias{name: "duplicates", call: "kth_smallest([5, 5, 5, 5], 3)", expected: "5"},
			caseAlias{name: "k-first", call: "kth_smallest([3, 1, 2], 1)", expected: "1"},
			caseAlias{name: "k-last", call: "kth_smallest([3, 1, 2], 3)", expected: "3"},
			caseAlias{name: "negatives", call: "kth_smallest([-2, -8, 0, 5], 2)", expected: "-2"},
		)
		stmt := fmt.Sprintf("返回第 k 小的元素（k 从 1 起；约定 1 ≤ k ≤ len(nums)）。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{0: true, 5: true}, visible, hidden, answer)
	}}
}

// 排序 L4：插入排序（参考解 11 行）。
func sortInsertionL4() question.Template {
	meta := question.Meta{ID: "sort-insertion-L4-attack", Topic: question.TopicSort,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "single-element", "already-sorted", "reverse-sorted"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), 0, 20, question.ListOpts{})
		sig := "def insertion_sort(nums: list) -> list:"
		answer := "def insertion_sort(nums: list) -> list:\n    arr = list(nums)\n    for i in range(1, len(arr)):\n        key = arr[i]\n        j = i - 1\n        while j >= 0 and arr[j] > key:\n            arr[j + 1] = arr[j]\n            j -= 1\n        arr[j + 1] = key\n    if len(arr) == 0:\n        return []\n    return arr\n"
		body := []string{"    arr = list(nums)", "    for i in range(1, len(arr)):", "        key = arr[i]",
			"        j = i - 1", "        while j >= 0 and arr[j] > key:", "            arr[j + 1] = arr[j]",
			"            j -= 1", "        arr[j + 1] = key", "    if len(arr) == 0:", "        return []", "    return arr"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("insertion_sort(%s)", pyList(nums)), expected: pyList(sortedGo(nums))},
			caseAlias{name: "example-2", call: "insertion_sort([5, 2, 4, 1])", expected: "[1, 2, 4, 5]"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "insertion_sort([])", expected: "[]"},
			caseAlias{name: "single-element", call: "insertion_sort([3])", expected: "[3]"},
			caseAlias{name: "already-sorted", call: "insertion_sort([1, 2, 3, 4])", expected: "[1, 2, 3, 4]"},
			caseAlias{name: "reverse-sorted", call: "insertion_sort([4, 3, 2, 1])", expected: "[1, 2, 3, 4]"},
			caseAlias{name: "duplicates-sort", call: "insertion_sort([3, 1, 3, 2])", expected: "[1, 2, 3, 3]"},
		)
		stmt := fmt.Sprintf("实现插入排序，返回升序新列表（不修改输入）。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{4: true, 5: true}, visible, hidden, answer)
	}}
}

func parityGo(nums []int) []int {
	out := []int{}
	for _, v := range nums {
		if v%2 == 0 {
			out = append(out, v)
		}
	}
	for _, v := range nums {
		if v%2 != 0 {
			out = append(out, v)
		}
	}
	return out
}

func kthGo(nums []int, k int) int {
	cp := append([]int{}, nums...)
	for i := 1; i < len(cp); i++ {
		for j := i; j > 0 && cp[j] < cp[j-1]; j-- {
			cp[j], cp[j-1] = cp[j-1], cp[j]
		}
	}
	if k < 1 {
		k = 1
	}
	if k > len(cp) {
		k = len(cp)
	}
	return cp[k-1]
}

func sortedGo(nums []int) []int {
	cp := append([]int{}, nums...)
	for i := 1; i < len(cp); i++ {
		for j := i; j > 0 && cp[j] < cp[j-1]; j-- {
			cp[j], cp[j-1] = cp[j-1], cp[j]
		}
	}
	return cp
}

// 排序 L1：最小元素（单语法点：sorted + 下标，参考解 2 行；12 号票批 6）。
func sortSmallestL1() question.Template {
	meta := question.Meta{ID: "sort-smallest-L1-attack", Topic: question.TopicSort,
		Difficulty: question.L1, Action: question.ActionAttack, Form: question.FormBoth}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(2, 5), -9, 12, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def smallest(nums: list) -> int:"
		answer := "def smallest(nums: list) -> int:\n    ordered = sorted(nums)\n    return ordered[0]\n"
		body := []string{"    ordered = sorted(nums)", "    return ordered[0]"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("smallest(%s)", pyList(nums)), expected: fmt.Sprintf("%d", smallestGo(nums))},
			caseAlias{name: "example-2", call: "smallest([4, 2, 7])", expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "smallest([6])", expected: "6"},
			caseAlias{name: "sorted", call: "smallest([1, 2, 3])", expected: "1"},
			caseAlias{name: "reverse-sorted", call: "smallest([9, 5, 3])", expected: "3"},
			caseAlias{name: "duplicates", call: "smallest([4, 4, 4])", expected: "4"},
			caseAlias{name: "negatives", call: "smallest([-2, -8, 0])", expected: "-8"},
		)
		stmt := fmt.Sprintf("%s最弱一环：返回列表中的最小元素（约定非空）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{0: true}, visible, hidden, answer)
	}}
}

// 排序 L3（关键格副本）：最小相邻差（排序 + 相邻扫描，参考解 7 行；12 号票批 6）。
func sortMinGapL3() question.Template {
	meta := question.Meta{ID: "sort-min-gap-L3-attack", Topic: question.TopicSort,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"duplicates", "adjacent-pair", "unsorted-input"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), 1, 12, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def min_gap(nums: list) -> int:"
		answer := "def min_gap(nums: list) -> int:\n    ordered = sorted(nums)\n    best = ordered[1] - ordered[0]\n    for i in range(2, len(ordered)):\n        gap = ordered[i] - ordered[i - 1]\n        if gap < best:\n            best = gap\n    return best\n"
		body := []string{
			"    ordered = sorted(nums)",
			"    best = ordered[1] - ordered[0]",
			"    for i in range(2, len(ordered)):",
			"        gap = ordered[i] - ordered[i - 1]",
			"        if gap < best:",
			"            best = gap",
			"    return best",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("min_gap(%s)", pyList(nums)), expected: fmt.Sprintf("%d", minGapGo(nums))},
			caseAlias{name: "example-2", call: "min_gap([4, 2, 7])", expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "duplicates", call: "min_gap([5, 5])", expected: "0"},
			caseAlias{name: "adjacent-pair", call: "min_gap([1, 9])", expected: "8"},
			caseAlias{name: "unsorted-input", call: "min_gap([10, 1, 5])", expected: "4"},
			caseAlias{name: "even-spread", call: "min_gap([1, 3, 6, 10])", expected: "2"},
			caseAlias{name: "all-same", call: "min_gap([2, 2, 2])", expected: "0"},
		)
		stmt := fmt.Sprintf("%s最小间隙：返回排序后相邻两元素差值的最小值（约定至少 2 个元素）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{3: true, 5: true}, visible, hidden, answer)
	}}
}

// 排序 L4（关键格副本）：合并区间（排序 + 贪心合并，参考解 11 行；12 号票批 6，
// 算法内核参考 codetop59 合并区间，题面经情境池重包装）。
func sortMergeIntervalsL4() question.Template {
	meta := question.Meta{ID: "sort-merge-intervals-L4-attack", Topic: question.TopicSort,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "touching", "contained", "unsorted-input"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		intervals := randRanges(rng, rng.Int(2, 4))
		kind := rng.Scenario("arena")
		sig := "def merge_intervals(intervals: list) -> list:"
		answer := "def merge_intervals(intervals: list) -> list:\n" +
			"    if not intervals:\n" +
			"        return []\n" +
			"    ordered = sorted(intervals)\n" +
			"    merged = [ordered[0][:]]\n" +
			"    for start, end in ordered[1:]:\n" +
			"        if start <= merged[-1][1]:\n" +
			"            if end > merged[-1][1]:\n" +
			"                merged[-1][1] = end\n" +
			"        else:\n" +
			"            merged.append([start, end])\n" +
			"    return merged\n"
		body := []string{
			"    if not intervals:",
			"        return []",
			"    ordered = sorted(intervals)",
			"    merged = [ordered[0][:]]",
			"    for start, end in ordered[1:]:",
			"        if start <= merged[-1][1]:",
			"            if end > merged[-1][1]:",
			"                merged[-1][1] = end",
			"        else:",
			"            merged.append([start, end])",
			"    return merged",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("merge_intervals(%s)", pyRanges(intervals)), expected: pyRanges(mergeRangesGo(intervals))},
			caseAlias{name: "example-2", call: "merge_intervals([[1, 3], [2, 6], [8, 10], [15, 18]])", expected: "[[1, 6], [8, 10], [15, 18]]"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "merge_intervals([])", expected: "[]"},
			caseAlias{name: "touching", call: "merge_intervals([[1, 4], [4, 5]])", expected: "[[1, 5]]"},
			caseAlias{name: "contained", call: "merge_intervals([[1, 4], [2, 3]])", expected: "[[1, 4]]"},
			caseAlias{name: "unsorted-input", call: "merge_intervals([[5, 6], [1, 2]])", expected: "[[1, 2], [5, 6]]"},
			caseAlias{name: "single", call: "merge_intervals([[1, 10]])", expected: "[[1, 10]]"},
		)
		stmt := fmt.Sprintf("%s占领区合并：合并所有重叠或相接的区间（start ≤ 上一区间 end 即合并），按起点升序返回。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{5: true, 6: true}, visible, hidden, answer)
	}}
}

// 排序 L5：有序数组平方（双指针从两端取大，参考解 12 行；12 号票批 6，
// 算法内核参考 codetop59 有序数组的平方，题面经情境池重包装）。
func sortSortedSquaresL5() question.Template {
	meta := question.Meta{ID: "sort-sorted-squares-L5-attack", Topic: question.TopicSort,
		Difficulty: question.L5, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"all-negative", "all-positive", "mixed", "single-element"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), -8, 8, question.ListOpts{Sorted: true})
		kind := rng.Scenario("arena")
		sig := "def sorted_squares(nums: list) -> list:"
		answer := "def sorted_squares(nums: list) -> list:\n" +
			"    n = len(nums)\n" +
			"    result = [0] * n\n" +
			"    left = 0\n" +
			"    right = n - 1\n" +
			"    for i in range(n - 1, -1, -1):\n" +
			"        if abs(nums[left]) > abs(nums[right]):\n" +
			"            result[i] = nums[left] * nums[left]\n" +
			"            left += 1\n" +
			"        else:\n" +
			"            result[i] = nums[right] * nums[right]\n" +
			"            right -= 1\n" +
			"    return result\n"
		body := []string{
			"    n = len(nums)",
			"    result = [0] * n",
			"    left = 0",
			"    right = n - 1",
			"    for i in range(n - 1, -1, -1):",
			"        if abs(nums[left]) > abs(nums[right]):",
			"            result[i] = nums[left] * nums[left]",
			"            left += 1",
			"        else:",
			"            result[i] = nums[right] * nums[right]",
			"            right -= 1",
			"    return result",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("sorted_squares(%s)", pyList(nums)), expected: pyList(sortedSquaresGo(nums))},
			caseAlias{name: "example-2", call: "sorted_squares([-4, -1, 0, 3, 10])", expected: "[0, 1, 9, 16, 100]"},
		)
		hidden := cases(
			caseAlias{name: "all-negative", call: "sorted_squares([-5, -4, -3])", expected: "[9, 16, 25]"},
			caseAlias{name: "all-positive", call: "sorted_squares([1, 2, 3])", expected: "[1, 4, 9]"},
			caseAlias{name: "mixed", call: "sorted_squares([-3, -1, 0, 2])", expected: "[0, 1, 4, 9]"},
			caseAlias{name: "single-element", call: "sorted_squares([7])", expected: "[49]"},
			caseAlias{name: "negative-single", call: "sorted_squares([-2])", expected: "[4]"},
			caseAlias{name: "zeros", call: "sorted_squares([0, 0])", expected: "[0, 0]"},
		)
		stmt := fmt.Sprintf("%s能量平方：输入为非递减整数列表（可含负数），返回每个元素平方后的非递减列表。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{5: true, 6: true}, visible, hidden, answer)
	}}
}

// 排序 L5（关键格副本）：最少箭数（按右端点排序 + 贪心穿刺，参考解 13 行；12 号票批 6，
// 算法内核参考 codetop59 用最少数量的箭引爆气球，题面经情境池重包装）。
func sortMinArrowsL5() question.Template {
	meta := question.Meta{ID: "sort-min-arrows-L5-attack", Topic: question.TopicSort,
		Difficulty: question.L5, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "touching-points", "disjoint", "single-balloon"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		points := randRanges(rng, rng.Int(2, 4))
		kind := rng.Scenario("arena")
		sig := "def min_arrows(points: list) -> int:"
		answer := "def min_arrows(points: list) -> int:\n" +
			"    if not points:\n" +
			"        return 0\n" +
			"    ordered = []\n" +
			"    for p in points:\n" +
			"        ordered.append([p[0], p[1]])\n" +
			"    ordered.sort(key=lambda p: p[1])\n" +
			"    arrows = 1\n" +
			"    last = ordered[0][1]\n" +
			"    for start, end in ordered[1:]:\n" +
			"        if start > last:\n" +
			"            arrows += 1\n" +
			"            last = end\n" +
			"    return arrows\n"
		body := []string{
			"    if not points:",
			"        return 0",
			"    ordered = []",
			"    for p in points:",
			"        ordered.append([p[0], p[1]])",
			"    ordered.sort(key=lambda p: p[1])",
			"    arrows = 1",
			"    last = ordered[0][1]",
			"    for start, end in ordered[1:]:",
			"        if start > last:",
			"            arrows += 1",
			"            last = end",
			"    return arrows",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("min_arrows(%s)", pyRanges(points)), expected: fmt.Sprintf("%d", minArrowsGo(points))},
			caseAlias{name: "example-2", call: "min_arrows([[10, 16], [2, 8], [1, 6], [7, 12]])", expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "min_arrows([])", expected: "0"},
			caseAlias{name: "touching-points", call: "min_arrows([[1, 2], [2, 3]])", expected: "1"},
			caseAlias{name: "disjoint", call: "min_arrows([[1, 2], [3, 4]])", expected: "2"},
			caseAlias{name: "single-balloon", call: "min_arrows([[1, 2]])", expected: "1"},
			caseAlias{name: "chain", call: "min_arrows([[1, 2], [2, 3], [3, 4], [4, 5]])", expected: "2"},
			caseAlias{name: "nested", call: "min_arrows([[1, 10], [2, 3], [4, 5]])", expected: "2"},
		)
		stmt := fmt.Sprintf("%s气球穿刺：每个区间是一支气球的水平范围，箭沿竖线穿刺，求最少箭数（端点相接可同箭）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{5: true, 9: true}, visible, hidden, answer)
	}}
}

// ---- 期望值计算（与参考解同语义） ----

func smallestGo(nums []int) int {
	m := nums[0]
	for _, v := range nums[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func minGapGo(nums []int) int {
	cp := sortedGo(nums)
	best := cp[1] - cp[0]
	for i := 2; i < len(cp); i++ {
		if cp[i]-cp[i-1] < best {
			best = cp[i] - cp[i-1]
		}
	}
	return best
}

// randRanges：随机小区间（start < end），用于区间类题目。
func randRanges(rng *question.RNG, n int) [][2]int {
	out := make([][2]int, n)
	for i := range out {
		s := rng.Int(0, 8)
		out[i] = [2]int{s, s + rng.Int(1, 4)}
	}
	return out
}

// pyRanges：区间列表 Python repr（如 "[[1, 3], [2, 6]]"）。
func pyRanges(rs [][2]int) string {
	parts := make([]string, len(rs))
	for i, r := range rs {
		parts[i] = fmt.Sprintf("[%d, %d]", r[0], r[1])
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func mergeRangesGo(rs [][2]int) [][2]int {
	if len(rs) == 0 {
		return [][2]int{}
	}
	cp := append([][2]int{}, rs...)
	sort.Slice(cp, func(i, j int) bool {
		if cp[i][0] != cp[j][0] {
			return cp[i][0] < cp[j][0]
		}
		return cp[i][1] < cp[j][1]
	})
	out := [][2]int{cp[0]}
	for _, r := range cp[1:] {
		if r[0] <= out[len(out)-1][1] {
			if r[1] > out[len(out)-1][1] {
				out[len(out)-1][1] = r[1]
			}
		} else {
			out = append(out, r)
		}
	}
	return out
}

func sortedSquaresGo(nums []int) []int {
	n := len(nums)
	out := make([]int, n)
	l, r := 0, n-1
	for i := n - 1; i >= 0; i-- {
		if absGo(nums[l]) > absGo(nums[r]) {
			out[i] = nums[l] * nums[l]
			l++
		} else {
			out[i] = nums[r] * nums[r]
			r--
		}
	}
	return out
}

func minArrowsGo(points [][2]int) int {
	if len(points) == 0 {
		return 0
	}
	cp := append([][2]int{}, points...)
	sort.Slice(cp, func(i, j int) bool { return cp[i][1] < cp[j][1] })
	arrows := 1
	last := cp[0][1]
	for _, p := range cp[1:] {
		if p[0] > last {
			arrows++
			last = p[1]
		}
	}
	return arrows
}
