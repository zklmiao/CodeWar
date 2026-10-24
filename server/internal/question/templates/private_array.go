// 私有行动题·数组与列表主题（12 号票批 4：move/defend/skill × L1/L3/L4/L5）。
// 形式均为 complete；题型标注进入矩阵格（coverage_test.go 口径）。
package templates

import (
	"fmt"

	"codewar/server/internal/question"
)

// 移动 L1·数组：最省步数（单语法点：min，参考解 1 行；约定非空）。
func moveNearestL1() question.Template {
	meta := question.Meta{ID: "move-nearest-L1-move", Topic: question.TopicArray,
		Difficulty: question.L1, Action: question.ActionMove, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		steps := rng.IntList(rng.Int(2, 5), 0, 6, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def min_steps(steps: list) -> int:"
		answer := "def min_steps(steps: list) -> int:\n    return min(steps)\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("min_steps(%s)", pyList(steps)), expected: fmt.Sprintf("%d", minListGo(steps))},
			caseAlias{name: "example-2", call: "min_steps([4, 2, 6])", expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "single", call: "min_steps([7])", expected: "7"},
			caseAlias{name: "all-same", call: "min_steps([3, 3, 3])", expected: "3"},
			caseAlias{name: "with-zero", call: "min_steps([0, 5])", expected: "0"},
			caseAlias{name: "mixed", call: "min_steps([9, 1, 4])", expected: "1"},
			caseAlias{name: "last", call: "min_steps([5, 4, 2])", expected: "2"},
		)
		stmt := fmt.Sprintf("%s最省步数：返回可选步数中的最小值（约定列表非空）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L3·数组：限界行走（累积 + 双向截断，参考解 8 行）。
func moveClampL3() question.Template {
	meta := question.Meta{ID: "move-clamp-L3-move", Topic: question.TopicArray,
		Difficulty: question.L3, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "clamp-high", "clamp-low"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		steps := rng.IntList(rng.Int(2, 5), -4, 6, question.ListOpts{})
		limit := rng.Int(3, 6)
		kind := rng.Scenario("arena")
		sig := "def walk_clamped(steps: list, limit: int) -> int:"
		answer := "def walk_clamped(steps: list, limit: int) -> int:\n    pos = 0\n    for s in steps:\n        pos += s\n        if pos > limit:\n            pos = limit\n        if pos < -limit:\n            pos = -limit\n    return pos\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("walk_clamped(%s, %d)", pyList(steps), limit), expected: fmt.Sprintf("%d", clampWalkGo(steps, limit))},
			caseAlias{name: "example-2", call: "walk_clamped([3, 3], 5)", expected: "5"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "walk_clamped([], 5)", expected: "0"},
			caseAlias{name: "clamp-high", call: "walk_clamped([9], 4)", expected: "4"},
			caseAlias{name: "clamp-low", call: "walk_clamped([-9], 4)", expected: "-4"},
			caseAlias{name: "exact-limit", call: "walk_clamped([4], 4)", expected: "4"},
			caseAlias{name: "mixed", call: "walk_clamped([2, -1, 2], 3)", expected: "3"},
		)
		stmt := fmt.Sprintf("%s限界行走：从 0 出发逐步累加，每步后坐标截断在 [−limit, limit] 内，返回最终坐标。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L4·数组：跳跃可达（贪心维护最远可达 + 边界，参考解 8 行；12 号票批 4，
// 算法内核参考 codetop59 跳跃游戏，题面经情境池重包装）。
func moveJumpL4() question.Template {
	meta := question.Meta{ID: "move-jump-L4-move", Topic: question.TopicArray,
		Difficulty: question.L4, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"empty-input", "stuck-zero", "single-element"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		jumps := rng.IntList(rng.Int(3, 6), 0, 4, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def can_reach(jumps: list) -> bool:"
		answer := "def can_reach(jumps: list) -> bool:\n    reach = 0\n    for i in range(len(jumps)):\n        if i > reach:\n            return False\n        land = i + jumps[i]\n        if land > reach:\n            reach = land\n    return True\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("can_reach(%s)", pyList(jumps)), expected: pyBool(canReachGo(jumps))},
			caseAlias{name: "example-2", call: "can_reach([2, 0, 0])", expected: "True"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "can_reach([])", expected: "True"},
			caseAlias{name: "single-zero", call: "can_reach([0])", expected: "True"},
			caseAlias{name: "stuck-zero", call: "can_reach([0, 1])", expected: "False"},
			caseAlias{name: "single-element", call: "can_reach([3])", expected: "True"},
			caseAlias{name: "classic-fail", call: "can_reach([3, 2, 1, 0, 4])", expected: "False"},
			caseAlias{name: "just-enough", call: "can_reach([1, 1, 0])", expected: "True"},
		)
		stmt := fmt.Sprintf("%s跳跃可达：每格给出跳跃力 jumps[i]，从下标 0 出发能否到达最后一格（空列表视为可达）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 移动 L5·数组：带障碍路径计数（二维 DP + 障碍分支，参考解 14 行）。
func movePathsL5() question.Template {
	meta := question.Meta{ID: "move-paths-L5-move", Topic: question.TopicArray,
		Difficulty: question.L5, Action: question.ActionMove, Form: question.FormComplete,
		Traps: []string{"zero-input", "blocked-start", "blocked-middle", "single-row"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		rows, cols := rng.Int(2, 4), rng.Int(2, 4)
		blocked := [][2]int{}
		for i := 0; i < rng.Int(0, 2); i++ {
			blocked = append(blocked, [2]int{rng.Int(0, rows-1), rng.Int(0, cols-1)})
		}
		kind := rng.Scenario("dungeon")
		sig := "def unique_paths(rows: int, cols: int, blocked: list) -> int:"
		answer := "def unique_paths(rows: int, cols: int, blocked: list) -> int:\n" +
			"    if rows <= 0 or cols <= 0:\n" +
			"        return 0\n" +
			"    grid = [[0] * cols for _ in range(rows)]\n" +
			"    for r in range(rows):\n" +
			"        for c in range(cols):\n" +
			"            if [r, c] in blocked:\n" +
			"                grid[r][c] = 0\n" +
			"            elif r == 0 and c == 0:\n" +
			"                grid[r][c] = 1\n" +
			"            else:\n" +
			"                up = grid[r - 1][c] if r > 0 else 0\n" +
			"                left = grid[r][c - 1] if c > 0 else 0\n" +
			"                grid[r][c] = up + left\n" +
			"    return grid[rows - 1][cols - 1]\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("unique_paths(%d, %d, %s)", rows, cols, pyRanges(blocked)), expected: fmt.Sprintf("%d", uniquePathsGo(rows, cols, blocked))},
			caseAlias{name: "example-2", call: "unique_paths(3, 3, [])", expected: "6"},
		)
		hidden := cases(
			caseAlias{name: "zero-input", call: "unique_paths(0, 5, [])", expected: "0"},
			caseAlias{name: "single-row", call: "unique_paths(1, 5, [])", expected: "1"},
			caseAlias{name: "blocked-start", call: "unique_paths(3, 3, [[0, 0]])", expected: "0"},
			caseAlias{name: "blocked-middle", call: "unique_paths(3, 3, [[1, 1]])", expected: fmt.Sprintf("%d", uniquePathsGo(3, 3, [][2]int{{1, 1}}))},
			caseAlias{name: "single-row-blocked", call: "unique_paths(1, 5, [[0, 2]])", expected: "0"},
		)
		stmt := fmt.Sprintf("%s路径计数：rows×cols 网格从左上到右下，每步只能向右或向下；blocked 为障碍格列表（不可通行），返回不同路径数（行列非正为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L1·数组：最轻一击（单语法点：min，参考解 1 行；约定非空）。
func defendLowestL1() question.Template {
	meta := question.Meta{ID: "defend-lowest-L1-defend", Topic: question.TopicArray,
		Difficulty: question.L1, Action: question.ActionDefend, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		hits := rng.IntList(rng.Int(2, 5), 1, 12, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def lightest_hit(hits: list) -> int:"
		answer := "def lightest_hit(hits: list) -> int:\n    return min(hits)\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("lightest_hit(%s)", pyList(hits)), expected: fmt.Sprintf("%d", minListGo(hits))},
			caseAlias{name: "example-2", call: "lightest_hit([8, 3, 5])", expected: "3"},
		)
		hidden := cases(
			caseAlias{name: "single", call: "lightest_hit([9])", expected: "9"},
			caseAlias{name: "all-same", call: "lightest_hit([4, 4])", expected: "4"},
			caseAlias{name: "first", call: "lightest_hit([1, 7])", expected: "1"},
			caseAlias{name: "last", call: "lightest_hit([9, 2])", expected: "2"},
			caseAlias{name: "middling", call: "lightest_hit([6, 2, 8])", expected: "2"},
		)
		stmt := fmt.Sprintf("%s最轻一击：返回命中列表中的最小伤害值（约定非空）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L3·数组：重击计数（循环 + 严格阈值，参考解 5 行）。
func defendCountL3() question.Template {
	meta := question.Meta{ID: "defend-count-L3-defend", Topic: question.TopicArray,
		Difficulty: question.L3, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "no-match", "all-match"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		hits := rng.IntList(rng.Int(3, 6), 1, 12, question.ListOpts{})
		limit := rng.Int(3, 9)
		kind := rng.Scenario("arena")
		sig := "def heavy_hits(hits: list, limit: int) -> int:"
		answer := "def heavy_hits(hits: list, limit: int) -> int:\n    count = 0\n    for h in hits:\n        if h > limit:\n            count += 1\n    return count\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("heavy_hits(%s, %d)", pyList(hits), limit), expected: fmt.Sprintf("%d", heavyHitsGo(hits, limit))},
			caseAlias{name: "example-2", call: "heavy_hits([1, 7, 12], 5)", expected: "2"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "heavy_hits([], 5)", expected: "0"},
			caseAlias{name: "no-match", call: "heavy_hits([1, 2], 5)", expected: "0"},
			caseAlias{name: "all-match", call: "heavy_hits([9, 9], 5)", expected: "2"},
			caseAlias{name: "boundary-not-counted", call: "heavy_hits([5], 5)", expected: "0"},
			caseAlias{name: "threshold-low", call: "heavy_hits([3, 4], 3)", expected: "1"},
		)
		stmt := fmt.Sprintf("%s重击统计：统计严格大于阈值 limit 的命中次数（等于阈值不算）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L4·数组：伤害分桶（多分支 + 边界，参考解 9 行）。
func defendBucketL4() question.Template {
	meta := question.Meta{ID: "defend-bucket-L4-defend", Topic: question.TopicArray,
		Difficulty: question.L4, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "boundary-five", "boundary-ten"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		hits := rng.IntList(rng.Int(3, 6), 1, 14, question.ListOpts{})
		kind := rng.Scenario("arena")
		sig := "def bucket_hits(hits: list) -> list:"
		answer := "def bucket_hits(hits: list) -> list:\n    buckets = [0, 0, 0]\n    for h in hits:\n        if h <= 5:\n            buckets[0] += 1\n        elif h <= 10:\n            buckets[1] += 1\n        else:\n            buckets[2] += 1\n    return buckets\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("bucket_hits(%s)", pyList(hits)), expected: pyList(bucketHitsGo(hits))},
			caseAlias{name: "example-2", call: "bucket_hits([3, 7, 12])", expected: "[1, 1, 1]"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "bucket_hits([])", expected: "[0, 0, 0]"},
			caseAlias{name: "boundary-five", call: "bucket_hits([5])", expected: "[1, 0, 0]"},
			caseAlias{name: "boundary-ten", call: "bucket_hits([10])", expected: "[0, 1, 0]"},
			caseAlias{name: "above-ten", call: "bucket_hits([11])", expected: "[0, 0, 1]"},
			caseAlias{name: "all-high", call: "bucket_hits([12, 14])", expected: "[0, 0, 2]"},
		)
		stmt := fmt.Sprintf("%s伤害分桶：把命中值分为三桶计数 [≤5, 6-10, ≥11]，返回三元列表。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 防御 L5·数组：定长窗口最大和（窗口滑动 + 全负边界，参考解 14 行；12 号票批 4，
// 算法内核参考 codetop59 长度最小的子数组/滑动窗口，题面经情境池重包装）。
func defendWindowL5() question.Template {
	meta := question.Meta{ID: "defend-window-L5-defend", Topic: question.TopicArray,
		Difficulty: question.L5, Action: question.ActionDefend, Form: question.FormComplete,
		Traps: []string{"empty-input", "k-larger-than-n", "all-negative", "k-equals-n"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(4, 7), -6, 9, question.ListOpts{})
		k := rng.Int(1, 5)
		kind := rng.Scenario("arena")
		sig := "def max_window_sum(nums: list, k: int) -> int:"
		answer := "def max_window_sum(nums: list, k: int) -> int:\n" +
			"    n = len(nums)\n" +
			"    if n == 0 or k <= 0:\n" +
			"        return 0\n" +
			"    if k > n:\n" +
			"        k = n\n" +
			"    current = 0\n" +
			"    for i in range(k):\n" +
			"        current += nums[i]\n" +
			"    best = current\n" +
			"    for i in range(k, n):\n" +
			"        current = current - nums[i - k] + nums[i]\n" +
			"        if current > best:\n" +
			"            best = current\n" +
			"    return best\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("max_window_sum(%s, %d)", pyList(nums), k), expected: fmt.Sprintf("%d", maxWindowSumGo(nums, k))},
			caseAlias{name: "example-2", call: "max_window_sum([2, 1, 5, 1, 3, 2], 3)", expected: "9"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "max_window_sum([], 2)", expected: "0"},
			caseAlias{name: "k-larger-than-n", call: "max_window_sum([1, 2, 3], 5)", expected: "6"},
			caseAlias{name: "all-negative", call: "max_window_sum([-3, -1, -2], 1)", expected: "-1"},
			caseAlias{name: "k-equals-n", call: "max_window_sum([1, 2, 3], 3)", expected: "6"},
			caseAlias{name: "single", call: "max_window_sum([0], 1)", expected: "0"},
		)
		stmt := fmt.Sprintf("%s窗口承伤：求长度为 k 的连续窗口最大和（k 超过长度时取全数组；空数组为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L1·数组：宝珠点数（单语法点：len，参考解 1 行）。
func skillOrbL1() question.Template {
	meta := question.Meta{ID: "skill-orb-L1-skill", Topic: question.TopicArray,
		Difficulty: question.L1, Action: question.ActionSkill, Form: question.FormComplete}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		orbs := rng.IntList(rng.Int(1, 5), 1, 6, question.ListOpts{})
		kind := rng.Scenario("spell")
		sig := "def orb_count(orbs: list) -> int:"
		answer := "def orb_count(orbs: list) -> int:\n    return len(orbs)\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("orb_count(%s)", pyList(orbs)), expected: fmt.Sprintf("%d", len(orbs))},
			caseAlias{name: "example-2", call: "orb_count([1, 2, 3])", expected: "3"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "orb_count([])", expected: "0"},
			caseAlias{name: "single", call: "orb_count([9])", expected: "1"},
			caseAlias{name: "duplicates", call: "orb_count([2, 2, 2])", expected: "3"},
			caseAlias{name: "large", call: "orb_count([1, 2, 3, 4, 5, 6, 7])", expected: "7"},
			caseAlias{name: "zeros", call: "orb_count([0, 0])", expected: "2"},
		)
		stmt := fmt.Sprintf("%s宝珠点数：返回宝珠列表的元素个数（空列表为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L3·数组：充能合成（求和 + 整除，参考解 5 行）。
func skillChargeL3() question.Template {
	meta := question.Meta{ID: "skill-charge-L3-skill", Topic: question.TopicArray,
		Difficulty: question.L3, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"empty-input", "below-threshold"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		shards := rng.IntList(rng.Int(2, 5), 0, 5, question.ListOpts{})
		kind := rng.Scenario("spell")
		sig := "def craft_charges(shards: list) -> int:"
		answer := "def craft_charges(shards: list) -> int:\n    total = 0\n    for s in shards:\n        total += s\n    charges = total // 3\n    return charges\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("craft_charges(%s)", pyList(shards)), expected: fmt.Sprintf("%d", craftChargesGo(shards))},
			caseAlias{name: "example-2", call: "craft_charges([1, 2])", expected: "1"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "craft_charges([])", expected: "0"},
			caseAlias{name: "below-threshold", call: "craft_charges([1])", expected: "0"},
			caseAlias{name: "exact-multiple", call: "craft_charges([3])", expected: "1"},
			caseAlias{name: "remainder", call: "craft_charges([2, 2])", expected: "1"},
			caseAlias{name: "batch", call: "craft_charges([9, 0])", expected: "3"},
		)
		stmt := fmt.Sprintf("%s充能合成：每 3 个碎片合成 1 点充能，返回可合成点数（向下取整）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L4·数组：成对合并（双指针扫描 + 跳过成对项，参考解 12 行）。
func skillMergeL4() question.Template {
	meta := question.Meta{ID: "skill-merge-L4-skill", Topic: question.TopicArray,
		Difficulty: question.L4, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-element", "no-adjacent"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		nums := rng.IntList(rng.Int(2, 6), 1, 4, question.ListOpts{})
		kind := rng.Scenario("spell")
		sig := "def merge_adjacent(nums: list) -> list:"
		answer := "def merge_adjacent(nums: list) -> list:\n" +
			"    if not nums:\n" +
			"        return []\n" +
			"    merged = []\n" +
			"    i = 0\n" +
			"    while i < len(nums):\n" +
			"        if i + 1 < len(nums) and nums[i] == nums[i + 1]:\n" +
			"            merged.append(nums[i] + nums[i + 1])\n" +
			"            i += 2\n" +
			"        else:\n" +
			"            merged.append(nums[i])\n" +
			"            i += 1\n" +
			"    return merged\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("merge_adjacent(%s)", pyList(nums)), expected: pyList(mergeAdjacentGo(nums))},
			caseAlias{name: "example-2", call: "merge_adjacent([2, 2, 2, 3])", expected: "[4, 2, 3]"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "merge_adjacent([])", expected: "[]"},
			caseAlias{name: "single-element", call: "merge_adjacent([5])", expected: "[5]"},
			caseAlias{name: "no-adjacent", call: "merge_adjacent([1, 2, 3])", expected: "[1, 2, 3]"},
			caseAlias{name: "one-pair", call: "merge_adjacent([4, 4])", expected: "[8]"},
			caseAlias{name: "two-runs", call: "merge_adjacent([1, 1, 2, 2])", expected: "[2, 4]"},
			caseAlias{name: "tail-run", call: "merge_adjacent([1, 3, 3, 3])", expected: "[1, 6, 3]"},
		)
		stmt := fmt.Sprintf("%s熔炼合并：从左到右扫描，相邻两数相等则合并为两数之和并跳过这两项，否则保留当前项（保持顺序）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// 技能 L5·数组：打家劫舍（DP 双状态滚动 + 边界，参考解 12 行；12 号票批 4，
// 算法内核参考 codetop59 打家劫舍，题面经情境池重包装）。
func skillRobL5() question.Template {
	meta := question.Meta{ID: "skill-rob-L5-skill", Topic: question.TopicArray,
		Difficulty: question.L5, Action: question.ActionSkill, Form: question.FormComplete,
		Traps: []string{"empty-input", "single-house", "two-houses", "all-zero"}}
	return funcTemplate{meta: meta, gen: func(rng *question.RNG) (question.Instance, error) {
		houses := rng.IntList(rng.Int(2, 6), 0, 9, question.ListOpts{})
		kind := rng.Scenario("spell")
		sig := "def rob(houses: list) -> int:"
		answer := "def rob(houses: list) -> int:\n" +
			"    n = len(houses)\n" +
			"    if n == 0:\n" +
			"        return 0\n" +
			"    if n == 1:\n" +
			"        return houses[0]\n" +
			"    prev2 = houses[0]\n" +
			"    prev1 = max(houses[0], houses[1])\n" +
			"    for i in range(2, n):\n" +
			"        current = max(prev1, prev2 + houses[i])\n" +
			"        prev2 = prev1\n" +
			"        prev1 = current\n" +
			"    return prev1\n"
		visible := cases(
			caseAlias{name: "example-1", call: fmt.Sprintf("rob(%s)", pyList(houses)), expected: fmt.Sprintf("%d", robGo(houses))},
			caseAlias{name: "example-2", call: "rob([2, 7, 9, 3, 1])", expected: "12"},
		)
		hidden := cases(
			caseAlias{name: "empty-input", call: "rob([])", expected: "0"},
			caseAlias{name: "single-house", call: "rob([5])", expected: "5"},
			caseAlias{name: "two-houses", call: "rob([2, 7])", expected: "7"},
			caseAlias{name: "all-zero", call: "rob([0, 0])", expected: "0"},
			caseAlias{name: "classic", call: "rob([1, 2, 3, 1])", expected: "4"},
			caseAlias{name: "skip-middle", call: "rob([1, 5, 1])", expected: "5"},
		)
		stmt := fmt.Sprintf("%s能量掠夺：从房屋列表中取数，相邻房屋不可同时选取，求最大总和（空列表为 0）。\n%s", kind, examplesText(visible))
		return bareInstance(meta, stmt, sig, answer, visible, hidden)
	}}
}

// ---- 期望值计算（与参考解同语义） ----

func minListGo(nums []int) int {
	m := nums[0]
	for _, v := range nums[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func clampWalkGo(steps []int, limit int) int {
	pos := 0
	for _, s := range steps {
		pos += s
		if pos > limit {
			pos = limit
		}
		if pos < -limit {
			pos = -limit
		}
	}
	return pos
}

func canReachGo(jumps []int) bool {
	reach := 0
	for i := range jumps {
		if i > reach {
			return false
		}
		if i+jumps[i] > reach {
			reach = i + jumps[i]
		}
	}
	return true
}

func uniquePathsGo(rows, cols int, blocked [][2]int) int {
	if rows <= 0 || cols <= 0 {
		return 0
	}
	blockedSet := map[[2]int]bool{}
	for _, b := range blocked {
		blockedSet[b] = true
	}
	grid := make([][]int, rows)
	for r := range grid {
		grid[r] = make([]int, cols)
	}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if blockedSet[[2]int{r, c}] {
				grid[r][c] = 0
			} else if r == 0 && c == 0 {
				grid[r][c] = 1
			} else {
				up, left := 0, 0
				if r > 0 {
					up = grid[r-1][c]
				}
				if c > 0 {
					left = grid[r][c-1]
				}
				grid[r][c] = up + left
			}
		}
	}
	return grid[rows-1][cols-1]
}

func heavyHitsGo(hits []int, limit int) int {
	n := 0
	for _, h := range hits {
		if h > limit {
			n++
		}
	}
	return n
}

func bucketHitsGo(hits []int) []int {
	out := []int{0, 0, 0}
	for _, h := range hits {
		if h <= 5 {
			out[0]++
		} else if h <= 10 {
			out[1]++
		} else {
			out[2]++
		}
	}
	return out
}

func maxWindowSumGo(nums []int, k int) int {
	n := len(nums)
	if n == 0 || k <= 0 {
		return 0
	}
	if k > n {
		k = n
	}
	cur := 0
	for i := 0; i < k; i++ {
		cur += nums[i]
	}
	best := cur
	for i := k; i < n; i++ {
		cur += nums[i] - nums[i-k]
		if cur > best {
			best = cur
		}
	}
	return best
}

func craftChargesGo(shards []int) int {
	total := 0
	for _, s := range shards {
		total += s
	}
	return total / 3
}

func mergeAdjacentGo(nums []int) []int {
	if len(nums) == 0 {
		return []int{}
	}
	out := []int{}
	for i := 0; i < len(nums); {
		if i+1 < len(nums) && nums[i] == nums[i+1] {
			out = append(out, nums[i]+nums[i+1])
			i += 2
		} else {
			out = append(out, nums[i])
			i++
		}
	}
	return out
}

func robGo(houses []int) int {
	n := len(houses)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return houses[0]
	}
	prev2 := houses[0]
	prev1 := houses[0]
	if houses[1] > prev1 {
		prev1 = houses[1]
	}
	for i := 2; i < n; i++ {
		current := prev1
		if prev2+houses[i] > current {
			current = prev2 + houses[i]
		}
		prev2, prev1 = prev1, current
	}
	return prev1
}
