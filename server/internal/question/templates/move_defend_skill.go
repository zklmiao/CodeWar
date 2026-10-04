// 私有行动题·移动 / 防御 / 技能（8 模板：形式均为 complete；
// 引擎本期只消费攻击题，移动/防御的消费方是 03/06 号票——本票只在工厂层实现并测试覆盖）。
package templates

import (
	"fmt"

	"codewar/server/internal/question"
)

// ---- 移动题 ----

// 移动 L1：曼哈顿距离（参考解 3 行）。
func moveManhattanL1() question.Template {
	meta := question.Meta{ID: "move-manhattan-L1-move", Topic: question.TopicSyntax,
		Difficulty: question.L1, Action: question.ActionMove, Form: question.FormComplete, Golden: true}
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
		Traps: []string{"already-there"}, Golden: false}
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
		Traps: []string{"empty-input", "invalid-char"}, Golden: false}
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
		Difficulty: question.L1, Action: question.ActionDefend, Form: question.FormComplete, Golden: true}
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
		Traps: []string{"empty-input", "overkill"}, Golden: false}
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
		Traps: []string{"empty-input", "lethal", "full-block"}, Golden: false}
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

// ---- 技能题（供 06 号票参照，本期引擎不消费） ----

// 技能 L3：蓄力爆伤（上限 60，参考解 7 行）。
func skillBlastL3() question.Template {
	meta := question.Meta{ID: "skill-blast-L3-skill", Topic: question.TopicSyntax,
		Difficulty: question.L3, Action: question.ActionSkill, Form: question.FormComplete, Golden: false}
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
		Traps: []string{"no-missing", "over-heal"}, Golden: false}
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
