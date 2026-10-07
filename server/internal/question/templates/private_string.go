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
