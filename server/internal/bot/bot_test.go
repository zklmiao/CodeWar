package bot

import (
	"context"
	"math"
	"math/rand"
	"testing"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/classes"
	"codewar/server/internal/match"
)

// seqRNG：按固定序列返回的随机源（边界采样与确定性断言用）。
type seqRNG struct {
	floats []float64
	ints   []int
	fi, ii int
}

func (r *seqRNG) Float64() float64 {
	v := r.floats[r.fi%len(r.floats)]
	r.fi++
	return v
}

func (r *seqRNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	v := r.ints[r.ii%len(r.ints)]
	r.ii++
	if v < 0 || v >= n {
		return v % n
	}
	return v
}

func TestParseDifficulty(t *testing.T) {
	cases := map[string]Difficulty{
		"easy": Easy, "EASY": Easy, " Normal ": Normal, "hard": Hard,
	}
	for in, want := range cases {
		got, ok := ParseDifficulty(in)
		if !ok || got != want {
			t.Fatalf("ParseDifficulty(%q) = %q,%v，want %q,true", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "bogus", "1"} {
		if got, ok := ParseDifficulty(in); ok {
			t.Fatalf("ParseDifficulty(%q) 应非法，得到 %q", in, got)
		}
	}
	if !(Easy.Label() == "简单" && Normal.Label() == "普通" && Hard.Label() == "困难") {
		t.Fatal("难度展示名不符")
	}
}

// 三档参数必须与 spec/架构裁决的字面值一致（独立于实现硬编码，防止调参
// 时"测试跟着实现一起漂移"）。
func TestParamsForMatchesSpec(t *testing.T) {
	want := map[Difficulty]Params{
		Easy: {MinSolveMs: 45000, MaxSolveMs: 70000, Accuracy: 0.55,
			Weights: Weights{Attack: 60, Move: 15, Defend: 15, Skill: 10}},
		Normal: {MinSolveMs: 25000, MaxSolveMs: 45000, Accuracy: 0.75,
			Weights: Weights{Attack: 70, Move: 10, Defend: 10, Skill: 10}},
		Hard: {MinSolveMs: 12000, MaxSolveMs: 25000, Accuracy: 0.90,
			Weights: Weights{Attack: 75, Move: 5, Defend: 10, Skill: 10}},
	}
	for d, w := range want {
		if got := ParamsFor(d); got != w {
			t.Fatalf("%s 参数与 spec 初稿不一致：got %+v want %+v", d, got, w)
		}
	}
}

// 三档耗时窗边界：Float64=0 → 下界；趋近 1 → 上界（含端点）。
func TestSolveWindowBoundaries(t *testing.T) {
	for _, d := range All {
		p := ParamsFor(d)
		lo := Decide(p, &seqRNG{floats: []float64{0, 0.5}, ints: []int{0}}, true)
		if lo.SolveMs != p.MinSolveMs {
			t.Fatalf("%s 下界应为 %d，实际 %d", d, p.MinSolveMs, lo.SolveMs)
		}
		hi := Decide(p, &seqRNG{floats: []float64{0.999999999999, 0.5}, ints: []int{0}}, true)
		if hi.SolveMs != p.MaxSolveMs {
			t.Fatalf("%s 上界应为 %d，实际 %d", d, p.MaxSolveMs, hi.SolveMs)
		}
	}
}

// 固定 seed 大样本：耗时窗、正确率、行动权重分布（统计区间，非单局）。
func TestParamsStatistics(t *testing.T) {
	const n = 20000
	avg := map[Difficulty]float64{}
	for _, d := range All {
		p := ParamsFor(d)
		r := rand.New(rand.NewSource(20261021))
		var sum int64
		correct, attack, move, defend, skill := 0, 0, 0, 0, 0
		for i := 0; i < n; i++ {
			dec := Decide(p, r, true)
			if dec.SolveMs < p.MinSolveMs || dec.SolveMs > p.MaxSolveMs {
				t.Fatalf("%s solveMs=%d 越出窗 [%d,%d]", d, dec.SolveMs, p.MinSolveMs, p.MaxSolveMs)
			}
			sum += dec.SolveMs
			if dec.Correct {
				correct++
			}
			switch dec.Action {
			case match.ActionAttack:
				attack++
			case match.ActionMove:
				move++
			case match.ActionDefend:
				defend++
			case match.ActionSkill:
				skill++
			default:
				t.Fatalf("未知行动 %q", dec.Action)
			}
		}
		avg[d] = float64(sum) / n
		if got := float64(correct) / n; math.Abs(got-p.Accuracy) > 0.02 {
			t.Fatalf("%s 正确率 %.3f 偏离目标 %.2f 超过 2%%", d, got, p.Accuracy)
		}
		total := float64(p.Weights.Attack + p.Weights.Move + p.Weights.Defend + p.Weights.Skill)
		checks := []struct {
			name string
			got  int
			want int
		}{
			{"attack", attack, p.Weights.Attack},
			{"move", move, p.Weights.Move},
			{"defend", defend, p.Weights.Defend},
			{"skill", skill, p.Weights.Skill},
		}
		for _, c := range checks {
			if got := float64(c.got) / n; math.Abs(got-float64(c.want)/total) > 0.02 {
				t.Fatalf("%s 行动 %s 占比 %.3f，目标 %.3f", d, c.name, got, float64(c.want)/total)
			}
		}
	}
	// 难度可感知：easy 平均耗时 > normal > hard，且 easy ≥ hard 的 2 倍。
	if !(avg[Easy] > avg[Normal] && avg[Normal] > avg[Hard]) {
		t.Fatalf("平均耗时未随难度单调下降：easy=%.0f normal=%.0f hard=%.0f", avg[Easy], avg[Normal], avg[Hard])
	}
	if avg[Easy] < 2*avg[Hard] {
		t.Fatalf("easy 平均耗时（%.0f）应至少为 hard（%.0f）的 2 倍", avg[Easy], avg[Hard])
	}
}

// 技能冷却中：技能权重并入攻击（normal 70+10=80 / 10 / 10）。
func TestSkillCooldownWeightMergesIntoAttack(t *testing.T) {
	const n = 20000
	p := ParamsFor(Normal)
	r := rand.New(rand.NewSource(99))
	attack, move, defend, skill := 0, 0, 0, 0
	for i := 0; i < n; i++ {
		switch Decide(p, r, false).Action {
		case match.ActionAttack:
			attack++
		case match.ActionMove:
			move++
		case match.ActionDefend:
			defend++
		case match.ActionSkill:
			skill++
		}
	}
	if skill != 0 {
		t.Fatalf("冷却中不应选择技能，实际 %d 次", skill)
	}
	if got := float64(attack) / n; math.Abs(got-0.80) > 0.02 {
		t.Fatalf("冷却中攻击占比应约 0.80，实际 %.3f", got)
	}
	if got := float64(move) / n; math.Abs(got-0.10) > 0.02 {
		t.Fatalf("移动占比应约 0.10，实际 %.3f", got)
	}
	if got := float64(defend) / n; math.Abs(got-0.10) > 0.02 {
		t.Fatalf("防御占比应约 0.10，实际 %.3f", got)
	}
}

func TestPlanTargeting(t *testing.T) {
	from := match.Pos{Row: 3, Col: 2}
	cases := []struct {
		name   string
		opp    match.Pos
		shape  string
		rangeM int32
		want   match.Targeting
		ok     bool
	}{
		{"点狙向上距离3", match.Pos{Row: 0, Col: 2}, classes.ShapeSnipe, 3, match.Targeting{Direction: classes.DirUp, Distance: 3}, true},
		{"点狙距离钳到射程", match.Pos{Row: 0, Col: 2}, classes.ShapeSnipe, 2, match.Targeting{Direction: classes.DirUp, Distance: 2}, true},
		{"点狙至少1格", match.Pos{Row: 3, Col: 2}, classes.ShapeSnipe, 3, match.Targeting{Direction: classes.DirRight, Distance: 1}, true},
		{"斜向取主轴（行）", match.Pos{Row: 0, Col: 0}, classes.ShapeSnipe, 3, match.Targeting{Direction: classes.DirUp, Distance: 3}, true},
		{"斜向取主轴（列）", match.Pos{Row: 2, Col: 4}, classes.ShapeSnipe, 3, match.Targeting{Direction: classes.DirRight, Distance: 2}, true},
		{"斜向平局取行轴", match.Pos{Row: 1, Col: 4}, classes.ShapeSnipe, 3, match.Targeting{Direction: classes.DirUp, Distance: 2}, true},
		{"线扫向右", match.Pos{Row: 3, Col: 4}, classes.ShapeLine, 0, match.Targeting{Direction: classes.DirRight}, true},
		{"横带向上", match.Pos{Row: 1, Col: 2}, classes.ShapeBand, 0, match.Targeting{Direction: classes.DirUp}, true},
		{"近战无需目标", match.Pos{Row: 2, Col: 2}, classes.ShapeMelee, 0, match.Targeting{}, false},
		{"邻域无需方向", match.Pos{Row: 2, Col: 2}, classes.ShapeBurst8, 0, match.Targeting{}, false},
		{"治疗自身", match.Pos{Row: 3, Col: 2}, classes.ShapeSelf, 0, match.Targeting{}, false},
	}
	for _, tc := range cases {
		got, ok := PlanTargeting(from, tc.opp, tc.shape, tc.rangeM)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("%s：got %+v,%v want %+v,%v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

func TestLegalMovesMirrorEngineRules(t *testing.T) {
	from := match.Pos{Row: 3, Col: 2}
	opp := match.Pos{Row: 2, Col: 2} // 正上方相邻
	moves := LegalMoves(from, opp, 2)

	has := func(p match.Pos) bool {
		for _, m := range moves {
			if m == p {
				return true
			}
		}
		return false
	}
	// 上方相邻格被对方占据；向上 2 格路径中间格也是对方 → 均非法。
	if has(opp) || has(match.Pos{Row: 1, Col: 2}) {
		t.Fatalf("穿越/占据对方的落点不应出现在合法移动中：%v", moves)
	}
	// 其余三个方向各 2 格（下 1、左 2、右 2；下 2 越界）。
	want := []match.Pos{{Row: 4, Col: 2}, {Row: 3, Col: 1}, {Row: 3, Col: 0}, {Row: 3, Col: 3}, {Row: 3, Col: 4}}
	if len(moves) != len(want) {
		t.Fatalf("合法移动数 %d，want %d：%v", len(moves), len(want), moves)
	}
	for _, w := range want {
		if !has(w) {
			t.Fatalf("缺少合法落点 %v：%v", w, moves)
		}
	}
}

// 规划器的"合法"必须被引擎接受、"非法"必须被引擎拒绝：纯函数测试只证明
// 副本内部自洽，这里用引擎 seam 钉住副本与 validateMove 的同语义（漂移即红）。
type passAllJudge struct{}

func (passAllJudge) Judge(context.Context, string, []match.TestCase) match.JudgeResult {
	return match.JudgeResult{Passed: true, Category: match.CategoryOK}
}

func submitMoveOnce(t *testing.T, to match.Pos) []*matchpb.Envelope {
	t.Helper()
	var events []*matchpb.Envelope
	eng := match.NewEngine(match.RealClock{}, match.StubQuestions{}, passAllJudge{}, func(env *matchpb.Envelope) {
		events = append(events, env)
	})
	eng.StartMatch()
	eng.Submit(match.P1, match.ActionMove, "code", to)
	eng.Stop()
	return events
}

func TestLegalMovesAcceptedByEngine(t *testing.T) {
	from := match.Pos{Row: 3, Col: 2}
	opp := match.Pos{Row: 2, Col: 2}

	for _, to := range LegalMoves(from, opp, 2) {
		for _, env := range submitMoveOnce(t, to) {
			if r := env.GetSubmissionRejected(); r != nil {
				t.Fatalf("规划为合法的移动 %v 被引擎拒绝：%s", to, r.GetReason())
			}
		}
	}

	illegal := []struct {
		to     match.Pos
		reason string
	}{
		{match.Pos{Row: 2, Col: 1}, "diagonal"},
		{match.Pos{Row: 5, Col: 2}, "out-of-bounds"},
		{match.Pos{Row: 2, Col: 2}, "dest-occupied"},
		{match.Pos{Row: 1, Col: 2}, "path-blocked"},
	}
	for _, tc := range illegal {
		found := false
		for _, env := range submitMoveOnce(t, tc.to) {
			if r := env.GetSubmissionRejected(); r != nil {
				found = true
				if r.GetReason() != tc.reason {
					t.Fatalf("非法移动 %v 拒绝原因 %q，want %q", tc.to, r.GetReason(), tc.reason)
				}
			}
		}
		if !found {
			t.Fatalf("非法移动 %v 未被引擎拒绝", tc.to)
		}
	}
}

func TestLoadoutDeterministicAndValid(t *testing.T) {
	a1, s1 := Loadout("seed-1")
	a2, s2 := Loadout("seed-1")
	if a1 != a2 || s1 != s2 {
		t.Fatalf("同 seed 职业/技能应一致：%s/%s vs %s/%s", a1, s1, a2, s2)
	}
	if _, ok := classes.ByID(a1); !ok {
		t.Fatalf("Loadout 返回未知职业 %q", a1)
	}
	if err := classes.ValidateLoadout(a1, s1); err != nil {
		t.Fatalf("Loadout 技能不属于职业：%v", err)
	}
	seen := map[string]bool{}
	for i := 0; i < 64; i++ {
		id, _ := Loadout("round-robin-" + string(rune('a'+i%26)) + string(rune('a'+i/26)))
		seen[id] = true
	}
	if len(seen) < 3 {
		t.Fatalf("64 个 seed 应覆盖三个职业，实际 %v", seen)
	}
}
