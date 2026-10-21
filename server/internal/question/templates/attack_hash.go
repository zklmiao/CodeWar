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

// 哈希 L1：命中判定（单语法点：成员判断，参考解 1 行；12 号票批 5）。
func hashContainsL1() question.Template {
	meta := question.Meta{ID: "hash-contains-L1-attack", Topic: question.TopicHash,
		Difficulty: question.L1, Action: question.ActionAttack, Form: question.FormBoth}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(2, 5), -5, 9, question.ListOpts{})
		target := rng.Int(-5, 9)
		kind := rng.Scenario("arena")
		sig := "def contains(nums: list, target: int) -> bool:"
		answer := "def contains(nums: list, target: int) -> bool:\n    return target in nums\n"
		body := []string{"    return target in nums"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("contains(%s, %d)", pyList(nums), target), expected: pyBool(hasIntGo(nums, target))},
			caseAlias{name: "example-2", call: "contains([3, 1, 4], 4)", expected: "True"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "contains([], 3)", expected: "False"},
			caseAlias{name: "single-hit", call: "contains([7], 7)", expected: "True"},
			caseAlias{name: "single-miss", call: "contains([7], 8)", expected: "False"},
			caseAlias{name: "duplicates", call: "contains([2, 2, 2], 2)", expected: "True"},
			caseAlias{name: "negative", call: "contains([-3, 5], -3)", expected: "True"},
		)
		stmt := fmt.Sprintf("%s命中判定：判断目标值是否在列表中（空列表为 False）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{0: true}, visible, hidden, answer)
	}}
}

// 哈希 L3（关键格副本）：两数配对存在性（集合 + 补数查找，参考解 6 行；12 号票批 5，
// 算法内核参考 codetop59 两数之和，题面经情境池重包装）。
func hashPairSumL3() question.Template {
	meta := question.Meta{ID: "hash-pair-sum-L3-attack", Topic: question.TopicHash,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "no-pair", "duplicates"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), -9, 9, question.ListOpts{})
		target := rng.Int(-8, 12)
		kind := rng.Scenario("arena")
		sig := "def has_pair(nums: list, target: int) -> bool:"
		answer := "def has_pair(nums: list, target: int) -> bool:\n    seen = set()\n    for x in nums:\n        if target - x in seen:\n            return True\n        seen.add(x)\n    return False\n"
		body := []string{
			"    seen = set()",
			"    for x in nums:",
			"        if target - x in seen:",
			"            return True",
			"        seen.add(x)",
			"    return False",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("has_pair(%s, %d)", pyList(nums), target), expected: pyBool(hasPairGo(nums, target))},
			caseAlias{name: "example-2", call: "has_pair([1, 2, 3], 5)", expected: "True"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "has_pair([], 0)", expected: "False"},
			caseAlias{name: "no-pair", call: "has_pair([1, 2, 3], 6)", expected: "False"},
			caseAlias{name: "duplicates", call: "has_pair([3, 3], 6)", expected: "True"},
			caseAlias{name: "single-element", call: "has_pair([5], 10)", expected: "False"},
			caseAlias{name: "negatives", call: "has_pair([-1, 1], 0)", expected: "True"},
		)
		stmt := fmt.Sprintf("%s双人合击：判断列表中是否存在两个不同下标元素之和等于 target（空列表为 False）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{2: true, 4: true}, visible, hidden, answer)
	}}
}

// 哈希 L4（关键格副本）：异位词判定（长度守卫 + 计数增减，参考解 10 行；12 号票批 5）。
func hashAnagramL4() question.Template {
	meta := question.Meta{ID: "hash-anagram-L4-attack", Topic: question.TopicHash,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"different-length", "empty-input", "repeated-chars", "no-match"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		word := rng.Str(rng.Int(3, 5), "alpha")
		kind := rng.Scenario("rune")
		sig := "def is_anagram(a: str, b: str) -> bool:"
		answer := "def is_anagram(a: str, b: str) -> bool:\n" +
			"    if len(a) != len(b):\n" +
			"        return False\n" +
			"    counts = {}\n" +
			"    for ch in a:\n" +
			"        counts[ch] = counts.get(ch, 0) + 1\n" +
			"    for ch in b:\n" +
			"        if ch not in counts or counts[ch] == 0:\n" +
			"            return False\n" +
			"        counts[ch] -= 1\n" +
			"    return True\n"
		body := []string{
			"    if len(a) != len(b):",
			"        return False",
			"    counts = {}",
			"    for ch in a:",
			"        counts[ch] = counts.get(ch, 0) + 1",
			"    for ch in b:",
			"        if ch not in counts or counts[ch] == 0:",
			"            return False",
			"        counts[ch] -= 1",
			"    return True",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("is_anagram(%s, %s)", pyStr(word), pyStr(reverseStr(word))), expected: pyBool(isAnagramGo(word, reverseStr(word)))},
			caseAlias{name: "example-2", call: `is_anagram("rat", "car")`, expected: "False"},
		)
		hidden := cases(
			caseAlias{name: "different-length", call: `is_anagram("a", "ab")`, expected: "False"},
			caseAlias{name: "empty-input", call: `is_anagram("", "")`, expected: "True"},
			caseAlias{name: "repeated-chars", call: `is_anagram("aab", "aba")`, expected: "True"},
			caseAlias{name: "no-match", call: `is_anagram("aacc", "ccac")`, expected: "False"},
			caseAlias{name: "classic", call: `is_anagram("abc", "cba")`, expected: "True"},
		)
		stmt := fmt.Sprintf("%s铭文异位：判断两个字符串是否为异位词（字符种类与数量完全一致；空串为 True）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{4: true, 6: true}, visible, hidden, answer)
	}}
}

// 哈希 L5：最长连续序列（集合去重 + 起点扩展，参考解 13 行；12 号票批 5，
// 算法内核参考 codetop59 最长连续序列，题面经情境池重包装）。
func hashLongestConsecutiveL5() question.Template {
	meta := question.Meta{ID: "hash-longest-consecutive-L5-attack", Topic: question.TopicHash,
		Difficulty: question.L5, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "all-same", "no-consecutive", "negative"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(4, 7), -5, 9, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def longest_consecutive(nums: list) -> int:"
		answer := "def longest_consecutive(nums: list) -> int:\n" +
			"    values = set(nums)\n" +
			"    best = 0\n" +
			"    for x in values:\n" +
			"        if x - 1 in values:\n" +
			"            continue\n" +
			"        current = x\n" +
			"        length = 1\n" +
			"        while current + 1 in values:\n" +
			"            current += 1\n" +
			"            length += 1\n" +
			"        if length > best:\n" +
			"            best = length\n" +
			"    return best\n"
		body := []string{
			"    values = set(nums)",
			"    best = 0",
			"    for x in values:",
			"        if x - 1 in values:",
			"            continue",
			"        current = x",
			"        length = 1",
			"        while current + 1 in values:",
			"            current += 1",
			"            length += 1",
			"        if length > best:",
			"            best = length",
			"    return best",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("longest_consecutive(%s)", pyList(nums)), expected: fmt.Sprintf("%d", longestConsecutiveGo(nums))},
			caseAlias{name: "example-2", call: "longest_consecutive([100, 4, 200, 1, 3, 2])", expected: "4"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "longest_consecutive([])", expected: "0"},
			caseAlias{name: "all-same", call: "longest_consecutive([1, 1, 1])", expected: "1"},
			caseAlias{name: "no-consecutive", call: "longest_consecutive([10, 5, 20])", expected: "1"},
			caseAlias{name: "negative", call: "longest_consecutive([-2, -1, 0, 5])", expected: "3"},
			caseAlias{name: "unsorted", call: "longest_consecutive([1, 2, 0, 1])", expected: "3"},
			caseAlias{name: "gap", call: "longest_consecutive([1, 2, 4, 5, 6])", expected: "3"},
		)
		stmt := fmt.Sprintf("%s连续序列：求数组中最长连续整数序列的长度（不要求原序相邻；空数组为 0）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{3: true, 7: true}, visible, hidden, answer)
	}}
}

// 哈希 L5（关键格副本）：异位词最少删除（双向计数差 + 绝对值汇总，参考解 12 行；12 号票批 5，
// 算法内核参考 codetop59 两个字符串的删除操作，题面经情境池重包装）。
func hashMinStepsL5() question.Template {
	meta := question.Meta{ID: "hash-min-steps-L5-attack", Topic: question.TopicHash,
		Difficulty: question.L5, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "identical", "no-overlap", "one-empty"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		word := rng.Str(rng.Int(3, 5), "alpha")
		kind := rng.Scenario("rune")
		sig := "def min_steps(a: str, b: str) -> int:"
		answer := "def min_steps(a: str, b: str) -> int:\n" +
			"    counts = {}\n" +
			"    for ch in a:\n" +
			"        counts[ch] = counts.get(ch, 0) + 1\n" +
			"    for ch in b:\n" +
			"        counts[ch] = counts.get(ch, 0) - 1\n" +
			"    steps = 0\n" +
			"    for v in counts.values():\n" +
			"        if v > 0:\n" +
			"            steps += v\n" +
			"        elif v < 0:\n" +
			"            steps -= v\n" +
			"    return steps\n"
		body := []string{
			"    counts = {}",
			"    for ch in a:",
			"        counts[ch] = counts.get(ch, 0) + 1",
			"    for ch in b:",
			"        counts[ch] = counts.get(ch, 0) - 1",
			"    steps = 0",
			"    for v in counts.values():",
			"        if v > 0:",
			"            steps += v",
			"        elif v < 0:",
			"            steps -= v",
			"    return steps",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("min_steps(%s, %s)", pyStr(word), pyStr(reverseStr(word))), expected: fmt.Sprintf("%d", minStepsGo(word, reverseStr(word)))},
			caseAlias{name: "example-2", call: `min_steps("aab", "abb")`, expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `min_steps("", "")`, expected: "0"},
			caseAlias{name: "identical", call: `min_steps("a", "a")`, expected: "0"},
			caseAlias{name: "no-overlap", call: `min_steps("a", "b")`, expected: "2"},
			caseAlias{name: "one-empty", call: `min_steps("", "ab")`, expected: "2"},
			caseAlias{name: "classic", call: `min_steps("abc", "cba")`, expected: "0"},
			caseAlias{name: "count-diff", call: `min_steps("aaa", "a")`, expected: "2"},
		)
		stmt := fmt.Sprintf("%s铭文对齐：最少删除多少个字符能使两串互为异位词（只能删字符）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{2: true, 4: true}, visible, hidden, answer)
	}}
}

// ---- 期望值计算（与参考解同语义） ----

func hasIntGo(nums []int, t int) bool {
	for _, v := range nums {
		if v == t {
			return true
		}
	}
	return false
}

func hasPairGo(nums []int, target int) bool {
	seen := map[int]bool{}
	for _, x := range nums {
		if seen[target-x] {
			return true
		}
		seen[x] = true
	}
	return false
}

func reverseStr(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

func isAnagramGo(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := map[byte]int{}
	for i := 0; i < len(a); i++ {
		counts[a[i]]++
	}
	for i := 0; i < len(b); i++ {
		if counts[b[i]] == 0 {
			return false
		}
		counts[b[i]]--
	}
	return true
}

func longestConsecutiveGo(nums []int) int {
	values := map[int]bool{}
	for _, v := range nums {
		values[v] = true
	}
	best := 0
	for x := range values {
		if values[x-1] {
			continue
		}
		current, length := x, 1
		for values[current+1] {
			current++
			length++
		}
		if length > best {
			best = length
		}
	}
	return best
}

func minStepsGo(a, b string) int {
	counts := map[byte]int{}
	for i := 0; i < len(a); i++ {
		counts[a[i]]++
	}
	for i := 0; i < len(b); i++ {
		counts[b[i]]--
	}
	steps := 0
	for _, v := range counts {
		if v > 0 {
			steps += v
		} else {
			steps -= v
		}
	}
	return steps
}
