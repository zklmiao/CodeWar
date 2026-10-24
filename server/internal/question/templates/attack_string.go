// 攻击题·字符串处理（L2/L3/L4 黄金样板 + L5 十二号票扩容）。
package templates

import (
	"fmt"
	"strconv"
	"strings"

	"codewar/server/internal/question"
)

// 字符串 L2：回文判断（切片 + 比较，参考解 5 行）。
func stringPalindromeL2() question.Template {
	meta := question.Meta{ID: "string-palindrome-L2-attack", Topic: question.TopicString,
		Difficulty: question.L2, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "case-sensitive"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		word := rng.Pick([]string{"level", "abba", "racecar", "hello", "Abba", "a"})
		sig := "def is_palindrome(s: str) -> bool:"
		answer := "def is_palindrome(s: str) -> bool:\n    cleaned = s\n    reversed_s = cleaned[::-1]\n    if cleaned == reversed_s:\n        return True\n    return False\n"
		body := []string{"    cleaned = s", "    reversed_s = cleaned[::-1]", "    if cleaned == reversed_s:", "        return True", "    return False"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("is_palindrome(%s)", pyStr(word)), expected: pyBool(isPal(word))},
			caseAlias{name: "example-2", call: `is_palindrome("hello")`, expected: "False"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `is_palindrome("")`, expected: "True"},
			caseAlias{name: "single-char", call: `is_palindrome("a")`, expected: "True"},
			caseAlias{name: "case-sensitive", call: `is_palindrome("Abba")`, expected: "False"},
			caseAlias{name: "even-palindrome", call: `is_palindrome("abba")`, expected: "True"},
			caseAlias{name: "not-palindrome", call: `is_palindrome("world")`, expected: "False"},
		)
		stmt := fmt.Sprintf("判断字符串是否为回文（区分大小写，空串视为回文）。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{1: true, 2: true}, visible, hidden, answer)
	}}
}

// 字符串 L3：元音计数（循环 + 成员判断，参考解 8 行）。
func stringVowelCountL3() question.Template {
	meta := question.Meta{ID: "string-vowel-count-L3-attack", Topic: question.TopicString,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "no-vowel"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		n := rng.Int(4, 8)
		s := rng.Str(n, "lower")
		sig := "def count_vowels(s: str) -> int:"
		answer := "def count_vowels(s: str) -> int:\n    vowels = 'aeiouAEIOU'\n    count = 0\n    for ch in s:\n        if ch in vowels:\n            count += 1\n    if len(s) == 0:\n        return 0\n    return count\n"
		body := []string{"    vowels = 'aeiouAEIOU'", "    count = 0", "    for ch in s:", "        if ch in vowels:", "            count += 1", "    if len(s) == 0:", "        return 0", "    return count"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("count_vowels(%s)", pyStr(s)), expected: fmt.Sprintf("%d", countVowels(s))},
			caseAlias{name: "example-2", call: `count_vowels("hello")`, expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `count_vowels("")`, expected: "0"},
			caseAlias{name: "no-vowel", call: `count_vowels("xyz")`, expected: "0"},
			caseAlias{name: "all-vowels", call: `count_vowels("aeiou")`, expected: "5"},
			caseAlias{name: "uppercase", call: `count_vowels("AbE")`, expected: "2"},
			caseAlias{name: "mixed", call: `count_vowels("code123")`, expected: "2"},
		)
		stmt := fmt.Sprintf("统计字符串中元音字母（aeiou 大小写）个数。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{3: true, 4: true}, visible, hidden, answer)
	}}
}

// 字符串 L4：游程压缩（边界密集，参考解 12 行）。
func stringRunLengthL4() question.Template {
	meta := question.Meta{ID: "string-run-length-L4-attack", Topic: question.TopicString,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "single-char", "long-run"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		alpha := rng.Pick([]string{"alpha", "binary"})
		n := rng.Int(3, 6)
		s := rng.Str(n, alpha)
		sig := "def compress(s: str) -> str:"
		answer := "def compress(s: str) -> str:\n    if len(s) == 0:\n        return ''\n    res = ''\n    count = 1\n    for i in range(1, len(s)):\n        if s[i] == s[i - 1]:\n            count += 1\n        else:\n            res += s[i - 1] + str(count)\n            count = 1\n    res += s[-1] + str(count)\n    return res\n"
		body := []string{"    if len(s) == 0:", "        return ''", "    res = ''", "    count = 1",
			"    for i in range(1, len(s)):", "        if s[i] == s[i - 1]:", "            count += 1",
			"        else:", "            res += s[i - 1] + str(count)", "            count = 1",
			"    res += s[-1] + str(count)", "    return res"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("compress(%s)", pyStr(s)), expected: pyStr(runLength(s))},
			caseAlias{name: "example-2", call: `compress("aabcc")`, expected: pyStr("a2b1c2")},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `compress("")`, expected: pyStr("")},
			caseAlias{name: "single-char", call: `compress("x")`, expected: pyStr("x1")},
			caseAlias{name: "no-repeat", call: `compress("abc")`, expected: pyStr("a1b1c1")},
			caseAlias{name: "long-run", call: `compress("aaaa")`, expected: pyStr("a4")},
			caseAlias{name: "mixed-run", call: `compress("aaabbbc")`, expected: pyStr("a3b3c1")},
		)
		stmt := fmt.Sprintf("字符串压缩：连续字符按「字符+次数」输出（次数为 1 也要输出）。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{5: true, 8: true}, visible, hidden, answer)
	}}
}

func isPal(s string) bool {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		if s[i] != s[j] {
			return false
		}
	}
	return true
}

func pyBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

func countVowels(s string) int {
	n := 0
	for _, ch := range s {
		if strings.ContainsRune("aeiouAEIOU", ch) {
			n++
		}
	}
	return n
}

func runLength(s string) string {
	if len(s) == 0 {
		return ""
	}
	var b strings.Builder
	count := 1
	for i := 1; i < len(s); i++ {
		if s[i] == s[i-1] {
			count++
		} else {
			fmt.Fprintf(&b, "%c%d", s[i-1], count)
			count = 1
		}
	}
	fmt.Fprintf(&b, "%c%d", s[len(s)-1], count)
	return b.String()
}

// 字符串 L5：最长回文子串（中心扩展 + 双指针双形式，参考解 13 行；12 号票批 1 高档示范，
// 算法内核参考 codetop59 最长回文子串，题面经情境池重包装）。
func stringPalindromeSubstrL5() question.Template {
	meta := question.Meta{ID: "string-palindrome-substr-L5-attack", Topic: question.TopicString,
		Difficulty: question.L5, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "single-char", "all-same", "tie-first"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		drawn := rng.Str(rng.Int(4, 7), "binary")
		kind := rng.Scenario("rune")
		sig := "def longest_palindrome(s: str) -> str:"
		answer := "def longest_palindrome(s: str) -> str:\n" +
			"    if not s:\n" +
			"        return \"\"\n" +
			"    best_start = 0\n" +
			"    best_len = 0\n" +
			"    for i in range(len(s)):\n" +
			"        for left, right in ((i, i), (i, i + 1)):\n" +
			"            while left >= 0 and right < len(s) and s[left] == s[right]:\n" +
			"                left -= 1\n" +
			"                right += 1\n" +
			"            if right - left - 1 > best_len:\n" +
			"                best_start = left + 1\n" +
			"                best_len = right - left - 1\n" +
			"    return s[best_start:best_start + best_len]\n"
		body := []string{
			"    if not s:",
			"        return \"\"",
			"    best_start = 0",
			"    best_len = 0",
			"    for i in range(len(s)):",
			"        for left, right in ((i, i), (i, i + 1)):",
			"            while left >= 0 and right < len(s) and s[left] == s[right]:",
			"                left -= 1",
			"                right += 1",
			"            if right - left - 1 > best_len:",
			"                best_start = left + 1",
			"                best_len = right - left - 1",
			"    return s[best_start:best_start + best_len]",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("longest_palindrome(%s)", pyStr(drawn)), expected: pyStr(longestPalindromeGo(drawn))},
			caseAlias{name: "example-2", call: `longest_palindrome("cbbd")`, expected: pyStr("bb")},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `longest_palindrome("")`, expected: pyStr("")},
			caseAlias{name: "single-char", call: `longest_palindrome("x")`, expected: pyStr("x")},
			caseAlias{name: "all-same", call: `longest_palindrome("aaaa")`, expected: pyStr("aaaa")},
			caseAlias{name: "tie-first", call: `longest_palindrome("babad")`, expected: pyStr("bab")},
			caseAlias{name: "even-center", call: `longest_palindrome("abba")`, expected: pyStr("abba")},
			caseAlias{name: "odd-center", call: `longest_palindrome("racecar")`, expected: pyStr("racecar")},
			caseAlias{name: "no-repeat", call: `longest_palindrome("abc")`, expected: pyStr("a")},
		)
		stmt := fmt.Sprintf("修复断裂的%s：返回字符串中最长的回文子串（并列取最先出现者；空串返回空串）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{6: true, 9: true}, visible, hidden, answer)
	}}
}

// longestPalindromeGo：期望值计算（与参考解同语义：枚举中心 + 奇偶双扩展，严格大于保首个）。
func longestPalindromeGo(s string) string {
	bestStart, bestLen := 0, 0
	for i := 0; i < len(s); i++ {
		for _, pair := range [2][2]int{{i, i}, {i, i + 1}} {
			l, r := pair[0], pair[1]
			for l >= 0 && r < len(s) && s[l] == s[r] {
				l--
				r++
			}
			if r-l-1 > bestLen {
				bestStart, bestLen = l+1, r-l-1
			}
		}
	}
	return s[bestStart : bestStart+bestLen]
}

// 字符串 L1：战斗呐喊（单语法点：upper + 拼接，参考解 1 行；12 号票批 3）。
func stringShoutL1() question.Template {
	meta := question.Meta{ID: "string-shout-L1-attack", Topic: question.TopicString,
		Difficulty: question.L1, Action: question.ActionAttack, Form: question.FormBoth}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		word := rng.Str(rng.Int(3, 5), "lower")
		sig := "def shout(s: str) -> str:"
		answer := "def shout(s: str) -> str:\n    return s.upper() + '!'\n"
		body := []string{"    return s.upper() + '!'"}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("shout(%s)", pyStr(word)), expected: pyStr(strings.ToUpper(word) + "!")},
			caseAlias{name: "example-2", call: `shout("go")`, expected: pyStr("GO!")},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `shout("")`, expected: pyStr("!")},
			caseAlias{name: "already-upper", call: `shout("GO")`, expected: pyStr("GO!")},
			caseAlias{name: "mixed-case", call: `shout("Go")`, expected: pyStr("GO!")},
			caseAlias{name: "digits", call: `shout("12")`, expected: pyStr("12!")},
			caseAlias{name: "single", call: `shout("a")`, expected: pyStr("A!")},
		)
		stmt := fmt.Sprintf("战斗呐喊：把呐喊词转为全大写并追加感叹号。\n%s", examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{0: true}, visible, hidden, answer)
	}}
}

// 字符串 L3（关键格副本）：反转单词顺序（分割 + 过滤空段，参考解 6 行；12 号票批 3，
// 算法内核参考 codetop59 反转字符串中的单词，题面经情境池重包装）。
func stringReverseWordsL3() question.Template {
	meta := question.Meta{ID: "string-reverse-words-L3-attack", Topic: question.TopicString,
		Difficulty: question.L3, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"multi-space", "single-word", "empty-input"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		drawn := rng.Str(rng.Int(2, 3), "alpha") + " " + rng.Str(rng.Int(2, 3), "alpha")
		kind := rng.Scenario("rune")
		sig := "def reverse_words(s: str) -> str:"
		answer := "def reverse_words(s: str) -> str:\n" +
			"    words = []\n" +
			"    for w in s.split(' '):\n" +
			"        if w != '':\n" +
			"            words.append(w)\n" +
			"    words.reverse()\n" +
			"    return ' '.join(words)\n"
		body := []string{
			"    words = []",
			"    for w in s.split(' '):",
			"        if w != '':",
			"            words.append(w)",
			"    words.reverse()",
			"    return ' '.join(words)",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("reverse_words(%s)", pyStr(drawn)), expected: pyStr(reverseWordsGo(drawn))},
			caseAlias{name: "example-2", call: `reverse_words("the sky is blue")`, expected: pyStr("blue is sky the")},
		)
		hidden := cases(
			caseAlias{name: "multi-space", call: `reverse_words("a  good   example")`, expected: pyStr("example good a")},
			caseAlias{name: "single-word", call: `reverse_words("hello")`, expected: pyStr("hello")},
			caseAlias{name: "empty-input", call: `reverse_words("")`, expected: pyStr("")},
			caseAlias{name: "leading-trailing", call: `reverse_words("  hi  ")`, expected: pyStr("hi")},
			caseAlias{name: "two-words", call: `reverse_words("a b")`, expected: pyStr("b a")},
		)
		stmt := fmt.Sprintf("%s语序颠倒：把句子按单词反转（多余空格剔除，单词间单空格）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{3: true, 4: true}, visible, hidden, answer)
	}}
}

// 字符串 L4（关键格副本）：版本号比较（分割 + 补齐 + 多段比较，参考解 11 行；12 号票批 3，
// 算法内核参考 codetop59 比较版本号，题面经情境池重包装）。
func stringVersionCompareL4() question.Template {
	meta := question.Meta{ID: "string-version-compare-L4-attack", Topic: question.TopicString,
		Difficulty: question.L4, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"equal", "leading-zeros", "different-length", "less-than"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		a, b := rng.Int(0, 5), rng.Int(0, 5)
		kind := rng.Scenario("rune")
		sig := "def compare_versions(v1: str, v2: str) -> int:"
		answer := "def compare_versions(v1: str, v2: str) -> int:\n" +
			"    parts1 = v1.split('.')\n" +
			"    parts2 = v2.split('.')\n" +
			"    n = max(len(parts1), len(parts2))\n" +
			"    for i in range(n):\n" +
			"        a = int(parts1[i]) if i < len(parts1) else 0\n" +
			"        b = int(parts2[i]) if i < len(parts2) else 0\n" +
			"        if a < b:\n" +
			"            return -1\n" +
			"        if a > b:\n" +
			"            return 1\n" +
			"    return 0\n"
		body := []string{
			"    parts1 = v1.split('.')",
			"    parts2 = v2.split('.')",
			"    n = max(len(parts1), len(parts2))",
			"    for i in range(n):",
			"        a = int(parts1[i]) if i < len(parts1) else 0",
			"        b = int(parts2[i]) if i < len(parts2) else 0",
			"        if a < b:",
			"            return -1",
			"        if a > b:",
			"            return 1",
			"    return 0",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("compare_versions('%d.1', '%d.2')", a, b), expected: fmt.Sprintf("%d", compareVersionsGo(fmt.Sprintf("%d.1", a), fmt.Sprintf("%d.2", b)))},
			caseAlias{name: "example-2", call: "compare_versions('2.1', '1.9.9')", expected: "1"},
		)
		hidden := cases(
			caseAlias{name: "equal", call: "compare_versions('1.0', '1.0')", expected: "0"},
			caseAlias{name: "leading-zeros", call: "compare_versions('1.01', '1.001')", expected: "0"},
			caseAlias{name: "different-length", call: "compare_versions('1.0', '1')", expected: "0"},
			caseAlias{name: "less-than", call: "compare_versions('0.1', '1.1')", expected: "-1"},
			caseAlias{name: "multi-seg", call: "compare_versions('1.2.3.4', '1.2.3')", expected: "1"},
		)
		stmt := fmt.Sprintf("%s版次比对：按 '.' 分段逐段比较版本号（前导零忽略；缺段按 0 处理；返回 1/-1/0）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{4: true, 5: true}, visible, hidden, answer)
	}}
}

// 字符串 L5（关键格副本）：无重复字符最长子串（滑动窗口 + 集合，参考解 12 行；12 号票批 3，
// 算法内核参考 codetop59 无重复字符的最长子串，题面经情境池重包装）。
func stringUniqueWindowL5() question.Template {
	meta := question.Meta{ID: "string-unique-window-L5-attack", Topic: question.TopicString,
		Difficulty: question.L5, Action: question.ActionAttack, Form: question.FormBoth,
		Traps: []string{"empty-input", "single-char", "all-same", "all-distinct"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		drawn := rng.Str(rng.Int(4, 7), "binary")
		kind := rng.Scenario("rune")
		sig := "def longest_unique(s: str) -> int:"
		answer := "def longest_unique(s: str) -> int:\n" +
			"    seen = set()\n" +
			"    left = 0\n" +
			"    best = 0\n" +
			"    for right in range(len(s)):\n" +
			"        while s[right] in seen:\n" +
			"            seen.discard(s[left])\n" +
			"            left = left + 1\n" +
			"        seen.add(s[right])\n" +
			"        window = right - left + 1\n" +
			"        if window > best:\n" +
			"            best = window\n" +
			"    return best\n"
		body := []string{
			"    seen = set()",
			"    left = 0",
			"    best = 0",
			"    for right in range(len(s)):",
			"        while s[right] in seen:",
			"            seen.discard(s[left])",
			"            left = left + 1",
			"        seen.add(s[right])",
			"        window = right - left + 1",
			"        if window > best:",
			"            best = window",
			"    return best",
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("longest_unique(%s)", pyStr(drawn)), expected: fmt.Sprintf("%d", longestUniqueGo(drawn))},
			caseAlias{name: "example-2", call: `longest_unique("pwwkew")`, expected: "3"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `longest_unique("")`, expected: "0"},
			caseAlias{name: "single-char", call: `longest_unique("a")`, expected: "1"},
			caseAlias{name: "all-same", call: `longest_unique("bbbb")`, expected: "1"},
			caseAlias{name: "all-distinct", call: `longest_unique("abcdef")`, expected: "6"},
			caseAlias{name: "repeats", call: `longest_unique("abcabcbb")`, expected: "3"},
			caseAlias{name: "tricky-shift", call: `longest_unique("dvdf")`, expected: "3"},
		)
		stmt := fmt.Sprintf("%s铭环测量：字符串中最长的不含重复字符的连续子串长度（空串为 0）。\n%s", kind, examplesText(visible))
		return assemble(meta, "", stmt, sig, body, map[int]bool{4: true, 8: true}, visible, hidden, answer)
	}}
}

// reverseWordsGo：期望值计算（与参考解同语义：按空格分割、滤空、反转、单空格连接）。
func reverseWordsGo(s string) string {
	var words []string
	for _, w := range strings.Split(s, " ") {
		if w != "" {
			words = append(words, w)
		}
	}
	for i, j := 0, len(words)-1; i < j; i, j = i+1, j-1 {
		words[i], words[j] = words[j], words[i]
	}
	return strings.Join(words, " ")
}

// compareVersionsGo：期望值计算（与参考解同语义：分段、右补 0、逐段比较）。
func compareVersionsGo(v1, v2 string) int {
	p1 := strings.Split(v1, ".")
	p2 := strings.Split(v2, ".")
	n := len(p1)
	if len(p2) > n {
		n = len(p2)
	}
	for i := 0; i < n; i++ {
		a, b := 0, 0
		if i < len(p1) {
			a, _ = strconv.Atoi(p1[i])
		}
		if i < len(p2) {
			b, _ = strconv.Atoi(p2[i])
		}
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
	}
	return 0
}

// longestUniqueGo：期望值计算（与参考解同语义：滑动窗口收缩左界）。
func longestUniqueGo(s string) int {
	seen := map[byte]bool{}
	left, best := 0, 0
	for right := 0; right < len(s); right++ {
		for seen[s[right]] {
			delete(seen, s[left])
			left++
		}
		seen[s[right]] = true
		if right-left+1 > best {
			best = right - left + 1
		}
	}
	return best
}
