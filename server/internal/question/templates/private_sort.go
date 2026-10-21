// 私有行动题·排序与贪心主题（12 号票批 6：move/defend/skill × L1-L5）。
// 形式均为 complete；题型标注进入矩阵格（coverage_test.go 口径）。
package templates

import (
	"fmt"
	"sort"

	"codewar/server/internal/question"
)

// 移动 L1·排序：最大步幅（单语法点：sorted + 下标，参考解 1 行；约定非空）。
func moveSortPickL1() question.Template {
	meta := question.Meta{ID: "move-sortpick-L1-move", Topic: question.TopicSort,
		Difficulty: question.L1, Action: question.ActionMove, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		steps := rng.IntList(rng.Int(2, 5), 0, 9, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def max_step(steps: list) -> int:"
		answer := "def max_step(steps: list) -> int:\n    return sorted(steps)[-1]\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("max_step(%s)", pyList(steps)), expected: fmt.Sprintf("%d", maxListGo(steps))},
			caseAlias{name: "example-2", call: "max_step([3, 9, 4])", expected: "9"},
		)
		hidden := cases(
			caseAlias{name: "single", call: "max_step([7])", expected: "7"},
			caseAlias{name: "sorted", call: "max_step([1, 2, 3])", expected: "3"},
			caseAlias{name: "reverse-sorted", call: "max_step([9, 5, 3])", expected: "9"},
			caseAlias{name: "duplicates", call: "max_step([4, 9, 9])", expected: "9"},
			caseAlias{name: "with-zero", call: "max_step([0, 5])", expected: "5"},
		)
		stmt := fmt.Sprintf("%s最大步幅：返回列表中的最大步数（约定非空）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L2·排序：最省 k 步（排序 + 取前 k 个之和，参考解 5 行）。
func moveCheapestL2() question.Template {
	meta := question.Meta{ID: "move-cheapest-L2-move", Topic: question.TopicSort,
		Difficulty: question.L2, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"k-zero", "empty-input"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		steps := rng.IntList(rng.Int(2, 5), 0, 9, question.ListOpts{})
		k := rng.Int(0, len(steps))
		kind := rng.Scenario("arena")
		sig := "def cheapest_steps(steps: list, k: int) -> int:"
		answer := "def cheapest_steps(steps: list, k: int) -> int:\n    ordered = sorted(steps)\n    total = 0\n    for i in range(k):\n        total += ordered[i]\n    return total\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("cheapest_steps(%s, %d)", pyList(steps), k), expected: fmt.Sprintf("%d", cheapestGo(steps, k))},
			caseAlias{name: "example-2", call: "cheapest_steps([3, 1, 2], 2)", expected: "3"},
		)
		hidden := cases(
			caseAlias{name: "k-zero", call: "cheapest_steps([4, 3, 2, 1], 0)", expected: "0"},
			caseAlias{name: "empty-input", call: "cheapest_steps([], 0)", expected: "0"},
			caseAlias{name: "single", call: "cheapest_steps([5], 1)", expected: "5"},
			caseAlias{name: "all", call: "cheapest_steps([2, 2], 2)", expected: "4"},
			caseAlias{name: "with-zero", call: "cheapest_steps([0, 5], 1)", expected: "0"},
		)
		stmt := fmt.Sprintf("%s最省 k 步：排序后取前 k 小的步数之和（k 可为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L3·排序：连续可达（排序 + 相邻差检查，参考解 6 行）。
func moveWalkAllL3() question.Template {
	meta := question.Meta{ID: "move-walkall-L3-move", Topic: question.TopicSort,
		Difficulty: question.L3, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-element", "gap"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		steps := rng.IntList(rng.Int(2, 5), 0, 6, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def can_walk_all(steps: list) -> bool:"
		answer := "def can_walk_all(steps: list) -> bool:\n    ordered = sorted(steps)\n    for i in range(1, len(ordered)):\n        if ordered[i] - ordered[i - 1] > 1:\n            return False\n    return True\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("can_walk_all(%s)", pyList(steps)), expected: pyBool(canWalkGo(steps))},
			caseAlias{name: "example-2", call: "can_walk_all([1, 2, 3])", expected: "True"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "can_walk_all([])", expected: "True"},
			caseAlias{name: "single-element", call: "can_walk_all([1])", expected: "True"},
			caseAlias{name: "gap", call: "can_walk_all([1, 3])", expected: "False"},
			caseAlias{name: "duplicates", call: "can_walk_all([2, 2, 3])", expected: "True"},
			caseAlias{name: "trailing-gap", call: "can_walk_all([1, 1, 3])", expected: "False"},
		)
		stmt := fmt.Sprintf("%s连续可达：排序后若相邻步数差均不超过 1 则返回 True（空/单元素为 True）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L4·排序：名次映射（排序 + 字典回填，参考解 8 行）。
func moveRanksL4() question.Template {
	meta := question.Meta{ID: "move-ranks-L4-move", Topic: question.TopicSort,
		Difficulty: question.L4, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"duplicates", "single-element", "all-same"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		scores := rng.IntList(rng.Int(2, 5), 1, 9, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def rank_list(scores: list) -> list:"
		answer := "def rank_list(scores: list) -> list:\n" +
			"    ordered = sorted(scores)\n" +
			"    ranks = {}\n" +
			"    for i in range(len(ordered)):\n" +
			"        ranks[ordered[i]] = i + 1\n" +
			"    result = []\n" +
			"    for s in scores:\n" +
			"        result.append(ranks[s])\n" +
			"    return result\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("rank_list(%s)", pyList(scores)), expected: pyList(rankListGo(scores))},
			caseAlias{name: "example-2", call: "rank_list([20, 10, 20])", expected: "[3, 1, 3]"},
		)
		hidden := cases(
			caseAlias{name: "duplicates", call: "rank_list([20, 10, 20])", expected: "[3, 1, 3]"},
			caseAlias{name: "single-element", call: "rank_list([5])", expected: "[1]"},
			caseAlias{name: "all-same", call: "rank_list([7, 7, 7])", expected: "[3, 3, 3]"},
			caseAlias{name: "two-values", call: "rank_list([10, 20])", expected: "[1, 2]"},
			caseAlias{name: "unsorted", call: "rank_list([3, 1, 2])", expected: "[3, 1, 2]"},
		)
		stmt := fmt.Sprintf("%s名次映射：按分数升序给出每个分数的名次（1 起；并列分数取较大名次），按原顺序返回。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L5·排序：最多行程（按结束时间排序 + 贪心选择，参考解 13 行；12 号票批 6，
// 算法内核参考 codetop59 无重叠区间，题面经情境池重包装）。
func moveEventsL5() question.Template {
	meta := question.Meta{ID: "move-events-L5-move", Topic: question.TopicSort,
		Difficulty: question.L5, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "chained", "disjoint", "all-overlapping"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		intervals := randRanges(rng, rng.Int(2, 4))
		kind := rng.Scenario("arena")
		sig := "def max_events(intervals: list) -> int:"
		answer := "def max_events(intervals: list) -> int:\n" +
			"    if not intervals:\n" +
			"        return 0\n" +
			"    ordered = []\n" +
			"    for item in intervals:\n" +
			"        ordered.append([item[0], item[1]])\n" +
			"    ordered.sort(key=lambda x: x[1])\n" +
			"    count = 0\n" +
			"    last_end = -1\n" +
			"    for start, end in ordered:\n" +
			"        if start >= last_end:\n" +
			"            count += 1\n" +
			"            last_end = end\n" +
			"    return count\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("max_events(%s)", pyRanges(intervals)), expected: fmt.Sprintf("%d", maxEventsGo(intervals))},
			caseAlias{name: "example-2", call: "max_events([[1, 2], [2, 3]])", expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "max_events([])", expected: "0"},
			caseAlias{name: "chained", call: "max_events([[1, 2], [2, 3]])", expected: "2"},
			caseAlias{name: "disjoint", call: "max_events([[1, 2], [4, 5]])", expected: "2"},
			caseAlias{name: "all-overlapping", call: "max_events([[1, 5], [2, 6], [3, 7]])", expected: "1"},
			caseAlias{name: "nested", call: "max_events([[1, 10], [2, 3], [4, 5]])", expected: "2"},
		)
		stmt := fmt.Sprintf("%s最多行程：区间为可参加的行程时间，行程端点相接可连续参加，求最多参加数。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L1·排序：最弱一击（排序 + 取首元素，参考解 2 行）。
func defendWeakestL1() question.Template {
	meta := question.Meta{ID: "defend-weakest-L1-defend", Topic: question.TopicSort,
		Difficulty: question.L1, Action: question.ActionDefend, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		hits := rng.IntList(rng.Int(2, 5), 1, 12, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def weakest(hits: list) -> int:"
		answer := "def weakest(hits: list) -> int:\n    ordered = sorted(hits)\n    return ordered[0]\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("weakest(%s)", pyList(hits)), expected: fmt.Sprintf("%d", minListGo(hits))},
			caseAlias{name: "example-2", call: "weakest([8, 3, 5])", expected: "3"},
		)
		hidden := cases(
			caseAlias{name: "single", call: "weakest([9])", expected: "9"},
			caseAlias{name: "sorted", call: "weakest([1, 2, 3])", expected: "1"},
			caseAlias{name: "reverse-sorted", call: "weakest([9, 5, 3])", expected: "3"},
			caseAlias{name: "duplicates", call: "weakest([4, 4])", expected: "4"},
			caseAlias{name: "mixed", call: "weakest([6, 2, 8])", expected: "2"},
		)
		stmt := fmt.Sprintf("%s最弱一击：返回命中列表中的最小伤害值（约定非空）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L2·排序：最强 k 击之和（排序 + 尾部求和，参考解 5 行）。
func defendTopSumL2() question.Template {
	meta := question.Meta{ID: "defend-topsum-L2-defend", Topic: question.TopicSort,
		Difficulty: question.L2, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"k-zero", "empty-input"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		hits := rng.IntList(rng.Int(2, 5), 1, 9, question.ListOpts{})
		k := rng.Int(0, len(hits))
		kind := rng.Scenario("arena")
		sig := "def top_sum(hits: list, k: int) -> int:"
		answer := "def top_sum(hits: list, k: int) -> int:\n    ordered = sorted(hits)\n    total = 0\n    for i in range(k):\n        total += ordered[-1 - i]\n    return total\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("top_sum(%s, %d)", pyList(hits), k), expected: fmt.Sprintf("%d", topSumGo(hits, k))},
			caseAlias{name: "example-2", call: "top_sum([1, 2, 3], 2)", expected: "5"},
		)
		hidden := cases(
			caseAlias{name: "k-zero", call: "top_sum([1, 2, 3], 0)", expected: "0"},
			caseAlias{name: "empty-input", call: "top_sum([], 0)", expected: "0"},
			caseAlias{name: "single", call: "top_sum([5], 1)", expected: "5"},
			caseAlias{name: "all", call: "top_sum([2, 2], 2)", expected: "4"},
			caseAlias{name: "duplicates", call: "top_sum([3, 3, 1], 2)", expected: "6"},
		)
		stmt := fmt.Sprintf("%s最强 k 击：排序后取最大的 k 个命中值之和（k 可为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L3·排序：第二强（去重排序 + 边界，参考解 7 行）。
func defendSecondL3() question.Template {
	meta := question.Meta{ID: "defend-second-L3-defend", Topic: question.TopicSort,
		Difficulty: question.L3, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"single-distinct", "empty-input", "duplicates"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		hits := rng.IntList(rng.Int(2, 5), 1, 9, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def second_largest(hits: list) -> int:"
		answer := "def second_largest(hits: list) -> int:\n    ordered = sorted(set(hits), reverse=True)\n    if len(ordered) < 2:\n        return -1\n    best = ordered[0]\n    second = ordered[1]\n    return second\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("second_largest(%s)", pyList(hits)), expected: fmt.Sprintf("%d", secondLargestGo(hits))},
			caseAlias{name: "example-2", call: "second_largest([5, 3])", expected: "3"},
		)
		hidden := cases(
			caseAlias{name: "single-distinct", call: "second_largest([5, 5])", expected: "-1"},
			caseAlias{name: "empty-input", call: "second_largest([])", expected: "-1"},
			caseAlias{name: "single", call: "second_largest([5])", expected: "-1"},
			caseAlias{name: "duplicates", call: "second_largest([3, 3, 4])", expected: "3"},
			caseAlias{name: "mixed", call: "second_largest([1, 3, 3, 2])", expected: "2"},
		)
		stmt := fmt.Sprintf("%s第二强：返回去重后的第二大命中值（不足两个不同值返回 −1）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L4·排序：命中聚类（排序 + 相邻差聚类，参考解 10 行）。
func defendClusterL4() question.Template {
	meta := question.Meta{ID: "defend-cluster-L4-defend", Topic: question.TopicSort,
		Difficulty: question.L4, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-element", "all-one-group", "separated"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		hits := rng.IntList(rng.Int(2, 6), 1, 14, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def cluster_hits(hits: list) -> int:"
		answer := "def cluster_hits(hits: list) -> int:\n" +
			"    if not hits:\n" +
			"        return 0\n" +
			"    ordered = sorted(hits)\n" +
			"    groups = 1\n" +
			"    last = ordered[0]\n" +
			"    for h in ordered[1:]:\n" +
			"        if h - last > 2:\n" +
			"            groups += 1\n" +
			"        last = h\n" +
			"    return groups\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("cluster_hits(%s)", pyList(hits)), expected: fmt.Sprintf("%d", clusterHitsGo(hits))},
			caseAlias{name: "example-2", call: "cluster_hits([1, 2, 5, 6])", expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "cluster_hits([])", expected: "0"},
			caseAlias{name: "single-element", call: "cluster_hits([5])", expected: "1"},
			caseAlias{name: "all-one-group", call: "cluster_hits([1, 2, 3])", expected: "1"},
			caseAlias{name: "separated", call: "cluster_hits([1, 4])", expected: "2"},
			caseAlias{name: "unsorted-groups", call: "cluster_hits([10, 1, 3])", expected: "2"},
		)
		stmt := fmt.Sprintf("%s命中聚类：排序后相邻差值不超过 2 的命中归为同一组，返回组数（空列表为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L5·排序：乱序区间（排序 + 双端扫描，参考解 13 行；12 号票批 6，
// 算法内核参考 codetop59 最短无序连续子数组，题面经情境池重包装）。
func defendSpanL5() question.Template {
	meta := question.Meta{ID: "defend-span-L5-defend", Topic: question.TopicSort,
		Difficulty: question.L5, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "sorted-input", "single-element", "reverse-sorted"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), 1, 9, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def unsorted_span(nums: list) -> int:"
		answer := "def unsorted_span(nums: list) -> int:\n" +
			"    n = len(nums)\n" +
			"    if n == 0:\n" +
			"        return 0\n" +
			"    ordered = sorted(nums)\n" +
			"    left = 0\n" +
			"    while left < n and nums[left] == ordered[left]:\n" +
			"        left += 1\n" +
			"    if left == n:\n" +
			"        return 0\n" +
			"    right = n - 1\n" +
			"    while right > left and nums[right] == ordered[right]:\n" +
			"        right -= 1\n" +
			"    return right - left + 1\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("unsorted_span(%s)", pyList(nums)), expected: fmt.Sprintf("%d", unsortedSpanGo(nums))},
			caseAlias{name: "example-2", call: "unsorted_span([2, 6, 4, 8, 10, 9, 15])", expected: "5"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "unsorted_span([])", expected: "0"},
			caseAlias{name: "sorted-input", call: "unsorted_span([1, 2, 3])", expected: "0"},
			caseAlias{name: "single-element", call: "unsorted_span([5])", expected: "0"},
			caseAlias{name: "reverse-sorted", call: "unsorted_span([3, 2, 1])", expected: "3"},
			caseAlias{name: "middle-pair", call: "unsorted_span([1, 2, 4, 3, 5])", expected: "2"},
		)
		stmt := fmt.Sprintf("%s乱序区间：若把其中一段连续区间升序重排后整个列表有序，求该区间的最短长度（已有序为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L1·排序：最强一击（单语法点：sorted 取尾，参考解 2 行）。
func skillStrongestL1() question.Template {
	meta := question.Meta{ID: "skill-strongest-L1-skill", Topic: question.TopicSort,
		Difficulty: question.L1, Action: question.ActionSkill, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(2, 5), 1, 12, question.ListOpts{})
		kind := rng.Scenario("spell")
		sig := "def strongest(nums: list) -> int:"
		answer := "def strongest(nums: list) -> int:\n    ordered = sorted(nums)\n    return ordered[-1]\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("strongest(%s)", pyList(nums)), expected: fmt.Sprintf("%d", maxListGo(nums))},
			caseAlias{name: "example-2", call: "strongest([3, 9, 4])", expected: "9"},
		)
		hidden := cases(
			caseAlias{name: "single", call: "strongest([6])", expected: "6"},
			caseAlias{name: "sorted", call: "strongest([1, 2, 3])", expected: "3"},
			caseAlias{name: "reverse-sorted", call: "strongest([9, 5, 3])", expected: "9"},
			caseAlias{name: "duplicates", call: "strongest([4, 4])", expected: "4"},
			caseAlias{name: "with-zero", call: "strongest([0, 5])", expected: "5"},
		)
		stmt := fmt.Sprintf("%s最强一击：返回列表中的最大能量值（约定非空）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L2·排序：护盾充能（排序 + 前 k 个求和，参考解 5 行）。
func skillShieldL2() question.Template {
	meta := question.Meta{ID: "skill-shield-L2-skill", Topic: question.TopicSort,
		Difficulty: question.L2, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"k-zero", "empty-input"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		hits := rng.IntList(rng.Int(2, 5), 1, 9, question.ListOpts{})
		k := rng.Int(0, len(hits))
		kind := rng.Scenario("spell")
		sig := "def shield_sum(hits: list, k: int) -> int:"
		answer := "def shield_sum(hits: list, k: int) -> int:\n    ordered = sorted(hits)\n    total = 0\n    for i in range(k):\n        total += ordered[i]\n    return total\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("shield_sum(%s, %d)", pyList(hits), k), expected: fmt.Sprintf("%d", cheapestGo(hits, k))},
			caseAlias{name: "example-2", call: "shield_sum([5, 1, 3], 2)", expected: "4"},
		)
		hidden := cases(
			caseAlias{name: "k-zero", call: "shield_sum([4, 3], 0)", expected: "0"},
			caseAlias{name: "empty-input", call: "shield_sum([], 0)", expected: "0"},
			caseAlias{name: "single", call: "shield_sum([5], 1)", expected: "5"},
			caseAlias{name: "all", call: "shield_sum([2, 2], 2)", expected: "4"},
			caseAlias{name: "sorted-input", call: "shield_sum([1, 2, 3], 2)", expected: "3"},
		)
		stmt := fmt.Sprintf("%s护盾充能：排序后取最小的 k 个能量值之和（k 可为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L3·排序：近距配对（排序 + 相邻差计数，参考解 6 行）。
func skillClosePairsL3() question.Template {
	meta := question.Meta{ID: "skill-closepairs-L3-skill", Topic: question.TopicSort,
		Difficulty: question.L3, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"empty-input", "no-close", "all-close"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(2, 6), 1, 9, question.ListOpts{})
		limit := rng.Int(0, 3)
		kind := rng.Scenario("spell")
		sig := "def close_pairs(nums: list, limit: int) -> int:"
		answer := "def close_pairs(nums: list, limit: int) -> int:\n    ordered = sorted(nums)\n    count = 0\n    for i in range(1, len(ordered)):\n        if ordered[i] - ordered[i - 1] <= limit:\n            count += 1\n    return count\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("close_pairs(%s, %d)", pyList(nums), limit), expected: fmt.Sprintf("%d", closePairsGo(nums, limit))},
			caseAlias{name: "example-2", call: "close_pairs([1, 2, 3], 1)", expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "close_pairs([], 1)", expected: "0"},
			caseAlias{name: "no-close", call: "close_pairs([1, 5, 10], 1)", expected: "0"},
			caseAlias{name: "all-close", call: "close_pairs([1, 2, 3], 1)", expected: "2"},
			caseAlias{name: "zero-limit", call: "close_pairs([5, 5, 6], 0)", expected: "1"},
			caseAlias{name: "unsorted", call: "close_pairs([10, 1, 2], 1)", expected: "1"},
		)
		stmt := fmt.Sprintf("%s近距配对：排序后统计相邻两数差不超过 limit 的次数。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L4·排序：奖牌榜（去重 + 降序取前三，参考解 12 行）。
func skillMedalL4() question.Template {
	meta := question.Meta{ID: "skill-medal-L4-skill", Topic: question.TopicSort,
		Difficulty: question.L4, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"empty-input", "fewer-than-three", "duplicates"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		scores := rng.IntList(rng.Int(2, 6), 1, 9, question.ListOpts{})
		kind := rng.Scenario("spell")
		sig := "def medalists(scores: list) -> list:"
		answer := "def medalists(scores: list) -> list:\n" +
			"    unique = []\n" +
			"    for s in scores:\n" +
			"        if s not in unique:\n" +
			"            unique.append(s)\n" +
			"    unique.sort(reverse=True)\n" +
			"    result = []\n" +
			"    limit = 3\n" +
			"    if len(unique) < limit:\n" +
			"        limit = len(unique)\n" +
			"    for i in range(limit):\n" +
			"        result.append(unique[i])\n" +
			"    return result\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("medalists(%s)", pyList(scores)), expected: pyList(medalistsGo(scores))},
			caseAlias{name: "example-2", call: "medalists([9, 2, 7, 4])", expected: "[9, 7, 4]"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "medalists([])", expected: "[]"},
			caseAlias{name: "fewer-than-three", call: "medalists([5])", expected: "[5]"},
			caseAlias{name: "duplicates", call: "medalists([5, 5, 3, 1])", expected: "[5, 3, 1]"},
			caseAlias{name: "single-distinct", call: "medalists([1, 1, 1])", expected: "[1]"},
			caseAlias{name: "exact-three", call: "medalists([5, 3, 1])", expected: "[5, 3, 1]"},
		)
		stmt := fmt.Sprintf("%s奖牌榜：去重后按降序返回前三名分数（不足三名则全部返回）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L5·排序：最少交换（排序 + 置换环计数，参考解 18 行；12 号票批 6，
// 算法内核参考 codetop59 最少交换次数使数组有序，题面经情境池重包装）。
func skillSwapsL5() question.Template {
	meta := question.Meta{ID: "skill-swaps-L5-skill", Topic: question.TopicSort,
		Difficulty: question.L5, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"empty-input", "sorted-input", "single-cycle", "reverse-sorted"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(3, 6), 1, 12, question.ListOpts{Distinct: true})
		kind := rng.Scenario("spell")
		sig := "def min_swaps(nums: list) -> int:"
		answer := "def min_swaps(nums: list) -> int:\n" +
			"    n = len(nums)\n" +
			"    ordered = sorted(nums)\n" +
			"    position = {}\n" +
			"    for i in range(n):\n" +
			"        position[ordered[i]] = i\n" +
			"    visited = [False] * n\n" +
			"    swaps = 0\n" +
			"    for i in range(n):\n" +
			"        if visited[i] or nums[i] == ordered[i]:\n" +
			"            continue\n" +
			"        cycle = 0\n" +
			"        j = i\n" +
			"        while not visited[j]:\n" +
			"            visited[j] = True\n" +
			"            j = position[nums[j]]\n" +
			"            cycle += 1\n" +
			"        swaps += cycle - 1\n" +
			"    return swaps\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("min_swaps(%s)", pyList(nums)), expected: fmt.Sprintf("%d", minSwapsGo(nums))},
			caseAlias{name: "example-2", call: "min_swaps([3, 2, 1])", expected: "1"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "min_swaps([])", expected: "0"},
			caseAlias{name: "sorted-input", call: "min_swaps([1, 2, 3])", expected: "0"},
			caseAlias{name: "single-element", call: "min_swaps([5])", expected: "0"},
			caseAlias{name: "single-cycle", call: "min_swaps([2, 3, 1])", expected: "2"},
			caseAlias{name: "reverse-sorted", call: "min_swaps([4, 3, 2, 1])", expected: "2"},
			caseAlias{name: "duplicates", call: "min_swaps([1, 1])", expected: "0"},
		)
		stmt := fmt.Sprintf("%s置换仪式：每次可交换任意两个元素，求使列表升序的最少交换次数。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// ---- 期望值计算（与参考解同语义） ----

func maxListGo(nums []int) int {
	m := nums[0]
	for _, v := range nums[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func cheapestGo(nums []int, k int) int {
	cp := sortedGo(nums)
	total := 0
	for i := 0; i < k; i++ {
		total += cp[i]
	}
	return total
}

func canWalkGo(nums []int) bool {
	cp := sortedGo(nums)
	for i := 1; i < len(cp); i++ {
		if cp[i]-cp[i-1] > 1 {
			return false
		}
	}
	return true
}

func rankListGo(scores []int) []int {
	cp := sortedGo(scores)
	ranks := map[int]int{}
	for i, v := range cp {
		ranks[v] = i + 1
	}
	out := make([]int, len(scores))
	for i, s := range scores {
		out[i] = ranks[s]
	}
	return out
}

func maxEventsGo(intervals [][2]int) int {
	if len(intervals) == 0 {
		return 0
	}
	cp := append([][2]int{}, intervals...)
	sort.Slice(cp, func(i, j int) bool { return cp[i][1] < cp[j][1] })
	count := 0
	lastEnd := -1
	for _, it := range cp {
		if it[0] >= lastEnd {
			count++
			lastEnd = it[1]
		}
	}
	return count
}

func topSumGo(hits []int, k int) int {
	cp := sortedGo(hits)
	total := 0
	for i := 0; i < k; i++ {
		total += cp[len(cp)-1-i]
	}
	return total
}

func secondLargestGo(hits []int) int {
	seen := map[int]bool{}
	uniq := []int{}
	for _, v := range hits {
		if !seen[v] {
			seen[v] = true
			uniq = append(uniq, v)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(uniq)))
	if len(uniq) < 2 {
		return -1
	}
	return uniq[1]
}

func clusterHitsGo(hits []int) int {
	if len(hits) == 0 {
		return 0
	}
	cp := sortedGo(hits)
	groups := 1
	last := cp[0]
	for _, h := range cp[1:] {
		if h-last > 2 {
			groups++
		}
		last = h
	}
	return groups
}

func unsortedSpanGo(nums []int) int {
	n := len(nums)
	if n == 0 {
		return 0
	}
	ordered := sortedGo(nums)
	left := 0
	for left < n && nums[left] == ordered[left] {
		left++
	}
	if left == n {
		return 0
	}
	right := n - 1
	for right > left && nums[right] == ordered[right] {
		right--
	}
	return right - left + 1
}

func closePairsGo(nums []int, limit int) int {
	cp := sortedGo(nums)
	count := 0
	for i := 1; i < len(cp); i++ {
		if cp[i]-cp[i-1] <= limit {
			count++
		}
	}
	return count
}

func medalistsGo(scores []int) []int {
	seen := map[int]bool{}
	uniq := []int{}
	for _, s := range scores {
		if !seen[s] {
			seen[s] = true
			uniq = append(uniq, s)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(uniq)))
	if len(uniq) > 3 {
		uniq = uniq[:3]
	}
	return uniq
}

func minSwapsGo(nums []int) int {
	n := len(nums)
	ordered := sortedGo(nums)
	position := map[int]int{}
	for i, v := range ordered {
		position[v] = i
	}
	visited := make([]bool, n)
	swaps := 0
	for i := 0; i < n; i++ {
		if visited[i] || nums[i] == ordered[i] {
			continue
		}
		cycle := 0
		j := i
		for !visited[j] {
			visited[j] = true
			j = position[nums[j]]
			cycle++
		}
		swaps += cycle - 1
	}
	return swaps
}
