// 攻击题·排序与贪心（3 模板：L2/L3/L4，黄金样板）。
package templates

import (
	"fmt"

	"codewar/server/internal/question"
)

// 排序 L2：奇偶重排（偶数在前、稳定，参考解 5 行）。
func sortParityL2() question.Template {
	meta := question.Meta{ID: "sort-parity-L2-attack", Topic: question.TopicSort,
		Difficulty: question.L2, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input"}, Golden: true}
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
		Traps: []string{"single-element", "duplicates"}, Golden: true}
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
		Traps: []string{"empty-input", "single-element", "already-sorted", "reverse-sorted"}, Golden: true}
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
