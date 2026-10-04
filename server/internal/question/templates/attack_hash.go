// 攻击题·哈希与计数（3 模板：L2/L3/L4，黄金样板）。
package templates

import (
	"fmt"

	"codewar/server/internal/question"
)

// 哈希 L2：计数目标值（参考解 5 行）。
func hashCountL2() question.Template {
	meta := question.Meta{ID: "hash-count-L2-attack", Topic: question.TopicHash,
		Difficulty: question.L2, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "absent"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), 1, 5, question.ListOpts{})
		target := rng.Int(1, 5)
		sig := "def count_occurrences(nums: list, target: int) -> int:"
		answer := "def count_occurrences(nums: list, target: int) -> int:\n    count = 0\n    for x in nums:\n        if x == target:\n            count += 1\n    return count\n"
		body := []string{"    count = 0", "    for x in nums:", "        if x == target:", "            count += 1", "    return count"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("count_occurrences(%s, %d)", pyList(nums), target), expected: fmt.Sprintf("%d", countOcc(nums, target))},
			caseAlias{name: "example-2", call: "count_occurrences([1, 2, 2, 3], 2)", expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: fmt.Sprintf("count_occurrences([], %d)", target), expected: "0"},
			caseAlias{name: "absent", call: "count_occurrences([1, 2, 3], 9)", expected: "0"},
			caseAlias{name: "all-hit", call: "count_occurrences([4, 4, 4], 4)", expected: "3"},
			caseAlias{name: "single-hit", call: "count_occurrences([7], 7)", expected: "1"},
			caseAlias{name: "single-miss", call: "count_occurrences([7], 8)", expected: "0"},
		)
		stmt := fmt.Sprintf("统计 target 在列表中的出现次数。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{2: true, 3: true}, visible, hidden, answer)
	}}
}

// 哈希 L3：最高频单词（平局取首次出现，参考解 8 行；约定 words 非空）。
func hashWordFreqL3() question.Template {
	meta := question.Meta{ID: "hash-word-frequency-L3-attack", Topic: question.TopicHash,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"repeated-max", "single-distinct"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		pool := []string{"arena", "blade", "code", "storm", "quest"}
		n := rng.Int(4, 7)
		words := rng.PickN(pool, n)
		sig := "def most_common_word(words: list) -> str:"
		answer := "def most_common_word(words: list) -> str:\n    freq = {}\n    for w in words:\n        freq[w] = freq.get(w, 0) + 1\n    best = words[0]\n    for w in words:\n        if freq[w] > freq[best]:\n            best = w\n    return best\n"
		body := []string{"    freq = {}", "    for w in words:", "        freq[w] = freq.get(w, 0) + 1",
			"    best = words[0]", "    for w in words:", "        if freq[w] > freq[best]:", "            best = w", "    return best"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("most_common_word(%s)", pyStrList(words)), expected: pyStr(mostCommon(words))},
			caseAlias{name: "example-2", call: `most_common_word(["a", "b", "a"])`, expected: pyStr("a")},
		)
		hidden := cases(
			caseAlias{name: "repeated-max", call: `most_common_word(["x", "y", "x", "y", "x"])`, expected: pyStr("x")},
			caseAlias{name: "repeated-max-tie", call: `most_common_word(["p", "q", "p", "q"])`, expected: pyStr("p")},
			caseAlias{name: "single-distinct", call: `most_common_word(["z", "z", "z"])`, expected: pyStr("z")},
			caseAlias{name: "single-element", call: `most_common_word(["solo"])`, expected: pyStr("solo")},
			caseAlias{name: "late-winner", call: `most_common_word(["a", "b", "b", "b"])`, expected: pyStr("b")},
		)
		stmt := fmt.Sprintf("返回列表中出现次数最多的单词（并列取首次出现者；约定输入非空）。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{2: true, 5: true}, visible, hidden, answer)
	}}
}

// 哈希 L4：Top-K 高频（频率降序、并列值小优先，参考解 11 行；约定 nums 非空、k≥1）。
func hashTopKL4() question.Template {
	meta := question.Meta{ID: "hash-top-k-L4-attack", Topic: question.TopicHash,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"tie-order", "single-distinct", "k-equals-distinct"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(5, 8), 1, 5, question.ListOpts{})
		d := distinctCount(nums)
		k := rng.Int(1, d)
		sig := "def top_k_frequent(nums: list, k: int) -> list:"
		answer := "def top_k_frequent(nums: list, k: int) -> list:\n    freq = {}\n    for x in nums:\n        freq[x] = freq.get(x, 0) + 1\n    items = list(freq.keys())\n    items.sort(key=lambda v: (-freq[v], v))\n    if k > len(items):\n        k = len(items)\n    result = []\n    for i in range(k):\n        result.append(items[i])\n    return result\n"
		body := []string{"    freq = {}", "    for x in nums:", "        freq[x] = freq.get(x, 0) + 1",
			"    items = list(freq.keys())", "    items.sort(key=lambda v: (-freq[v], v))",
			"    if k > len(items):", "        k = len(items)", "    result = []",
			"    for i in range(k):", "        result.append(items[i])", "    return result"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("top_k_frequent(%s, %d)", pyList(nums), k), expected: pyList(topKGo(nums, k))},
			caseAlias{name: "example-2", call: "top_k_frequent([1, 1, 2, 2, 3], 2)", expected: "[1, 2]"},
		)
		hidden := cases(
			caseAlias{name: "tie-order", call: "top_k_frequent([2, 2, 1, 1, 3], 2)", expected: "[1, 2]"},
			caseAlias{name: "single-distinct", call: "top_k_frequent([5, 5, 5], 1)", expected: "[5]"},
			caseAlias{name: "k-equals-distinct", call: "top_k_frequent([3, 1, 2], 3)", expected: "[1, 2, 3]"},
			caseAlias{name: "k-one", call: "top_k_frequent([4, 4, 1, 2], 1)", expected: "[4]"},
			caseAlias{name: "freq-desc", call: "top_k_frequent([1, 2, 2, 3, 3, 3], 2)", expected: "[3, 2]"},
		)
		stmt := fmt.Sprintf("返回出现频率前 k 高的元素（频率降序，并列取较小值；约定输入非空、k≥1）。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{4: true, 9: true}, visible, hidden, answer)
	}}
}

func countOcc(nums []int, t int) int {
	n := 0
	for _, v := range nums {
		if v == t {
			n++
		}
	}
	return n
}

func mostCommon(words []string) string {
	freq := map[string]int{}
	for _, w := range words {
		freq[w]++
	}
	best := words[0]
	for _, w := range words {
		if freq[w] > freq[best] {
			best = w
		}
	}
	return best
}

func distinctCount(nums []int) int {
	s := map[int]bool{}
	for _, v := range nums {
		s[v] = true
	}
	return len(s)
}

func topKGo(nums []int, k int) []int {
	freq := map[int]int{}
	for _, v := range nums {
		freq[v]++
	}
	items := make([]int, 0, len(freq))
	for v := range freq {
		items = append(items, v)
	}
	for i := 1; i < len(items); i++ {
		for j := i; j > 0; j-- {
			a, b := items[j], items[j-1]
			if freq[a] > freq[b] || (freq[a] == freq[b] && a < b) {
				items[j], items[j-1] = items[j-1], items[j]
			} else {
				break
			}
		}
	}
	if k > len(items) {
		k = len(items)
	}
	out := append([]int{}, items[:k]...)
	return out
}
