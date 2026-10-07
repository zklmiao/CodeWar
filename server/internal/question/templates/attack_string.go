// 攻击题·字符串处理（L2/L3/L4 黄金样板 + L5 十二号票扩容）。
package templates

import (
	"fmt"
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
