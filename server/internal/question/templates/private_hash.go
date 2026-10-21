// 私有行动题·哈希与计数主题（12 号票批 5：move 5 / defend 4 / skill 4，覆盖 L1-L5 缺口）。
// 形式均为 complete；题型标注进入矩阵格（coverage_test.go 口径）。
package templates

import (
	"fmt"
	"sort"
	"strings"

	"codewar/server/internal/question"
)

// 移动 L1·哈希：查表步进（字典字面量 + get 兜底，参考解 2 行）。
func moveLookupL1() question.Template {
	meta := question.Meta{ID: "move-lookup-L1-move", Topic: question.TopicHash,
		Difficulty: question.L1, Action: question.ActionMove, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		cmd := rng.Pick([]string{"F", "B", "X", "U"})
		kind := rng.Scenario("arena")
		sig := "def step_for(cmd: str) -> int:"
		answer := "def step_for(cmd: str) -> int:\n    table = {'F': 1, 'B': -1}\n    return table.get(cmd, 0)\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("step_for(%s)", pyStr(cmd)), expected: fmt.Sprintf("%d", stepForGo(cmd))},
			caseAlias{name: "example-2", call: `step_for("F")`, expected: "1"},
		)
		hidden := cases(
			caseAlias{name: "back", call: `step_for("B")`, expected: "-1"},
			caseAlias{name: "unknown", call: `step_for("X")`, expected: "0"},
			caseAlias{name: "empty-input", call: `step_for("")`, expected: "0"},
			caseAlias{name: "multi-char", call: `step_for("FF")`, expected: "0"},
			caseAlias{name: "lower-case", call: `step_for("f")`, expected: "0"},
		)
		stmt := fmt.Sprintf("%s查表步进：F 为前进一格（+1）、B 为后退一格（−1），其余指令为 0。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L2·哈希：指令计数（字典计数 + get 兜底，参考解 5 行）。
func moveKeyCountL2() question.Template {
	meta := question.Meta{ID: "move-keycount-L2-move", Topic: question.TopicHash,
		Difficulty: question.L2, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "absent"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		moves := rng.PickN([]string{"U", "D", "L", "R", "F"}, rng.Int(3, 6))
		cmd := rng.Pick([]string{"U", "D", "L", "R", "F"})
		kind := rng.Scenario("arena")
		sig := "def count_cmd(moves: list, cmd: str) -> int:"
		answer := "def count_cmd(moves: list, cmd: str) -> int:\n    seen = {}\n    for m in moves:\n        seen[m] = seen.get(m, 0) + 1\n    return seen.get(cmd, 0)\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("count_cmd(%s, %s)", pyStrList(moves), pyStr(cmd)), expected: fmt.Sprintf("%d", countCmdGo(moves, cmd))},
			caseAlias{name: "example-2", call: `count_cmd(["F", "F"], "F")`, expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `count_cmd([], "F")`, expected: "0"},
			caseAlias{name: "absent", call: `count_cmd(["F"], "B")`, expected: "0"},
			caseAlias{name: "all-match", call: `count_cmd(["U", "U", "U"], "U")`, expected: "3"},
			caseAlias{name: "mixed", call: `count_cmd(["U", "D", "U"], "U")`, expected: "2"},
			caseAlias{name: "single", call: `count_cmd(["X"], "X")`, expected: "1"},
		)
		stmt := fmt.Sprintf("%s指令计数：统计指定指令在指令列表中出现的次数（空列表/不存在为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L3·哈希：首个重复指令（集合去重 + 下标返回，参考解 7 行）。
func moveFirstRepeatL3() question.Template {
	meta := question.Meta{ID: "move-first-repeat-L3-move", Topic: question.TopicHash,
		Difficulty: question.L3, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "no-repeat", "repeat-adjacent"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		moves := rng.PickN([]string{"U", "D", "L", "R"}, rng.Int(3, 6))
		kind := rng.Scenario("arena")
		sig := "def first_repeat(moves: list) -> int:"
		answer := "def first_repeat(moves: list) -> int:\n    seen = set()\n    for i in range(len(moves)):\n        if moves[i] in seen:\n            return i\n        seen.add(moves[i])\n    return -1\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("first_repeat(%s)", pyStrList(moves)), expected: fmt.Sprintf("%d", firstRepeatGo(moves))},
			caseAlias{name: "example-2", call: `first_repeat(["R", "L", "R"])`, expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "first_repeat([])", expected: "-1"},
			caseAlias{name: "no-repeat", call: `first_repeat(["R", "L", "U", "D"])`, expected: "-1"},
			caseAlias{name: "repeat-adjacent", call: `first_repeat(["R", "R"])`, expected: "1"},
			caseAlias{name: "single", call: `first_repeat(["R"])`, expected: "-1"},
			caseAlias{name: "late-repeat", call: `first_repeat(["A", "B", "C", "A"])`, expected: "3"},
		)
		stmt := fmt.Sprintf("%s重复指令：返回首次重复出现指令的下标（无重复返回 −1）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L4·哈希：最常用指令（计数 + 并列字典序，参考解 12 行）。
func moveFavoriteL4() question.Template {
	meta := question.Meta{ID: "move-favorite-L4-move", Topic: question.TopicHash,
		Difficulty: question.L4, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "tie-order", "single-distinct"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		moves := rng.PickN([]string{"U", "D", "L", "R"}, rng.Int(3, 7))
		kind := rng.Scenario("arena")
		sig := "def favorite_move(moves: list) -> list:"
		answer := "def favorite_move(moves: list) -> list:\n" +
			"    counts = {}\n" +
			"    for m in moves:\n" +
			"        counts[m] = counts.get(m, 0) + 1\n" +
			"    best = ''\n" +
			"    best_count = 0\n" +
			"    for m in counts:\n" +
			"        if counts[m] > best_count:\n" +
			"            best = m\n" +
			"            best_count = counts[m]\n" +
			"        elif counts[m] == best_count and m < best:\n" +
			"            best = m\n" +
			"    return [best, best_count]\n"
		best, cnt := favoriteGo(moves)
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("favorite_move(%s)", pyStrList(moves)), expected: fmt.Sprintf("[%s, %d]", pyStr(best), cnt)},
			caseAlias{name: "example-2", call: `favorite_move(["R", "L", "R"])`, expected: "['R', 2]"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "favorite_move([])", expected: "['', 0]"},
			caseAlias{name: "tie-order", call: `favorite_move(["L", "R"])`, expected: "['L', 1]"},
			caseAlias{name: "single-distinct", call: `favorite_move(["R"])`, expected: "['R', 1]"},
			caseAlias{name: "clear-winner", call: `favorite_move(["U", "U", "D"])`, expected: "['U', 2]"},
			caseAlias{name: "late-winner", call: `favorite_move(["D", "D", "U"])`, expected: "['D', 2]"},
		)
		stmt := fmt.Sprintf("%s常用指令：返回出现次数最多的指令及次数（并列取字典序较小者；空列表为 ['', 0]）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L5·哈希：指令调度（频率统计 + 冷却公式，参考解 13 行；12 号票批 5，
// 算法内核参考 codetop59 任务调度器，题面经情境池重包装）。
func moveScheduleL5() question.Template {
	meta := question.Meta{ID: "move-schedule-L5-move", Topic: question.TopicHash,
		Difficulty: question.L5, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "cooldown-zero", "single-task", "all-same"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		tasks := rng.PickN([]string{"A", "B", "C"}, rng.Int(3, 6))
		cooldown := rng.Int(0, 3)
		kind := rng.Scenario("arena")
		sig := "def min_rounds(tasks: list, cooldown: int) -> int:"
		answer := "def min_rounds(tasks: list, cooldown: int) -> int:\n" +
			"    counts = {}\n" +
			"    for t in tasks:\n" +
			"        counts[t] = counts.get(t, 0) + 1\n" +
			"    max_count = 0\n" +
			"    for v in counts.values():\n" +
			"        if v > max_count:\n" +
			"            max_count = v\n" +
			"    n_max = 0\n" +
			"    for v in counts.values():\n" +
			"        if v == max_count:\n" +
			"            n_max += 1\n" +
			"    slots = (max_count - 1) * (cooldown + 1) + n_max\n" +
			"    return slots if slots > len(tasks) else len(tasks)\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("min_rounds(%s, %d)", pyStrList(tasks), cooldown), expected: fmt.Sprintf("%d", minRoundsGo(tasks, cooldown))},
			caseAlias{name: "example-2", call: `min_rounds(["A", "A", "A", "B", "B", "B"], 2)`, expected: "8"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "min_rounds([], 2)", expected: "0"},
			caseAlias{name: "single-task", call: `min_rounds(["A"], 2)`, expected: "1"},
			caseAlias{name: "cooldown-zero", call: `min_rounds(["A", "A", "A"], 0)`, expected: "3"},
			caseAlias{name: "all-same", call: `min_rounds(["A", "A", "A"], 2)`, expected: "7"},
			caseAlias{name: "classic-full", call: `min_rounds(["A", "A", "A", "B", "B", "B"], 0)`, expected: "6"},
			caseAlias{name: "no-idle", call: `min_rounds(["A", "B", "C"], 2)`, expected: "3"},
		)
		stmt := fmt.Sprintf("%s指令调度：相同指令之间至少间隔 cooldown 个回合，求执行全部指令的最少回合数（空列表为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L1·哈希：弱点命中（单语法点：成员判断，参考解 1 行）。
func defendMembershipL1() question.Template {
	meta := question.Meta{ID: "defend-membership-L1-defend", Topic: question.TopicHash,
		Difficulty: question.L1, Action: question.ActionDefend, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		point := rng.Pick([]string{"fire", "ice", "thunder", "dark"})
		weak := []string{rng.Pick([]string{"fire", "ice", "thunder", "dark"})}
		if rng.Bool(0.5) {
			weak = append(weak, rng.Pick([]string{"wind", "earth"}))
		}
		kind := rng.Scenario("arena")
		sig := "def is_weak_point(point: str, weak: list) -> bool:"
		answer := "def is_weak_point(point: str, weak: list) -> bool:\n    return point in weak\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("is_weak_point(%s, %s)", pyStr(point), pyStrList(weak)), expected: pyBool(containsStrGo(weak, point))},
			caseAlias{name: "example-2", call: `is_weak_point("fire", ["ice"])`, expected: "False"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: `is_weak_point("fire", [])`, expected: "False"},
			caseAlias{name: "hit", call: `is_weak_point("ice", ["fire", "ice"])`, expected: "True"},
			caseAlias{name: "miss", call: `is_weak_point("dark", ["fire", "ice"])`, expected: "False"},
			caseAlias{name: "last-element", call: `is_weak_point("wind", ["fire", "wind"])`, expected: "True"},
			caseAlias{name: "case-sensitive", call: `is_weak_point("Fire", ["fire"])`, expected: "False"},
		)
		stmt := fmt.Sprintf("%s弱点命中：判断目标弱点是否在防御清单中（区分大小写；空清单为 False）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L2·哈希：重复命中（集合去重 + 长度比较，参考解 4 行）。
func defendDuplicateL2() question.Template {
	meta := question.Meta{ID: "defend-duplicate-L2-defend", Topic: question.TopicHash,
		Difficulty: question.L2, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "all-same"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		hits := rng.IntList(rng.Int(2, 5), 1, 6, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def has_duplicate(hits: list) -> bool:"
		answer := "def has_duplicate(hits: list) -> bool:\n    unique = set(hits)\n    if len(unique) < len(hits):\n        return True\n    return False\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("has_duplicate(%s)", pyList(hits)), expected: pyBool(hasDupGo(hits))},
			caseAlias{name: "example-2", call: "has_duplicate([1, 1])", expected: "True"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "has_duplicate([])", expected: "False"},
			caseAlias{name: "single", call: "has_duplicate([3])", expected: "False"},
			caseAlias{name: "all-same", call: "has_duplicate([2, 2, 2])", expected: "True"},
			caseAlias{name: "distinct", call: "has_duplicate([1, 2, 3])", expected: "False"},
			caseAlias{name: "duplicated-pair", call: "has_duplicate([2, 2, 3, 3])", expected: "True"},
		)
		stmt := fmt.Sprintf("%s重复命中：判断命中列表中是否存在重复值（空列表为 False）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L4·哈希：共同命中（集合交集 + 去重排序，参考解 9 行）。
func defendSharedL4() question.Template {
	meta := question.Meta{ID: "defend-shared-L4-defend", Topic: question.TopicHash,
		Difficulty: question.L4, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "no-overlap", "duplicates-in-b"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		a := rng.IntList(rng.Int(2, 4), 1, 6, question.ListOpts{})
		b := rng.IntList(rng.Int(2, 4), 1, 6, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def shared_hits(a: list, b: list) -> list:"
		answer := "def shared_hits(a: list, b: list) -> list:\n" +
			"    set_a = set(a)\n" +
			"    seen = set()\n" +
			"    shared = []\n" +
			"    for x in b:\n" +
			"        if x in set_a and x not in seen:\n" +
			"            shared.append(x)\n" +
			"            seen.add(x)\n" +
			"    shared.sort()\n" +
			"    return shared\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("shared_hits(%s, %s)", pyList(a), pyList(b)), expected: pyList(sharedHitsGo(a, b))},
			caseAlias{name: "example-2", call: "shared_hits([1, 2], [2, 3])", expected: "[2]"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "shared_hits([], [])", expected: "[]"},
			caseAlias{name: "no-overlap", call: "shared_hits([1, 2], [3])", expected: "[]"},
			caseAlias{name: "duplicates-in-b", call: "shared_hits([1], [1, 1, 1])", expected: "[1]"},
			caseAlias{name: "unique-output", call: "shared_hits([1, 2, 2], [2, 2])", expected: "[2]"},
			caseAlias{name: "sorted", call: "shared_hits([3, 1], [1, 3])", expected: "[1, 3]"},
		)
		stmt := fmt.Sprintf("%s共同弱点：返回两组命中列表的交集（去重、升序；无交集为空列表）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L5·哈希：多数命中（计数 + 严格过半判定 + 最大计数选择，参考解 14 行）。
func defendMajorityL5() question.Template {
	meta := question.Meta{ID: "defend-majority-L5-defend", Topic: question.TopicHash,
		Difficulty: question.L5, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "no-majority", "all-same", "exactly-half"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		hits := rng.IntList(rng.Int(2, 6), 1, 4, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def majority_hit(hits: list) -> int:"
		answer := "def majority_hit(hits: list) -> int:\n" +
			"    if not hits:\n" +
			"        return -1\n" +
			"    counts = {}\n" +
			"    for h in hits:\n" +
			"        counts[h] = counts.get(h, 0) + 1\n" +
			"    threshold = len(hits) // 2\n" +
			"    best = -1\n" +
			"    best_count = 0\n" +
			"    for h in counts:\n" +
			"        if counts[h] > threshold:\n" +
			"            if counts[h] > best_count:\n" +
			"                best = h\n" +
			"                best_count = counts[h]\n" +
			"    return best\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("majority_hit(%s)", pyList(hits)), expected: fmt.Sprintf("%d", majorityHitGo(hits))},
			caseAlias{name: "example-2", call: "majority_hit([2, 2, 1])", expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "majority_hit([])", expected: "-1"},
			caseAlias{name: "exactly-half", call: "majority_hit([1, 2])", expected: "-1"},
			caseAlias{name: "no-majority", call: "majority_hit([1, 2, 3])", expected: "-1"},
			caseAlias{name: "all-same", call: "majority_hit([3, 3])", expected: "3"},
			caseAlias{name: "clear", call: "majority_hit([3, 3, 3, 1])", expected: "3"},
			caseAlias{name: "single", call: "majority_hit([7])", expected: "7"},
		)
		stmt := fmt.Sprintf("%s多数命中：返回出现次数严格超过一半的命中值（不存在或空列表返回 −1）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L1·哈希：材料核验（单语法点：dict.get 判正，参考解 1 行）。
func skillIngredientL1() question.Template {
	meta := question.Meta{ID: "skill-ingredient-L1-skill", Topic: question.TopicHash,
		Difficulty: question.L1, Action: question.ActionSkill, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		item := rng.Pick([]string{"herb", "ore", "gem"})
		bag := map[string]int{}
		bag[rng.Pick([]string{"herb", "ore", "gem"})] = rng.Int(0, 3)
		bag[rng.Pick([]string{"herb", "ore", "gem"})] = rng.Int(0, 3)
		kind := rng.Scenario("alchemy")
		sig := "def has_ingredient(bag: dict, item: str) -> bool:"
		answer := "def has_ingredient(bag: dict, item: str) -> bool:\n    return bag.get(item, 0) > 0\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("has_ingredient(%s, %s)", pyDictGo(bag), pyStr(item)), expected: pyBool(bag[item] > 0)},
			caseAlias{name: "example-2", call: "has_ingredient({'herb': 2}, 'herb')", expected: "True"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "has_ingredient({}, 'herb')", expected: "False"},
			caseAlias{name: "zero-count", call: "has_ingredient({'ore': 0}, 'ore')", expected: "False"},
			caseAlias{name: "absent", call: "has_ingredient({'gem': 1}, 'ore')", expected: "False"},
			caseAlias{name: "present", call: "has_ingredient({'gem': 2, 'ore': 1}, 'ore')", expected: "True"},
			caseAlias{name: "single", call: "has_ingredient({'herb': 1}, 'herb')", expected: "True"},
		)
		stmt := fmt.Sprintf("%s材料核验：背包中该材料数量大于 0 时返回 True（缺省/空背包为 False）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L2·哈希：成本合计（遍历值求和，参考解 5 行）。
func skillCostL2() question.Template {
	meta := question.Meta{ID: "skill-cost-L2-skill", Topic: question.TopicHash,
		Difficulty: question.L2, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"empty-input"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		costs := map[string]int{}
		costs[rng.Pick([]string{"herb", "ore", "gem"})] = rng.Int(0, 9)
		costs[rng.Pick([]string{"herb", "ore", "gem"})] = rng.Int(0, 9)
		kind := rng.Scenario("alchemy")
		sig := "def total_cost(costs: dict) -> int:"
		answer := "def total_cost(costs: dict) -> int:\n    total = 0\n    for v in costs.values():\n        total += v\n    return total\n"
		total := 0
		for _, v := range costs {
			total += v
		}
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("total_cost(%s)", pyDictGo(costs)), expected: fmt.Sprintf("%d", total)},
			caseAlias{name: "example-2", call: "total_cost({'a': 1, 'b': 2})", expected: "3"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "total_cost({})", expected: "0"},
			caseAlias{name: "single", call: "total_cost({'a': 5})", expected: "5"},
			caseAlias{name: "zeros", call: "total_cost({'a': 0, 'b': 0})", expected: "0"},
			caseAlias{name: "three-keys", call: "total_cost({'a': 5, 'b': 5, 'c': 5})", expected: "15"},
			caseAlias{name: "mixed", call: "total_cost({'x': 2, 'y': 8})", expected: "10"},
		)
		stmt := fmt.Sprintf("%s成本合计：求背包中全部材料的成本总和（空背包为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L4·哈希：库存对账（并集键 + 差值排序，参考解 9 行）。
func skillStockDiffL4() question.Template {
	meta := question.Meta{ID: "skill-stockdiff-L4-skill", Topic: question.TopicHash,
		Difficulty: question.L4, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"empty-input", "only-left", "only-right"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		a := map[string]int{}
		b := map[string]int{}
		a[rng.Pick([]string{"herb", "ore", "gem"})] = rng.Int(0, 5)
		a[rng.Pick([]string{"herb", "ore", "gem"})] = rng.Int(0, 5)
		b[rng.Pick([]string{"herb", "ore", "gem"})] = rng.Int(0, 5)
		b[rng.Pick([]string{"herb", "ore", "gem"})] = rng.Int(0, 5)
		kind := rng.Scenario("alchemy")
		sig := "def stock_diff(a: dict, b: dict) -> list:"
		answer := "def stock_diff(a: dict, b: dict) -> list:\n" +
			"    keys = set(a) | set(b)\n" +
			"    diff = []\n" +
			"    for k in keys:\n" +
			"        va = a.get(k, 0)\n" +
			"        vb = b.get(k, 0)\n" +
			"        if va != vb:\n" +
			"            diff.append([k, va - vb])\n" +
			"    diff.sort()\n" +
			"    return diff\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("stock_diff(%s, %s)", pyDictGo(a), pyDictGo(b)), expected: stockDiffRepr(a, b)},
			caseAlias{name: "example-2", call: "stock_diff({'a': 2}, {'a': 1})", expected: "[['a', 1]]"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "stock_diff({}, {})", expected: "[]"},
			caseAlias{name: "equal", call: "stock_diff({'a': 1}, {'a': 1})", expected: "[]"},
			caseAlias{name: "only-left", call: "stock_diff({'a': 2}, {'a': 1})", expected: "[['a', 1]]"},
			caseAlias{name: "only-right", call: "stock_diff({}, {'b': 3})", expected: "[['b', -3]]"},
			caseAlias{name: "multi", call: "stock_diff({'a': 1, 'b': 2}, {'b': 2, 'c': 5})", expected: "[['a', 1], ['c', -5]]"},
		)
		stmt := fmt.Sprintf("%s库存对账：比较两份库存，返回键值差异列表 [[键, a−b], ...]（按差异列表升序；空差异为空列表）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L5·哈希：配方盘点（需求遍历 + 缺料/余料双列表 + 就绪判定，参考解 13 行）。
func skillCraftReportL5() question.Template {
	meta := question.Meta{ID: "skill-craftreport-L5-skill", Topic: question.TopicHash,
		Difficulty: question.L5, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"empty-input", "exact-match", "deficit", "surplus"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		have := map[string]int{}
		recipe := map[string]int{}
		have[rng.Pick([]string{"herb", "ore", "gem"})] = rng.Int(0, 4)
		have[rng.Pick([]string{"herb", "ore", "gem"})] = rng.Int(0, 4)
		recipe[rng.Pick([]string{"herb", "ore", "gem"})] = rng.Int(1, 4)
		recipe[rng.Pick([]string{"herb", "ore", "gem"})] = rng.Int(1, 4)
		kind := rng.Scenario("alchemy")
		sig := "def craft_report(have: dict, recipe: dict) -> list:"
		answer := "def craft_report(have: dict, recipe: dict) -> list:\n" +
			"    missing = []\n" +
			"    surplus = []\n" +
			"    for item in recipe:\n" +
			"        need = recipe[item]\n" +
			"        got = have.get(item, 0)\n" +
			"        if got < need:\n" +
			"            missing.append([item, need - got])\n" +
			"        elif got > need:\n" +
			"            surplus.append([item, got - need])\n" +
			"    missing.sort()\n" +
			"    surplus.sort()\n" +
			"    ready = len(missing) == 0\n" +
			"    return [ready, missing, surplus]\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("craft_report(%s, %s)", pyDictGo(have), pyDictGo(recipe)), expected: craftReportRepr(have, recipe)},
			caseAlias{name: "example-2", call: "craft_report({'a': 1}, {'a': 2})", expected: "[False, [['a', 1]], []]"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "craft_report({}, {})", expected: "[True, [], []]"},
			caseAlias{name: "exact-match", call: "craft_report({'a': 2}, {'a': 2})", expected: "[True, [], []]"},
			caseAlias{name: "deficit", call: "craft_report({'a': 1}, {'a': 2})", expected: "[False, [['a', 1]], []]"},
			caseAlias{name: "surplus", call: "craft_report({'a': 3}, {'a': 1})", expected: "[True, [], [['a', 2]]]"},
			caseAlias{name: "mixed", call: "craft_report({'a': 3, 'b': 1}, {'a': 2, 'b': 2})", expected: "[False, [['b', 1]], [['a', 1]]]"},
		)
		stmt := fmt.Sprintf("%s配方盘点：按配方逐项比对库存，返回 [是否就绪, 缺料列表, 余料列表]（列表项为 [材料, 数量]，升序）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// ---- 期望值计算（与参考解同语义） ----

func stepForGo(cmd string) int {
	if cmd == "F" {
		return 1
	}
	if cmd == "B" {
		return -1
	}
	return 0
}

func countCmdGo(moves []string, cmd string) int {
	n := 0
	for _, m := range moves {
		if m == cmd {
			n++
		}
	}
	return n
}

func firstRepeatGo(moves []string) int {
	seen := map[string]bool{}
	for i, m := range moves {
		if seen[m] {
			return i
		}
		seen[m] = true
	}
	return -1
}

func favoriteGo(moves []string) (string, int) {
	counts := map[string]int{}
	for _, m := range moves {
		counts[m]++
	}
	best, bestCount := "", 0
	for m, c := range counts {
		if c > bestCount || (c == bestCount && m < best) {
			best, bestCount = m, c
		}
	}
	return best, bestCount
}

func minRoundsGo(tasks []string, cooldown int) int {
	counts := map[string]int{}
	for _, t := range tasks {
		counts[t]++
	}
	maxCount, nMax := 0, 0
	for _, v := range counts {
		if v > maxCount {
			maxCount = v
		}
	}
	for _, v := range counts {
		if v == maxCount {
			nMax++
		}
	}
	slots := (maxCount-1)*(cooldown+1) + nMax
	if slots > len(tasks) {
		return slots
	}
	return len(tasks)
}

func containsStrGo(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func hasDupGo(nums []int) bool {
	seen := map[int]bool{}
	for _, v := range nums {
		if seen[v] {
			return true
		}
		seen[v] = true
	}
	return false
}

func sharedHitsGo(a, b []int) []int {
	setA := map[int]bool{}
	for _, v := range a {
		setA[v] = true
	}
	seen := map[int]bool{}
	out := []int{}
	for _, v := range b {
		if setA[v] && !seen[v] {
			out = append(out, v)
			seen[v] = true
		}
	}
	sort.Ints(out)
	return out
}

func majorityHitGo(hits []int) int {
	if len(hits) == 0 {
		return -1
	}
	counts := map[int]int{}
	for _, h := range hits {
		counts[h]++
	}
	threshold := len(hits) / 2
	best, bestCount := -1, 0
	for h, c := range counts {
		if c > threshold && c > bestCount {
			best, bestCount = h, c
		}
	}
	return best
}

// pyDictGo：dict 字面量 repr（键排序保证确定性；仅支持 str→int）。
func pyDictGo(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("'%s': %d", k, m[k])
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// stockDiffRepr：stock_diff 的期望 repr（并集键、差值为 0 不出现、按键升序）。
func stockDiffRepr(a, b map[string]int) string {
	seen := map[string]bool{}
	keys := []string{}
	for k := range a {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for k := range b {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	parts := []string{}
	for _, k := range keys {
		d := a[k] - b[k]
		if d != 0 {
			parts = append(parts, fmt.Sprintf("['%s', %d]", k, d))
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// craftReportRepr：craft_report 的期望 repr（缺料/余料按键升序）。
func craftReportRepr(have, recipe map[string]int) string {
	keys := make([]string, 0, len(recipe))
	for k := range recipe {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	missing := []string{}
	surplus := []string{}
	for _, k := range keys {
		need, got := recipe[k], have[k]
		if got < need {
			missing = append(missing, fmt.Sprintf("['%s', %d]", k, need-got))
		} else if got > need {
			surplus = append(surplus, fmt.Sprintf("['%s', %d]", k, got-need))
		}
	}
	ready := "False"
	if len(missing) == 0 {
		ready = "True"
	}
	return fmt.Sprintf("[%s, [%s], [%s]]", ready, strings.Join(missing, ", "), strings.Join(surplus, ", "))
}
