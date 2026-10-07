// 私有行动题·字符串主题（12 号票批 3：move L1/L2/L4/L5；defend/skill 见本文件后续提交）。
// 形式均为 complete；题型标注进入矩阵格（coverage_test.go 口径）。
package templates

import (
	"fmt"
	"strings"

	"codewar/server/internal/question"
)

// 移动 L1·字符串：指令首步（切片 + 查表兜底，参考解 3 行）。
func moveFirstStepL1() question.Template {
	meta := question.Meta{ID: "move-firststep-L1-move", Topic: question.TopicString,
		Difficulty: question.L1, Action: question.ActionMove, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		moves := rng.Str(rng.Int(2, 5), "moves")
		kind := rng.Scenario("rune")
		sig := "def first_step(moves: str) -> int:"
		answer := "def first_step(moves: str) -> int:\n    dx = {'R': 1, 'L': -1}\n    return dx.get(moves[:1], 0)\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("first_step(%s)", pyStr(moves)), expected: fmt.Sprintf("%d", firstStepGo(moves))},
			caseAlias{name: "example-2", call: `first_step("RR")`, expected: "1"},
		)
		hidden := cases(
			caseAlias{name: "right", call: `first_step("R")`, expected: "1"},
			caseAlias{name: "left", call: `first_step("L")`, expected: "-1"},
			caseAlias{name: "other-char", call: `first_step("U")`, expected: "0"},
			caseAlias{name: "empty-input", call: `first_step("")`, expected: "0"},
			caseAlias{name: "lower-case", call: `first_step("r")`, expected: "0"},
		)
		stmt := fmt.Sprintf("%s指令首步：取指令串第一个字符，R 计 +1、L 计 −1、其余计 0（空串为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L2·字符串：直行计数（循环 + 计数，参考解 5 行）。
func moveCountForwardL2() question.Template {
	meta := question.Meta{ID: "move-count-forward-L2-move", Topic: question.TopicString,
		Difficulty: question.L2, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "none-forward"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		moves := strings.NewReplacer("a", "F", "b", "U").Replace(rng.Str(rng.Int(2, 6), "binary"))
		kind := rng.Scenario("rune")
		sig := "def count_forward(moves: str) -> int:"
		answer := "def count_forward(moves: str) -> int:\n    steps = 0\n    for m in moves:\n        if m == 'F':\n            steps += 1\n    return steps\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("count_forward(%s)", pyStr(moves)), expected: fmt.Sprintf("%d", countForwardGo(moves))},
			caseAlias{name: "example-2", call: `count_forward("FFU")`, expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `count_forward("")`, expected: "0"},
			caseAlias{name: "all-forward", call: `count_forward("FFF")`, expected: "3"},
			caseAlias{name: "none-forward", call: `count_forward("UUU")`, expected: "0"},
			caseAlias{name: "mixed", call: `count_forward("FUUF")`, expected: "2"},
			caseAlias{name: "single", call: `count_forward("F")`, expected: "1"},
		)
		stmt := fmt.Sprintf("%s行军计数：指令串中 'F' 为前进一步，其余字符原地，统计前进总步数。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L4·字符串：转向解析（枚举 + 循环转向 + 取模，参考解 8 行）。
func moveFacingL4() question.Template {
	meta := question.Meta{ID: "move-facing-L4-move", Topic: question.TopicString,
		Difficulty: question.L4, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "full-circle", "unknown-char"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		start := rng.Pick([]string{"N", "E", "S", "W"})
		turns := rng.Str(rng.Int(2, 5), "binary") // L/R 两字母 alphabet
		turns = strings.ReplaceAll(turns, "a", "L")
		turns = strings.ReplaceAll(turns, "b", "R")
		sig := "def final_facing(start: str, turns: str) -> str:"
		answer := "def final_facing(start: str, turns: str) -> str:\n    dirs = ['N', 'E', 'S', 'W']\n    idx = dirs.index(start)\n    for t in turns:\n        if t == 'R':\n            idx = (idx + 1) % 4\n        elif t == 'L':\n            idx = (idx + 3) % 4\n    return dirs[idx]\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("final_facing(%s, %s)", pyStr(start), pyStr(turns)), expected: pyStr(string(facingGo(start, turns)))},
			caseAlias{name: "example-2", call: `final_facing("N", "RR")`, expected: pyStr("S")},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `final_facing("N", "")`, expected: pyStr("N")},
			caseAlias{name: "full-circle", call: `final_facing("N", "RRRR")`, expected: pyStr("N")},
			caseAlias{name: "quarter", call: `final_facing("N", "RRL")`, expected: pyStr("E")},
			caseAlias{name: "left-turn", call: `final_facing("N", "L")`, expected: pyStr("W")},
			caseAlias{name: "five-turns", call: `final_facing("E", "RRRRR")`, expected: pyStr("S")},
			caseAlias{name: "unknown-char", call: `final_facing("N", "RX")`, expected: pyStr("E")},
		)
		stmt := fmt.Sprintf("转向解析：朝向按 N→E→S→W 顺时针循环，R 右转 90°、L 左转 90°（未知字符忽略），返回最终朝向。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L5·字符串：信标定位（走位模拟 + 撤销栈，参考解 13 行）。
func moveBeaconL5() question.Template {
	meta := question.Meta{ID: "move-beacon-L5-move", Topic: question.TopicString,
		Difficulty: question.L5, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "undo-empty", "undo-back", "all-back"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		moves := strings.NewReplacer("a", "F", "b", "B").Replace(rng.Str(rng.Int(3, 6), "binary"))
		kind := rng.Scenario("rune")
		sig := "def beacon_end(moves: str) -> int:"
		answer := "def beacon_end(moves: str) -> int:\n" +
			"    pos = 0\n" +
			"    history = []\n" +
			"    for m in moves:\n" +
			"        if m == 'F':\n" +
			"            pos += 1\n" +
			"            history.append(1)\n" +
			"        elif m == 'B':\n" +
			"            pos -= 1\n" +
			"            history.append(-1)\n" +
			"        elif m == 'X':\n" +
			"            if history:\n" +
			"                pos -= history.pop()\n" +
			"    return pos\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("beacon_end(%s)", pyStr(moves)), expected: fmt.Sprintf("%d", beaconGo(moves))},
			caseAlias{name: "example-2", call: `beacon_end("FFXX")`, expected: "0"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `beacon_end("")`, expected: "0"},
			caseAlias{name: "undo-empty", call: `beacon_end("X")`, expected: "0"},
			caseAlias{name: "undo-forward", call: `beacon_end("FFX")`, expected: "1"},
			caseAlias{name: "undo-back", call: `beacon_end("FBX")`, expected: "1"},
			caseAlias{name: "all-back", call: `beacon_end("BBB")`, expected: "-3"},
			caseAlias{name: "mixed-undo", call: `beacon_end("BFX")`, expected: "-1"},
			caseAlias{name: "no-undo", call: `beacon_end("FFB")`, expected: "1"},
		)
		stmt := fmt.Sprintf("%s信标定位：F 前进一格、B 后退一格、X 撤销上一次移动（无可撤销则忽略），返回最终坐标。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// firstStepGo：期望值计算（与参考解同语义：首字符查表，未知/空为 0）。
func firstStepGo(moves string) int {
	if len(moves) > 0 && moves[0] == 'R' {
		return 1
	}
	if len(moves) > 0 && moves[0] == 'L' {
		return -1
	}
	return 0
}

// countForwardGo：期望值计算（与参考解同语义）。
func countForwardGo(moves string) int {
	n := 0
	for i := 0; i < len(moves); i++ {
		if moves[i] == 'F' {
			n++
		}
	}
	return n
}

// facingGo：期望值计算（与参考解同语义：N/E/S/W 顺时针环转）。
func facingGo(start, turns string) byte {
	dirs := "NESW"
	idx := strings.IndexByte(dirs, start[0])
	for i := 0; i < len(turns); i++ {
		if turns[i] == 'R' {
			idx = (idx + 1) % 4
		} else if turns[i] == 'L' {
			idx = (idx + 3) % 4
		}
	}
	return dirs[idx]
}

// beaconGo：期望值计算（与参考解同语义：F/B 走位 + X 弹栈撤销）。
func beaconGo(moves string) int {
	pos := 0
	var history []int
	for i := 0; i < len(moves); i++ {
		switch moves[i] {
		case 'F':
			pos++
			history = append(history, 1)
		case 'B':
			pos--
			history = append(history, -1)
		case 'X':
			if len(history) > 0 {
				pos -= history[len(history)-1]
				history = history[:len(history)-1]
			}
		}
	}
	return pos
}

// ---- 防御题·字符串主题（12 号票批 3b：L1-L5） ----

// 防御 L1·字符串：徽记判定（切片 + 比较，参考解 2 行）。
func defendBadgeL1() question.Template {
	meta := question.Meta{ID: "defend-badge-L1-defend", Topic: question.TopicString,
		Difficulty: question.L1, Action: question.ActionDefend, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		word := rng.Str(rng.Int(2, 5), "lower")
		kind := rng.Scenario("rune")
		sig := "def has_guard_prefix(s: str) -> bool:"
		answer := "def has_guard_prefix(s: str) -> bool:\n    return s[:1] == '#'\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("has_guard_prefix(%s)", pyStr("#"+word)), expected: "True"},
			caseAlias{name: "example-2", call: "has_guard_prefix('abc')", expected: "False"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `has_guard_prefix("")`, expected: "False"},
			caseAlias{name: "only-guard", call: `has_guard_prefix("#")`, expected: "True"},
			caseAlias{name: "double-guard", call: `has_guard_prefix("##")`, expected: "True"},
			caseAlias{name: "space-first", call: `has_guard_prefix(" #")`, expected: "False"},
			caseAlias{name: "plain", call: fmt.Sprintf("has_guard_prefix(%s)", pyStr(word)), expected: "False"},
		)
		stmt := fmt.Sprintf("%s徽记判定：字符串以 '#' 开头返回 True，否则 False（空串为 False）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L2·字符串：护符遮蔽（长度分支 + 重复拼接，参考解 4 行）。
func defendMaskL2() question.Template {
	meta := question.Meta{ID: "defend-mask-L2-defend", Topic: question.TopicString,
		Difficulty: question.L2, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "short-input"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		word := rng.Str(rng.Int(3, 7), "lower")
		kind := rng.Scenario("rune")
		sig := "def mask_tail(s: str) -> str:"
		answer := "def mask_tail(s: str) -> str:\n    n = len(s)\n    if n <= 4:\n        return '*' * n\n    return '*' * (n - 4) + s[-4:]\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("mask_tail(%s)", pyStr(word)), expected: pyStr(maskTailGo(word))},
			caseAlias{name: "example-2", call: `mask_tail("abcdefg")`, expected: pyStr("***defg")},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `mask_tail("")`, expected: pyStr("")},
			caseAlias{name: "short-input", call: `mask_tail("abc")`, expected: pyStr("***")},
			caseAlias{name: "exact-four", call: `mask_tail("abcd")`, expected: pyStr("****")},
			caseAlias{name: "five", call: `mask_tail("abcde")`, expected: pyStr("*bcde")},
			caseAlias{name: "long", call: `mask_tail("abcdefgh")`, expected: pyStr("****efgh")},
		)
		stmt := fmt.Sprintf("%s护符遮蔽：保留字符串最后 4 位，其余全部替换为 '*'（不足 4 位则全遮）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L3·字符串：禁语计数（循环 + 相等判断，参考解 5 行）。
func defendCensorL3() question.Template {
	meta := question.Meta{ID: "defend-censor-L3-defend", Topic: question.TopicString,
		Difficulty: question.L3, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "no-match", "all-match"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		word := rng.Str(rng.Int(3, 6), "lower")
		kind := rng.Scenario("rune")
		sig := "def censor_count(s: str, bad: str) -> int:"
		answer := "def censor_count(s: str, bad: str) -> int:\n    count = 0\n    for ch in s:\n        if ch == bad:\n            count += 1\n    return count\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("censor_count(%s, 'a')", pyStr(word)), expected: fmt.Sprintf("%d", strings.Count(word, "a"))},
			caseAlias{name: "example-2", call: `censor_count("axbxc", "x")`, expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `censor_count("", "x")`, expected: "0"},
			caseAlias{name: "no-match", call: `censor_count("abc", "z")`, expected: "0"},
			caseAlias{name: "all-match", call: `censor_count("xxx", "x")`, expected: "3"},
			caseAlias{name: "case-sensitive", call: `censor_count("aAa", "a")`, expected: "2"},
			caseAlias{name: "drawn", call: fmt.Sprintf("censor_count(%s, 'e')", pyStr(word)), expected: fmt.Sprintf("%d", strings.Count(word, "e"))},
		)
		stmt := fmt.Sprintf("%s禁语检测：统计字符串中禁用字符（区分大小写）出现的次数。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L4·字符串：古卷回文校验（清洗 + 双指针，参考解 12 行；
// 算法内核参考 codetop59 验证回文串，题面经情境池重包装）。
func defendScrollL4() question.Template {
	meta := question.Meta{ID: "defend-scroll-L4-defend", Topic: question.TopicString,
		Difficulty: question.L4, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "punctuation-only", "mixed-case", "odd-length"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		drawn := rng.Pick([]string{"Madam, I'm Adam!", "race a car", "no lemon, no melon", "hello world", "A man, a plan, a canal: Panama", "12321!"})
		kind := rng.Scenario("rune")
		sig := "def scroll_palindrome(s: str) -> bool:"
		answer := "def scroll_palindrome(s: str) -> bool:\n" +
			"    cleaned = ''\n" +
			"    for ch in s:\n" +
			"        if ch.isalnum():\n" +
			"            cleaned += ch.lower()\n" +
			"    left = 0\n" +
			"    right = len(cleaned) - 1\n" +
			"    while left < right:\n" +
			"        if cleaned[left] != cleaned[right]:\n" +
			"            return False\n" +
			"        left += 1\n" +
			"        right -= 1\n" +
			"    return True\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("scroll_palindrome(%s)", pyStr(drawn)), expected: pyBool(scrollPalindromeGo(drawn))},
			caseAlias{name: "example-2", call: `scroll_palindrome("race a car")`, expected: "False"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `scroll_palindrome("")`, expected: "True"},
			caseAlias{name: "punctuation-only", call: `scroll_palindrome(".,!")`, expected: "True"},
			caseAlias{name: "mixed-case", call: `scroll_palindrome("Aa")`, expected: "True"},
			caseAlias{name: "digit-vs-letter", call: `scroll_palindrome("0P")`, expected: "False"},
			caseAlias{name: "odd-length", call: `scroll_palindrome("Madam")`, expected: "True"},
			caseAlias{name: "spaces-ignored", call: `scroll_palindrome("never odd or even")`, expected: "True"},
		)
		stmt := fmt.Sprintf("%s古卷校验：忽略大小写与所有非字母数字字符后，判断该串是否为回文（空串视为回文）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L5·字符串：结界段测量（栈匹配 + 跨度统计，参考解 14 行；
// 算法内核为最长有效括号子串，题面经情境池重包装）。
func defendBarrierL5() question.Template {
	meta := question.Meta{ID: "defend-barrier-L5-defend", Topic: question.TopicString,
		Difficulty: question.L5, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "all-open", "all-close", "concatenated"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		drawn := rng.Pick([]string{"()(())", "(()", "()()", ")()())", "((()))", "())("})
		kind := rng.Scenario("rune")
		sig := "def barrier_span(s: str) -> int:"
		answer := "def barrier_span(s: str) -> int:\n" +
			"    best = 0\n" +
			"    stack = [-1]\n" +
			"    for i, ch in enumerate(s):\n" +
			"        if ch == '(':\n" +
			"            stack.append(i)\n" +
			"        else:\n" +
			"            stack.pop()\n" +
			"            if not stack:\n" +
			"                stack.append(i)\n" +
			"            else:\n" +
			"                span = i - stack[-1]\n" +
			"                if span > best:\n" +
			"                    best = span\n" +
			"    return best\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("barrier_span(%s)", pyStr(drawn)), expected: fmt.Sprintf("%d", barrierSpanGo(drawn))},
			caseAlias{name: "example-2", call: `barrier_span("(()")`, expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `barrier_span("")`, expected: "0"},
			caseAlias{name: "all-open", call: `barrier_span("(((")`, expected: "0"},
			caseAlias{name: "all-close", call: `barrier_span(")))")`, expected: "0"},
			caseAlias{name: "concatenated", call: `barrier_span("()()")`, expected: "4"},
			caseAlias{name: "trailing-break", call: `barrier_span(")()())")`, expected: "4"},
			caseAlias{name: "single-pair", call: `barrier_span("()")`, expected: "2"},
		)
		stmt := fmt.Sprintf("%s结界段测量：'(' 与 ')' 组成的结界串中，最长连续有效段（匹配闭合）的长度（空串为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// ---- 技能题·字符串主题（12 号票批 3b：L1-L5） ----

// 技能 L1·字符串：符文回响（重复运算符，参考解 1 行）。
func skillEchoL1() question.Template {
	meta := question.Meta{ID: "skill-echo-L1-skill", Topic: question.TopicString,
		Difficulty: question.L1, Action: question.ActionSkill, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		word := rng.Str(rng.Int(2, 3), "lower")
		n := rng.Int(1, 4)
		kind := rng.Scenario("spell")
		sig := "def echo_rune(s: str, n: int) -> str:"
		answer := "def echo_rune(s: str, n: int) -> str:\n    return s * n\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("echo_rune(%s, %d)", pyStr(word), n), expected: pyStr(strings.Repeat(word, n))},
			caseAlias{name: "example-2", call: `echo_rune("ab", 3)`, expected: pyStr("ababab")},
		)
		hidden := cases(
			caseAlias{name: "zero-repeat", call: `echo_rune("ab", 0)`, expected: pyStr("")},
			caseAlias{name: "single-repeat", call: `echo_rune("ab", 1)`, expected: pyStr("ab")},
			caseAlias{name: "empty-input", call: `echo_rune("", 3)`, expected: pyStr("")},
			caseAlias{name: "single-char", call: `echo_rune("x", 2)`, expected: pyStr("xx")},
			caseAlias{name: "drawn", call: fmt.Sprintf("echo_rune(%s, %d)", pyStr(word), 2), expected: pyStr(strings.Repeat(word, 2))},
		)
		stmt := fmt.Sprintf("%s回响：把符文串重复 n 次输出（n=0 为空串）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L2·字符串：奇位汲取（循环 + 下标判断，参考解 5 行）。
func skillOddL2() question.Template {
	meta := question.Meta{ID: "skill-odd-L2-skill", Topic: question.TopicString,
		Difficulty: question.L2, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-char"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		word := rng.Str(rng.Int(3, 6), "lower")
		kind := rng.Scenario("spell")
		sig := "def odd_chars(s: str) -> str:"
		answer := "def odd_chars(s: str) -> str:\n    result = ''\n    for i in range(len(s)):\n        if i % 2 == 1:\n            result += s[i]\n    return result\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("odd_chars(%s)", pyStr(word)), expected: pyStr(oddCharsGo(word))},
			caseAlias{name: "example-2", call: `odd_chars("abcde")`, expected: pyStr("bd")},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `odd_chars("")`, expected: pyStr("")},
			caseAlias{name: "single-char", call: `odd_chars("x")`, expected: pyStr("")},
			caseAlias{name: "two-chars", call: `odd_chars("hi")`, expected: pyStr("i")},
			caseAlias{name: "all-same", call: `odd_chars("aaaa")`, expected: pyStr("aa")},
			caseAlias{name: "drawn-even", call: fmt.Sprintf("odd_chars(%s)", pyStr(word+"z")), expected: pyStr(oddCharsGo(word+"z"))},
		)
		stmt := fmt.Sprintf("%s奇位汲取：抽取字符串中下标为奇数（0 起）的字符并按序拼接。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L3·字符串：数字能量求和（循环 + 数字判定，参考解 5 行）。
func skillDrawL3() question.Template {
	meta := question.Meta{ID: "skill-draw-L3-skill", Topic: question.TopicString,
		Difficulty: question.L3, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"empty-input", "no-digits"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		mix := rng.Str(rng.Int(3, 6), "digit")
		kind := rng.Scenario("spell")
		sig := "def digit_energy(s: str) -> int:"
		answer := "def digit_energy(s: str) -> int:\n    total = 0\n    for ch in s:\n        if ch.isdigit():\n            total += int(ch)\n    return total\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("digit_energy(%s)", pyStr(mix)), expected: fmt.Sprintf("%d", digitSumGo(mix))},
			caseAlias{name: "example-2", call: `digit_energy("a1b2")`, expected: "3"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `digit_energy("")`, expected: "0"},
			caseAlias{name: "no-digits", call: `digit_energy("abc")`, expected: "0"},
			caseAlias{name: "all-digits", call: `digit_energy("99")`, expected: "18"},
			caseAlias{name: "with-zero", call: `digit_energy("a0b")`, expected: "0"},
			caseAlias{name: "mixed", call: `digit_energy("x1y2z3")`, expected: "6"},
		)
		stmt := fmt.Sprintf("%s能量汲取：提取字符串中所有数字字符并求和（数字 0-9 各为其面值）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L4·字符串：压缩咒文还原（数字前缀解析 + 重复展开，参考解 9 行）。
// 语法刻意取「数字在字母前」：展开发生在遇到字母时，避免惰性冲刷类缺陷。
func skillDecompressL4() question.Template {
	meta := question.Meta{ID: "skill-decompress-L4-skill", Topic: question.TopicString,
		Difficulty: question.L4, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"empty-input", "multi-digit-count", "no-count"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		drawn := rng.Pick([]string{"2a1b2c", "10a", "3xy", "2b3a", "12z", "5q1w"})
		kind := rng.Scenario("spell")
		sig := "def decompress(s: str) -> str:"
		answer := "def decompress(s: str) -> str:\n" +
			"    result = ''\n" +
			"    count = 0\n" +
			"    for ch in s:\n" +
			"        if ch.isdigit():\n" +
			"            count = count * 10 + int(ch)\n" +
			"        else:\n" +
			"            if count == 0:\n" +
			"                count = 1\n" +
			"            result += ch * count\n" +
			"            count = 0\n" +
			"    return result\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("decompress(%s)", pyStr(drawn)), expected: pyStr(decompressGo(drawn))},
			caseAlias{name: "example-2", call: `decompress("10a")`, expected: pyStr("aaaaaaaaaa")},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `decompress("")`, expected: pyStr("")},
			caseAlias{name: "no-count", call: `decompress("ab")`, expected: pyStr("ab")},
			caseAlias{name: "multi-digit-count", call: `decompress("3xy")`, expected: pyStr("xxxy")},
			caseAlias{name: "drawn", call: `decompress("2b3a")`, expected: pyStr(decompressGo("2b3a"))},
			caseAlias{name: "single", call: `decompress("z")`, expected: pyStr("z")},
		)
		stmt := fmt.Sprintf("还原%s压缩咒文：形如 '2a1b2c' 展开为 'aabbcc'（字母前的数字为该字母重复次数，可多位；无数字视为 1 次）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L5·字符串：回文划分（回文表 + 线性 DP，参考解 18 行；
// 算法内核为回文切割 II 的最少切割数，题面经情境池重包装）。
func skillWardL5() question.Template {
	meta := question.Meta{ID: "skill-ward-L5-skill", Topic: question.TopicString,
		Difficulty: question.L5, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-char", "all-same", "whole-palindrome", "worst-case"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		drawn := rng.Pick([]string{"aab", "abab", "abcba", "aabb", "bbaa", "abcabc"})
		kind := rng.Scenario("spell")
		sig := "def min_cut(s: str) -> int:"
		answer := "def min_cut(s: str) -> int:\n" +
			"    n = len(s)\n" +
			"    if n == 0:\n" +
			"        return 0\n" +
			"    is_pal = [[False] * n for _ in range(n)]\n" +
			"    for i in range(n):\n" +
			"        is_pal[i][i] = True\n" +
			"    for length in range(2, n + 1):\n" +
			"        for i in range(n - length + 1):\n" +
			"            j = i + length - 1\n" +
			"            if s[i] == s[j] and (j - i < 2 or is_pal[i + 1][j - 1]):\n" +
			"                is_pal[i][j] = True\n" +
			"    cuts = list(range(n + 1))\n" +
			"    for i in range(1, n + 1):\n" +
			"        for j in range(i):\n" +
			"            if is_pal[j][i - 1]:\n" +
			"                if cuts[j] + 1 < cuts[i]:\n" +
			"                    cuts[i] = cuts[j] + 1\n" +
			"    return cuts[n] - 1\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("min_cut(%s)", pyStr(drawn)), expected: fmt.Sprintf("%d", minCutGo(drawn))},
			caseAlias{name: "example-2", call: `min_cut("racecar")`, expected: "0"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `min_cut("")`, expected: "0"},
			caseAlias{name: "single-char", call: `min_cut("a")`, expected: "0"},
			caseAlias{name: "all-same", call: `min_cut("aaaa")`, expected: "0"},
			caseAlias{name: "worst-case", call: `min_cut("abcde")`, expected: "4"},
			caseAlias{name: "two-cut", call: `min_cut("abab")`, expected: "1"},
			caseAlias{name: "whole-palindrome", call: `min_cut("abccba")`, expected: "0"},
			caseAlias{name: "mixed", call: `min_cut("bcacb")`, expected: fmt.Sprintf("%d", minCutGo("bcacb"))},
		)
		stmt := fmt.Sprintf("%s结界划分：把字符串划分成若干回文段，返回最少切割次数（整串回文为 0；空串为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// ---- 期望值计算辅助（与参考解同语义） ----

func maskTailGo(s string) string {
	n := len(s)
	if n <= 4 {
		return strings.Repeat("*", n)
	}
	return strings.Repeat("*", n-4) + s[n-4:]
}

func scrollPalindromeGo(s string) bool {
	cleaned := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			if ch >= 'A' && ch <= 'Z' {
				ch += 'a' - 'A'
			}
			cleaned = append(cleaned, ch)
		}
	}
	for i, j := 0, len(cleaned)-1; i < j; i, j = i+1, j-1 {
		if cleaned[i] != cleaned[j] {
			return false
		}
	}
	return true
}

func barrierSpanGo(s string) int {
	best := 0
	stack := []int{-1}
	for i := 0; i < len(s); i++ {
		if s[i] == '(' {
			stack = append(stack, i)
		} else {
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				stack = append(stack, i)
			} else if span := i - stack[len(stack)-1]; span > best {
				best = span
			}
		}
	}
	return best
}

func oddCharsGo(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if i%2 == 1 {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func digitSumGo(s string) int {
	total := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			total += int(s[i] - '0')
		}
	}
	return total
}

func decompressGo(s string) string {
	var b strings.Builder
	count := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch >= '0' && ch <= '9' {
			count = count*10 + int(ch-'0')
		} else {
			times := count
			if times == 0 {
				times = 1
			}
			b.WriteString(strings.Repeat(string(ch), times))
			count = 0
		}
	}
	return b.String()
}

func minCutGo(s string) int {
	n := len(s)
	if n == 0 {
		return 0
	}
	isPal := make([][]bool, n)
	for i := range isPal {
		isPal[i] = make([]bool, n)
		isPal[i][i] = true
	}
	for length := 2; length <= n; length++ {
		for i := 0; i+length-1 < n; i++ {
			j := i + length - 1
			if s[i] == s[j] && (j-i < 2 || isPal[i+1][j-1]) {
				isPal[i][j] = true
			}
		}
	}
	cuts := make([]int, n+1)
	for i := range cuts {
		cuts[i] = i
	}
	for i := 1; i <= n; i++ {
		for j := 0; j < i; j++ {
			if isPal[j][i-1] && cuts[j]+1 < cuts[i] {
				cuts[i] = cuts[j] + 1
			}
		}
	}
	return cuts[n] - 1
}
