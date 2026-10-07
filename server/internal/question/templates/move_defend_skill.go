// 私有行动题·移动 / 防御 / 技能（04 首批 8 + 12 号票扩容：形式均为 complete，
// 按「题型 × 难度 × 动作」矩阵格补齐；引擎本期只消费攻击题，私有题消费方是 03/06 号票）。
package templates

import (
	"fmt"

	"codewar/server/internal/question"
)

// ---- 移动题 ----

// 移动 L1：曼哈顿距离（参考解 3 行）。
func moveManhattanL1() question.Template {
	meta := question.Meta{ID: "move-manhattan-L1-move", Topic: question.TopicSyntax,
		Difficulty: question.L1, Action: question.ActionMove, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		x1, y1 := rng.Int(0, 4), rng.Int(0, 4)
		x2, y2 := rng.Int(0, 4), rng.Int(0, 4)
		sig := "def manhattan(x1: int, y1: int, x2: int, y2: int) -> int:"
		answer := "def manhattan(x1: int, y1: int, x2: int, y2: int) -> int:\n    dx = abs(x1 - x2)\n    dy = abs(y1 - y2)\n    return dx + dy\n"
		want := absGo(x1-x2) + absGo(y1-y2)
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("manhattan(%d, %d, %d, %d)", x1, y1, x2, y2), expected: fmt.Sprintf("%d", want)},
			caseAlias{name: "example-2", call: "manhattan(0, 0, 3, 4)", expected: "7"},
		)
		hidden := cases(
			caseAlias{name: "same-cell", call: "manhattan(2, 2, 2, 2)", expected: "0"},
			caseAlias{name: "adjacent", call: "manhattan(1, 1, 1, 2)", expected: "1"},
			caseAlias{name: "diagonal", call: "manhattan(0, 0, 4, 4)", expected: "8"},
			caseAlias{name: "reverse", call: "manhattan(4, 0, 0, 4)", expected: "8"},
			caseAlias{name: "row-walk", call: "manhattan(0, 3, 4, 3)", expected: "4"},
		)
		stmt := fmt.Sprintf("求两格之间的曼哈顿距离（正交步数）。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L2：朝目标走一步（优先 x 轴，参考解 5 行）。
func moveStepL2() question.Template {
	meta := question.Meta{ID: "move-step-L2-move", Topic: question.TopicArray,
		Difficulty: question.L2, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"already-there"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		px, py := rng.Int(0, 4), rng.Int(0, 4)
		tx, ty := rng.Int(0, 4), rng.Int(0, 4)
		sig := "def step_toward(px: int, py: int, tx: int, ty: int) -> list:"
		answer := "def step_toward(px: int, py: int, tx: int, ty: int) -> list:\n    dx = 1 if tx > px else (-1 if tx < px else 0)\n    dy = 1 if ty > py else (-1 if ty < py else 0)\n    if dx != 0:\n        return [px + dx, py]\n    return [px, py + dy]\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("step_toward(%d, %d, %d, %d)", px, py, tx, ty), expected: pyList(stepGo(px, py, tx, ty))},
			caseAlias{name: "example-2", call: "step_toward(1, 1, 3, 2)", expected: "[2, 1]"},
		)
		hidden := cases(
			caseAlias{name: "already-there", call: "step_toward(2, 2, 2, 2)", expected: "[2, 2]"},
			caseAlias{name: "x-first", call: "step_toward(0, 0, 2, 2)", expected: "[1, 0]"},
			caseAlias{name: "y-only", call: "step_toward(1, 1, 1, 3)", expected: "[1, 2]"},
			caseAlias{name: "negative-dir", call: "step_toward(3, 3, 1, 1)", expected: "[2, 3]"},
			caseAlias{name: "adjacent-x", call: "step_toward(0, 4, 1, 4)", expected: "[1, 4]"},
		)
		stmt := fmt.Sprintf("向目标走一格（正交；x 轴优先；已在目标则原地）。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L3：指令串终点（字典映射，参考解 8 行）。
func moveFollowL3() question.Template {
	meta := question.Meta{ID: "move-follow-L3-move", Topic: question.TopicString,
		Difficulty: question.L3, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "invalid-char"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		moves := rng.Str(rng.Int(2, 5), "moves")
		if moves == "" {
			moves = "UR"
		}
		sig := "def follow_moves(moves: str) -> list:"
		answer := "def follow_moves(moves: str) -> list:\n    dx = {'U': 0, 'D': 0, 'L': -1, 'R': 1}\n    dy = {'U': -1, 'D': 1, 'L': 0, 'R': 0}\n    x = 0\n    y = 0\n    for m in moves:\n        x += dx.get(m, 0)\n        y += dy.get(m, 0)\n    return [x, y]\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("follow_moves(%s)", pyStr(moves)), expected: pyList(followGo(moves))},
			caseAlias{name: "example-2", call: `follow_moves("UUR")`, expected: "[1, -2]"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `follow_moves("")`, expected: "[0, 0]"},
			caseAlias{name: "invalid-char", call: `follow_moves("UXR")`, expected: "[1, -1]"},
			caseAlias{name: "loop-back", call: `follow_moves("UDLR")`, expected: "[0, 0]"},
			caseAlias{name: "down-only", call: `follow_moves("DDD")`, expected: "[0, 3]"},
			caseAlias{name: "left-only", call: `follow_moves("LL")`, expected: "[-2, 0]"},
		)
		stmt := fmt.Sprintf("从原点按 U/D/L/R 指令串行走（U y-1；未知字符忽略），返回终点 [x, y]。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// ---- 防御题 ----

// 防御 L1：减伤承伤（参考解 3 行）。
func defendReduceL1() question.Template {
	meta := question.Meta{ID: "defend-reduce-L1-defend", Topic: question.TopicSyntax,
		Difficulty: question.L1, Action: question.ActionDefend, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		raw := rng.Int(5, 40)
		sig := "def reduced_damage(raw: int) -> int:"
		answer := "def reduced_damage(raw: int) -> int:\n    blocked = raw * 3 // 5\n    taken = raw - blocked\n    return taken\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("reduced_damage(%d)", raw), expected: fmt.Sprintf("%d", raw-raw*3/5)},
			caseAlias{name: "example-2", call: "reduced_damage(22)", expected: "9"},
		)
		hidden := cases(
			caseAlias{name: "zero", call: "reduced_damage(0)", expected: "0"},
			caseAlias{name: "small", call: "reduced_damage(5)", expected: "2"},
			caseAlias{name: "odd", call: "reduced_damage(7)", expected: "3"},
			caseAlias{name: "large", call: "reduced_damage(100)", expected: "40"},
			caseAlias{name: "one", call: "reduced_damage(1)", expected: "1"},
		)
		stmt := fmt.Sprintf("防御减伤 60%%：返回实际承伤（向下取整）。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L2：护盾余量（参考解 5 行）。
func defendShieldL2() question.Template {
	meta := question.Meta{ID: "defend-shield-L2-defend", Topic: question.TopicArray,
		Difficulty: question.L2, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "overkill"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		cap := rng.Int(10, 30)
		n := rng.Int(1, 3)
		hits := rng.IntList(n, 1, 12, question.ListOpts{})
		sig := "def shield_left(capacity: int, hits: list) -> int:"
		answer := "def shield_left(capacity: int, hits: list) -> int:\n    total = 0\n    for h in hits:\n        total += h\n    left = capacity - total\n    return left if left > 0 else 0\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("shield_left(%d, %s)", cap, pyList(hits)), expected: fmt.Sprintf("%d", shieldGo(cap, hits))},
			caseAlias{name: "example-2", call: "shield_left(20, [5, 6])", expected: "9"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "shield_left(15, [])", expected: "15"},
			caseAlias{name: "overkill", call: "shield_left(10, [6, 8])", expected: "0"},
			caseAlias{name: "exact", call: "shield_left(10, [4, 6])", expected: "0"},
			caseAlias{name: "single-small", call: "shield_left(10, [3])", expected: "7"},
			caseAlias{name: "multi", call: "shield_left(30, [5, 5, 5])", expected: "15"},
		)
		stmt := fmt.Sprintf("护盾容量减去累计命中（余量下限 0）。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L3：能否存活（逐击减伤，参考解 8 行）。
func defendSurviveL3() question.Template {
	meta := question.Meta{ID: "defend-survive-L3-defend", Topic: question.TopicHash,
		Difficulty: question.L3, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "lethal", "full-block"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		hp := rng.Int(10, 30)
		n := rng.Int(1, 3)
		hits := rng.IntList(n, 1, 15, question.ListOpts{})
		red := rng.Int(0, 8)
		sig := "def will_survive(hp: int, hits: list, reduction: int) -> bool:"
		answer := "def will_survive(hp: int, hits: list, reduction: int) -> bool:\n    current = hp\n    for h in hits:\n        taken = h - reduction\n        taken = taken if taken > 0 else 0\n        current -= taken\n    if current <= 0:\n        return False\n    return True\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("will_survive(%d, %s, %d)", hp, pyList(hits), red), expected: pyBool(surviveGo(hp, hits, red))},
			caseAlias{name: "example-2", call: "will_survive(20, [8, 8], 3)", expected: "True"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "will_survive(10, [], 3)", expected: "True"},
			caseAlias{name: "lethal", call: "will_survive(10, [8, 8], 0)", expected: "False"},
			caseAlias{name: "full-block", call: "will_survive(10, [5, 5], 10)", expected: "True"},
			caseAlias{name: "exact-zero", call: "will_survive(10, [10], 0)", expected: "False"},
			caseAlias{name: "chip-survive", call: "will_survive(10, [6, 6], 3)", expected: "True"},
		)
		stmt := fmt.Sprintf("每击先减免 reduction（下限 0），结算后 HP>0 则存活。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// ---- 技能题（06 号票消费；L1/L2 为 12 号票批 1 补缺口——cap<L3 时技能位不再空缺） ----

// 技能 L3：蓄力爆伤（上限 60，参考解 7 行）。
func skillBlastL3() question.Template {
	meta := question.Meta{ID: "skill-blast-L3-skill", Topic: question.TopicSyntax,
		Difficulty: question.L3, Action: question.ActionSkill, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		base, charge := rng.Int(10, 40), rng.Int(0, 20)
		sig := "def blast_damage(base: int, charge: int) -> int:"
		answer := "def blast_damage(base: int, charge: int) -> int:\n    total = base + charge * 2\n    if total > 60:\n        total = 60\n    if total < 0:\n        total = 0\n    return total\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("blast_damage(%d, %d)", base, charge), expected: fmt.Sprintf("%d", blastGo(base, charge))},
			caseAlias{name: "example-2", call: "blast_damage(42, 5)", expected: "52"},
		)
		hidden := cases(
			caseAlias{name: "overcap", call: "blast_damage(50, 20)", expected: "60"},
			caseAlias{name: "no-charge", call: "blast_damage(30, 0)", expected: "30"},
			caseAlias{name: "exact-cap", call: "blast_damage(40, 10)", expected: "60"},
			caseAlias{name: "negative", call: "blast_damage(-10, 0)", expected: "0"},
			caseAlias{name: "small", call: "blast_damage(5, 3)", expected: "11"},
		)
		stmt := fmt.Sprintf("技能伤害 = base + charge×2（上限 60，下限 0）。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L3：治疗量（不溢出，参考解 8 行）。
func skillHealL3() question.Template {
	meta := question.Meta{ID: "skill-heal-L3-skill", Topic: question.TopicHash,
		Difficulty: question.L3, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"no-missing", "over-heal"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		missing, power := rng.Int(0, 40), rng.Int(5, 35)
		sig := "def heal_amount(missing: int, power: int) -> int:"
		answer := "def heal_amount(missing: int, power: int) -> int:\n    if power < 0:\n        power = 0\n    if missing < 0:\n        missing = 0\n    healed = missing if missing < power else power\n    if healed == 0:\n        return 0\n    return healed\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("heal_amount(%d, %d)", missing, power), expected: fmt.Sprintf("%d", healGo(missing, power))},
			caseAlias{name: "example-2", call: "heal_amount(30, 20)", expected: "20"},
		)
		hidden := cases(
			caseAlias{name: "no-missing", call: "heal_amount(0, 20)", expected: "0"},
			caseAlias{name: "over-heal", call: "heal_amount(10, 30)", expected: "10"},
			caseAlias{name: "exact", call: "heal_amount(25, 25)", expected: "25"},
			caseAlias{name: "zero-power", call: "heal_amount(15, 0)", expected: "0"},
			caseAlias{name: "small", call: "heal_amount(8, 5)", expected: "5"},
		)
		stmt := fmt.Sprintf("治疗量 = min(已损, 强度)，不溢出（全满/零强度返回 0）。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L1：蓄能截断（单语法点，参考解 3 行；12 号票批 1）。
func skillClampL1() question.Template {
	meta := question.Meta{ID: "skill-clamp-L1-skill", Topic: question.TopicSyntax,
		Difficulty: question.L1, Action: question.ActionSkill, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		val, limit := rng.Int(-5, 40), rng.Int(10, 30)
		kind := rng.Scenario("spell")
		sig := "def clamp_charge(value: int, cap: int) -> int:"
		answer := "def clamp_charge(value: int, cap: int) -> int:\n    if value > cap:\n        return cap\n    return value\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("clamp_charge(%d, %d)", val, limit), expected: fmt.Sprintf("%d", clampGo(val, limit))},
			caseAlias{name: "example-2", call: "clamp_charge(30, 20)", expected: "20"},
		)
		hidden := cases(
			caseAlias{name: "over-cap", call: "clamp_charge(25, 20)", expected: "20"},
			caseAlias{name: "exact-cap", call: "clamp_charge(20, 20)", expected: "20"},
			caseAlias{name: "below-cap", call: "clamp_charge(12, 20)", expected: "12"},
			caseAlias{name: "zero", call: "clamp_charge(0, 20)", expected: "0"},
			caseAlias{name: "negative", call: "clamp_charge(-3, 20)", expected: "-3"},
			caseAlias{name: "zero-cap", call: "clamp_charge(7, 0)", expected: "0"},
		)
		stmt := fmt.Sprintf("%s蓄能截断：输入值不超过上限 cap 时原样返回，超过则返回 cap。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L2：连锁治疗（循环 + min 截断，参考解 4 行；12 号票批 1）。
func skillChainHealL2() question.Template {
	meta := question.Meta{ID: "skill-chain-heal-L2-skill", Topic: question.TopicArray,
		Difficulty: question.L2, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"empty-input"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		power := rng.Int(3, 12)
		missing := rng.IntList(rng.Int(2, 5), 1, 15, question.ListOpts{})
		kind := rng.Scenario("heal")
		sig := "def chain_heal(power: int, missing: list) -> int:"
		answer := "def chain_heal(power: int, missing: list) -> int:\n    total = 0\n    for m in missing:\n        total += m if m < power else power\n    return total\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("chain_heal(%d, %s)", power, pyList(missing)), expected: fmt.Sprintf("%d", chainHealGo(power, missing))},
			caseAlias{name: "example-2", call: "chain_heal(5, [3, 8, 2])", expected: "10"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "chain_heal(6, [])", expected: "0"},
			caseAlias{name: "all-capped", call: "chain_heal(5, [9, 12])", expected: "10"},
			caseAlias{name: "none-capped", call: "chain_heal(9, [2, 4])", expected: "6"},
			caseAlias{name: "mixed", call: "chain_heal(7, [4, 10, 2])", expected: "13"},
			caseAlias{name: "single", call: "chain_heal(10, [6])", expected: "6"},
		)
		stmt := fmt.Sprintf("%s连锁：对每位队友回复 min(power, 其已损生命)，返回总治疗量。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// ---- 语法主题私有题（12 号票批 2：该题型 × 难度 × 动作矩阵格补齐） ----

// 移动 L2·语法：方向向量（查表 + get 兜底，参考解 3 行）。
func moveVectorL2() question.Template {
	meta := question.Meta{ID: "move-vector-L2-move", Topic: question.TopicSyntax,
		Difficulty: question.L2, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"unknown-char"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		ch := rng.Pick([]string{"U", "D", "L", "R"})
		kind := rng.Scenario("rune")
		sig := "def direction_vector(ch: str) -> list:"
		answer := "def direction_vector(ch: str) -> list:\n    dx = {'U': 0, 'D': 0, 'L': -1, 'R': 1}\n    dy = {'U': -1, 'D': 1, 'L': 0, 'R': 0}\n    return [dx.get(ch, 0), dy.get(ch, 0)]\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("direction_vector(%s)", pyStr(ch)), expected: pyList(vectorGo(ch[0]))},
			caseAlias{name: "example-2", call: `direction_vector("U")`, expected: "[0, -1]"},
		)
		hidden := cases(
			caseAlias{name: "unknown-char", call: `direction_vector("X")`, expected: "[0, 0]"},
			caseAlias{name: "down", call: `direction_vector("D")`, expected: "[0, 1]"},
			caseAlias{name: "left", call: `direction_vector("L")`, expected: "[-1, 0]"},
			caseAlias{name: "right", call: `direction_vector("R")`, expected: "[1, 0]"},
			caseAlias{name: "empty", call: `direction_vector("")`, expected: "[0, 0]"},
		)
		stmt := fmt.Sprintf("%s号令翻译：单个方向字符 U/D/L/R 转为位移向量 [dx, dy]（U 为 y−1、R 为 x+1；未知字符返回 [0, 0]）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L3·语法：环形回廊行走（循环 + 取模环绕，参考解 5 行；Python 取模恒非负为考点）。
func moveCorridorL3() question.Template {
	meta := question.Meta{ID: "move-corridor-L3-move", Topic: question.TopicSyntax,
		Difficulty: question.L3, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "negative-steps", "full-loop", "wrap-around"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		length := rng.Int(5, 9)
		start := rng.Int(0, length-1)
		steps := rng.IntList(rng.Int(1, 4), -6, 9, question.ListOpts{})
		sig := "def corridor_end(start: int, steps: list, length: int) -> int:"
		answer := "def corridor_end(start: int, steps: list, length: int) -> int:\n    total = 0\n    for s in steps:\n        total += s\n    pos = (start + total) % length\n    return pos\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("corridor_end(%d, %s, %d)", start, pyList(steps), length), expected: fmt.Sprintf("%d", corridorGo(start, steps, length))},
			caseAlias{name: "example-2", call: "corridor_end(4, [2], 5)", expected: "1"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "corridor_end(3, [], 5)", expected: "3"},
			caseAlias{name: "negative-steps", call: "corridor_end(0, [-1], 5)", expected: "4"},
			caseAlias{name: "full-loop", call: "corridor_end(0, [7], 7)", expected: "0"},
			caseAlias{name: "wrap-around", call: "corridor_end(4, [4], 5)", expected: "3"},
			caseAlias{name: "multi-wrap", call: "corridor_end(2, [3, 9, -4], 7)", expected: "3"},
			caseAlias{name: "back-to-start", call: "corridor_end(1, [-1, 1], 6)", expected: "1"},
		)
		stmt := fmt.Sprintf("环形回廊行军：回廊编号 0..length−1，从 start 出发依次走 steps（负数即后退），返回最终编号。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L4·语法：行军令解压（多位数字解析 + 展开，参考解 10 行）。
func moveParseL4() question.Template {
	meta := question.Meta{ID: "move-parse-L4-move", Topic: question.TopicSyntax,
		Difficulty: question.L4, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "no-number", "multi-digit"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		compressed := rng.Pick([]string{"2R3L", "1U2D", "3L1R", "2R1L3U"})
		kind := rng.Scenario("rune")
		sig := "def parse_moves(compressed: str) -> list:"
		answer := "def parse_moves(compressed: str) -> list:\n    moves = []\n    count = 0\n    for ch in compressed:\n        if ch.isdigit():\n            count = count * 10 + int(ch)\n        else:\n            times = max(count, 1)\n            moves.extend([ch] * times)\n            count = 0\n    return moves\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("parse_moves(%s)", pyStr(compressed)), expected: pyStrList(parseMovesGo(compressed))},
			caseAlias{name: "example-2", call: `parse_moves("2R3L")`, expected: pyStrList([]string{"R", "R", "L", "L", "L"})},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `parse_moves("")`, expected: pyStrList([]string{})},
			caseAlias{name: "no-number", call: `parse_moves("RL")`, expected: pyStrList([]string{"R", "L"})},
			caseAlias{name: "multi-digit", call: `parse_moves("12R3L")`, expected: pyStrList(append(repeatStr("R", 12), "L", "L", "L"))},
			caseAlias{name: "single", call: `parse_moves("U")`, expected: pyStrList([]string{"U"})},
			caseAlias{name: "mixed", call: `parse_moves("3DU")`, expected: pyStrList([]string{"D", "D", "D", "U"})},
		)
		stmt := fmt.Sprintf("解读%s行军令：形如 '2R3L' 的指令串展开为 ['R','R','L','L','L']（字母前的数字为该字母的重复次数，无数字则重复 1 次；次数可多位；约定指令串以字母结尾），返回指令列表。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L2·语法：战地抢修（循环 + 上限截断，参考解 4 行）。
func defendRepairL2() question.Template {
	meta := question.Meta{ID: "defend-repair-L2-defend", Topic: question.TopicSyntax,
		Difficulty: question.L2, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"over-cap", "empty-input"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		current := rng.Int(0, 20)
		repairs := rng.IntList(rng.Int(1, 4), 3, 15, question.ListOpts{})
		shieldCap := rng.Int(25, 40)
		sig := "def repaired_shield(current: int, repairs: list, cap: int) -> int:"
		answer := "def repaired_shield(current: int, repairs: list, cap: int) -> int:\n    shield = current\n    for r in repairs:\n        shield = min(shield + r, cap)\n    return shield\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("repaired_shield(%d, %s, %d)", current, pyList(repairs), shieldCap), expected: fmt.Sprintf("%d", repairedGo(current, repairs, shieldCap))},
			caseAlias{name: "example-2", call: "repaired_shield(10, [20], 30)", expected: "30"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "repaired_shield(10, [], 30)", expected: "10"},
			caseAlias{name: "over-cap", call: "repaired_shield(10, [50], 30)", expected: "30"},
			caseAlias{name: "exact-cap", call: "repaired_shield(10, [20], 30)", expected: "30"},
			caseAlias{name: "accumulate", call: "repaired_shield(5, [10, 10, 10], 25)", expected: "25"},
			caseAlias{name: "at-cap", call: "repaired_shield(30, [5], 30)", expected: "30"},
		)
		stmt := fmt.Sprintf("战地抢修：护盾自 current 起，依次吸收 repairs 中每笔修复（不超过 cap 即截断），返回最终护盾值。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L3·语法：攻防结算（护盾优先吸收 + 溢出扣血，参考解 7 行）。
func defendHpAfterL3() question.Template {
	meta := question.Meta{ID: "defend-hp-after-L3-defend", Topic: question.TopicSyntax,
		Difficulty: question.L3, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "shield-eats-all", "overkill", "exact-zero"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		hp := rng.Int(15, 30)
		shield := rng.Int(0, 8)
		hits := rng.IntList(rng.Int(1, 4), 1, 12, question.ListOpts{})
		sig := "def hp_after(hp: int, shield: int, hits: list) -> int:"
		answer := "def hp_after(hp: int, shield: int, hits: list) -> int:\n    for h in hits:\n        absorbed = min(h, shield)\n        shield -= absorbed\n        hp -= h - absorbed\n    if hp < 0:\n        hp = 0\n    return hp\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("hp_after(%d, %d, %s)", hp, shield, pyList(hits)), expected: fmt.Sprintf("%d", hpAfterGo(hp, shield, hits))},
			caseAlias{name: "example-2", call: "hp_after(10, 3, [5, 9])", expected: "0"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "hp_after(20, 5, [])", expected: "20"},
			caseAlias{name: "shield-eats-all", call: "hp_after(20, 5, [3])", expected: "20"},
			caseAlias{name: "overkill", call: "hp_after(10, 0, [12])", expected: "0"},
			caseAlias{name: "exact-zero", call: "hp_after(10, 0, [10])", expected: "0"},
			caseAlias{name: "partial-absorb", call: "hp_after(20, 3, [4, 4])", expected: "15"},
			caseAlias{name: "shield-depletes", call: "hp_after(15, 6, [4, 4, 4])", expected: "9"},
		)
		stmt := fmt.Sprintf("攻防结算：每击先由护盾等额吸收（护盾不回复），溢出部分扣血；血量下限 0，返回战后血量。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L4·语法：护甲消耗战（减伤 + 逐击衰减 + 阵亡短路，参考解 9 行）。
func defendSiegeL4() question.Template {
	meta := question.Meta{ID: "defend-siege-L4-defend", Topic: question.TopicSyntax,
		Difficulty: question.L4, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "full-block", "degrade-chain", "lethal"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		hp := rng.Int(20, 40)
		armor := rng.Int(2, 10)
		hits := rng.IntList(rng.Int(2, 4), 3, 15, question.ListOpts{})
		sig := "def siege_endure(hp: int, armor: int, hits: list) -> int:"
		answer := "def siege_endure(hp: int, armor: int, hits: list) -> int:\n    for h in hits:\n        taken = h - armor\n        if taken < 0:\n            taken = 0\n        hp -= taken\n        armor = armor // 2\n        if hp <= 0:\n            return 0\n    return hp\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("siege_endure(%d, %d, %s)", hp, armor, pyList(hits)), expected: fmt.Sprintf("%d", siegeGo(hp, armor, hits))},
			caseAlias{name: "example-2", call: "siege_endure(30, 8, [10, 9, 8])", expected: "17"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "siege_endure(30, 8, [])", expected: "30"},
			caseAlias{name: "full-block", call: "siege_endure(30, 8, [5])", expected: "30"},
			caseAlias{name: "lethal", call: "siege_endure(10, 0, [12])", expected: "0"},
			caseAlias{name: "exact-lethal", call: "siege_endure(10, 0, [10])", expected: "0"},
			caseAlias{name: "degrade-chain", call: "siege_endure(20, 3, [4, 4])", expected: "16"},
			caseAlias{name: "mid-death", call: "siege_endure(9, 2, [8, 8])", expected: "0"},
		)
		stmt := fmt.Sprintf("护甲消耗战：每次受击先抵扣当前护甲（承伤 = max(0, 伤害 − 护甲)），受击后护甲减半（向下取整）；血量归零立即阵亡（返回 0），否则返回战后血量。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L2·语法：奥术涌动（算术 + 上限截断，参考解 4 行）。
func skillManaL2() question.Template {
	meta := question.Meta{ID: "skill-mana-L2-skill", Topic: question.TopicSyntax,
		Difficulty: question.L2, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"over-cap"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		start := rng.Int(0, 30)
		casts := rng.Int(0, 6)
		manaCap := rng.Int(35, 60)
		sig := "def mana_after(start: int, casts: int, cap: int) -> int:"
		answer := "def mana_after(start: int, casts: int, cap: int) -> int:\n    mana = start + casts * 3\n    if mana > cap:\n        mana = cap\n    return mana\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("mana_after(%d, %d, %d)", start, casts, manaCap), expected: fmt.Sprintf("%d", manaAfterGo(start, casts, manaCap))},
			caseAlias{name: "example-2", call: "mana_after(10, 5, 50)", expected: "25"},
		)
		hidden := cases(
			caseAlias{name: "no-cast", call: "mana_after(10, 0, 50)", expected: "10"},
			caseAlias{name: "over-cap", call: "mana_after(40, 5, 50)", expected: "50"},
			caseAlias{name: "exact-cap", call: "mana_after(0, 2, 6)", expected: "6"},
			caseAlias{name: "single-cast", call: "mana_after(5, 1, 30)", expected: "8"},
			caseAlias{name: "small-cap", call: "mana_after(4, 4, 10)", expected: "10"},
		)
		stmt := fmt.Sprintf("奥术涌动：每次施法回复 3 点法力，casts 次施法后法力不超过 cap（超出截断），返回最终法力。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L4·语法：连锁奥术（衰减链 + 下限钳制 + 总量截断，参考解 10 行）。
func skillArcanaL4() question.Template {
	meta := question.Meta{ID: "skill-arcana-L4-skill", Topic: question.TopicSyntax,
		Difficulty: question.L4, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"zero-targets", "decay-floor", "overcap"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		base := rng.Int(5, 40)
		targets := rng.Int(1, 6)
		sig := "def chain_arcana(base: int, targets: int) -> int:"
		answer := "def chain_arcana(base: int, targets: int) -> int:\n    total = 0\n    damage = base\n    for _ in range(targets):\n        total += damage\n        damage = damage // 2\n        if damage == 0:\n            damage = 1\n    if total > 99:\n        total = 99\n    return total\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("chain_arcana(%d, %d)", base, targets), expected: fmt.Sprintf("%d", chainArcanaGo(base, targets))},
			caseAlias{name: "example-2", call: "chain_arcana(16, 5)", expected: "31"},
		)
		hidden := cases(
			caseAlias{name: "zero-targets", call: "chain_arcana(8, 0)", expected: "0"},
			caseAlias{name: "single-target", call: "chain_arcana(8, 1)", expected: "8"},
			caseAlias{name: "decay-floor", call: "chain_arcana(64, 8)", expected: "99"},
			caseAlias{name: "overcap", call: "chain_arcana(50, 30)", expected: "99"},
			caseAlias{name: "two-targets", call: "chain_arcana(50, 2)", expected: "75"},
			caseAlias{name: "small-base", call: "chain_arcana(1, 3)", expected: "3"},
		)
		stmt := fmt.Sprintf("连锁奥术：首发伤害 = base，其后每个目标伤害减半（向下取整，至少 1）；总伤上限 99（超出截断），返回总伤害（targets=0 时为 0）。\n%s", examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// ---- 小型纯函数（期望值计算，与参考解同语义） ----

func absGo(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func stepGo(px, py, tx, ty int) []int {
	dx, dy := 0, 0
	if tx > px {
		dx = 1
	} else if tx < px {
		dx = -1
	}
	if ty > py {
		dy = 1
	} else if ty < py {
		dy = -1
	}
	if dx != 0 {
		return []int{px + dx, py}
	}
	return []int{px, py + dy}
}

func followGo(moves string) []int {
	dx := map[byte]int{'U': 0, 'D': 0, 'L': -1, 'R': 1}
	dy := map[byte]int{'U': -1, 'D': 1, 'L': 0, 'R': 0}
	x, y := 0, 0
	for i := 0; i < len(moves); i++ {
		x += dx[moves[i]]
		y += dy[moves[i]]
	}
	return []int{x, y}
}

func shieldGo(cap int, hits []int) int {
	t := 0
	for _, h := range hits {
		t += h
	}
	left := cap - t
	if left < 0 {
		return 0
	}
	return left
}

func surviveGo(hp int, hits []int, red int) bool {
	cur := hp
	for _, h := range hits {
		taken := h - red
		if taken < 0 {
			taken = 0
		}
		cur -= taken
	}
	return cur > 0
}

func blastGo(base, charge int) int {
	t := base + charge*2
	if t > 60 {
		t = 60
	}
	if t < 0 {
		t = 0
	}
	return t
}

func healGo(missing, power int) int {
	if power < 0 {
		power = 0
	}
	if missing < 0 {
		missing = 0
	}
	if missing < power {
		return missing
	}
	return power
}

func clampGo(v, limit int) int {
	if v > limit {
		return limit
	}
	return v
}

func chainHealGo(power int, missing []int) int {
	total := 0
	for _, m := range missing {
		if m < power {
			total += m
		} else {
			total += power
		}
	}
	return total
}

// vectorGo：期望值计算（与参考解同语义：未知字符 [0,0]）。
func vectorGo(ch byte) []int {
	dx := map[byte]int{'U': 0, 'D': 0, 'L': -1, 'R': 1}
	dy := map[byte]int{'U': -1, 'D': 1, 'L': 0, 'R': 0}
	return []int{dx[ch], dy[ch]}
}

// corridorGo：期望值计算（Python 取模恒非负语义，Go 需两次取模矫正）。
func corridorGo(start int, steps []int, length int) int {
	total := 0
	for _, s := range steps {
		total += s
	}
	return ((start+total)%length + length) % length
}

// parseMovesGo：期望值计算（与参考解同语义：无数字前缀重复 1 次）。
func parseMovesGo(s string) []string {
	var moves []string
	count := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch >= '0' && ch <= '9' {
			count = count*10 + int(ch-'0')
		} else {
			times := count
			if times <= 0 {
				times = 1
			}
			for k := 0; k < times; k++ {
				moves = append(moves, string(ch))
			}
			count = 0
		}
	}
	return moves
}

// repeatStr：字符串重复（用例期望值构造）。
func repeatStr(s string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = s
	}
	return out
}

// repairedGo：期望值计算（与参考解同语义：逐笔回复、上限截断）。
func repairedGo(current int, repairs []int, cap int) int {
	shield := current
	for _, r := range repairs {
		shield += r
		if shield > cap {
			shield = cap
		}
	}
	return shield
}

// hpAfterGo：期望值计算（与参考解同语义：护盾优先吸收、溢出扣血、下限 0）。
func hpAfterGo(hp, shield int, hits []int) int {
	for _, h := range hits {
		absorbed := h
		if shield < absorbed {
			absorbed = shield
		}
		shield -= absorbed
		hp -= h - absorbed
	}
	if hp < 0 {
		hp = 0
	}
	return hp
}

// siegeGo：期望值计算（与参考解同语义：减伤 + 护甲减半 + 阵亡短路）。
func siegeGo(hp, armor int, hits []int) int {
	for _, h := range hits {
		taken := h - armor
		if taken < 0 {
			taken = 0
		}
		hp -= taken
		armor /= 2
		if hp <= 0 {
			return 0
		}
	}
	return hp
}

// manaAfterGo：期望值计算（与参考解同语义）。
func manaAfterGo(start, casts, cap int) int {
	mana := start + casts*3
	if mana > cap {
		mana = cap
	}
	return mana
}

// chainArcanaGo：期望值计算（与参考解同语义：减半衰减、下限 1、总量截断 99）。
func chainArcanaGo(base, targets int) int {
	total := 0
	damage := base
	for i := 0; i < targets; i++ {
		total += damage
		damage /= 2
		if damage == 0 {
			damage = 1
		}
	}
	if total > 99 {
		total = 99
	}
	return total
}
