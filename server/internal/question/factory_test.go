// Factory seam 测试（spec「测试决策」：只经 Create / Attack-Move-DefendQuestion 公共边界断言）。
package question

import (
	"fmt"
	"strings"
	"testing"

	"codewar/server/internal/match"
)

// stubTemplate：测试用微型模板（固定 meta + 确定性生成）。
type stubTemplate struct {
	meta Meta
	body string
}

func (s stubTemplate) Meta() Meta { return s.meta }

func (s stubTemplate) Generate(rng *RNG) (Instance, error) {
	v := rng.Int(0, 9999)
	tc := match.TestCase{Name: "t", Call: fmt.Sprintf("f(%d)", v), Expected: fmt.Sprintf("%d", v)}
	vis := []match.TestCase{{Name: "v1", Call: "f(0)", Expected: "0"}, {Name: "v2", Call: "f(1)", Expected: "1"}}
	hidden := []match.TestCase{tc, {Name: "h2", Call: "f(2)", Expected: "2"}, {Name: "h3", Call: "f(3)", Expected: "3"},
		{Name: "h4", Call: "f(4)", Expected: "4"}, {Name: "h5", Call: "f(5)", Expected: "5"}}
	// 参考解行数随档位（Rubric S 维：L1:1-3 / L2:3-5 / L3:5-8）。
	want := map[Difficulty]int{L1: 2, L2: 4, L3: 6}[s.meta.Difficulty]
	var b strings.Builder
	b.WriteString("def f(x: int) -> int:\n")
	b.WriteString("    y = x\n")
	for i := 1; i < want-1; i++ {
		fmt.Fprintf(&b, "    y = y + %d\n", 0)
	}
	b.WriteString("    return y\n")
	return Instance{
		Meta: s.meta, Statement: fmt.Sprintf("题 %d", v), Signature: "def f(x: int) -> int:",
		CompleteStarter: RenderComplete("def f(x: int) -> int:"),
		VisibleTests:    vis, HiddenTests: hidden,
		AnswerKey: b.String(),
	}, nil
}

func testFactory() *Factory {
	f := NewFactory(Config{MaxDifficulty: L3, Strategy: StrategyUniform, MatchSeed: "test-seed"})
	// 每个动作 × 每档难度各一个桩，保证均匀抽取必命中（避免回退路径干扰断言）。
	for _, d := range []Difficulty{L1, L2, L3} {
		f.Register(stubTemplate{meta: Meta{ID: fmt.Sprintf("t%d", int(d)), Topic: TopicSyntax, Difficulty: d, Action: ActionAttack, Form: FormComplete}})
		f.Register(stubTemplate{meta: Meta{ID: fmt.Sprintf("m%d", int(d)), Topic: TopicSyntax, Difficulty: d, Action: ActionMove, Form: FormComplete}})
		f.Register(stubTemplate{meta: Meta{ID: fmt.Sprintf("d%d", int(d)), Topic: TopicSyntax, Difficulty: d, Action: ActionDefend, Form: FormComplete}})
	}
	// 多一个跨题型攻击模板，覆盖多候选种子选用分支。
	f.Register(stubTemplate{meta: Meta{ID: "tX", Topic: TopicArray, Difficulty: L3, Action: ActionAttack, Form: FormComplete}})
	return f
}

// 同 spec+seed 必同实例（确定性）。
func TestCreateDeterministic(t *testing.T) {
	f := testFactory()
	spec := Spec{Topic: TopicSyntax, Difficulty: L2, Action: ActionAttack}
	a, err := f.Create(spec, "seed-1")
	if err != nil {
		t.Fatalf("Create 失败：%v", err)
	}
	b, err := f.Create(spec, "seed-1")
	if err != nil {
		t.Fatalf("Create 失败：%v", err)
	}
	if !instancesEqual(a, b) {
		t.Fatal("同 seed 两次生成实例不一致")
	}
}

// 不同 seed 实例不同（参数随机化，0-9999 范围碰撞概率可忽略）。
func TestCreateDifferentSeedsDiffer(t *testing.T) {
	f := testFactory()
	spec := Spec{Topic: TopicSyntax, Difficulty: L2, Action: ActionAttack}
	seen := map[string]bool{}
	for i := 0; i < 8; i++ {
		in, err := f.Create(spec, fmt.Sprintf("seed-%d", i))
		if err != nil {
			t.Fatalf("Create 失败：%v", err)
		}
		seen[in.Statement] = true
	}
	if len(seen) < 5 {
		t.Fatalf("8 个 seed 仅产出 %d 种实例，随机化退化", len(seen))
	}
}

// 错误路径优先：未知题型 / 空注册表。
func TestCreateNoMatchError(t *testing.T) {
	f := testFactory()
	if _, err := f.Create(Spec{Topic: TopicHash, Difficulty: L2, Action: ActionAttack}, "s"); err == nil {
		t.Fatal("无匹配模板应返回错误")
	}
	if _, err := f.Create(Spec{Action: "unknown-action"}, "s"); err == nil {
		t.Fatal("未知动作应返回错误")
	}
	empty := NewFactory(Config{MatchSeed: "x"})
	if _, err := empty.Create(Spec{Action: ActionAttack}, "s"); err == nil {
		t.Fatal("空注册表应返回错误")
	}
}

// 均匀策略：难度 ∈ [L1, 上限]。
func TestUniformDifficultyInRange(t *testing.T) {
	for round := 1; round <= 20; round++ {
		rng := NewRNG(DeriveSeed("u", fmt.Sprintf("attack:round=%d", round)))
		d := pickDifficulty(rng, round, L2, StrategyUniform)
		if d < L1 || d > L2 {
			t.Fatalf("回合 %d 难度 %v 超出 [L1,L2]", round, d)
		}
	}
}

// 爬坡策略：开局 L1、单调不降、终到上限。
func TestClimbMonotonicToMax(t *testing.T) {
	rng := NewRNG(42)
	var prev Difficulty
	for round := 1; round <= 10; round++ {
		d := pickDifficulty(rng, round, L4, StrategyClimb)
		if round == 1 && d != L1 {
			t.Fatalf("爬坡开局应为 L1，实际 %v", d)
		}
		if d < prev {
			t.Fatalf("回合 %d 难度回退：%v < %v", round, d, prev)
		}
		prev = d
	}
	if prev != L4 {
		t.Fatalf("第 10 回合应逼近上限 L4，实际 %v", prev)
	}
}

// 均匀策略统计：L1~L3 各约 1/3（300 抽容差 ±10%）。
func TestUniformRoughlyEven(t *testing.T) {
	counts := map[Difficulty]int{}
	for i := 0; i < 300; i++ {
		rng := NewRNG(DeriveSeed("dist", fmt.Sprintf("n=%d", i)))
		counts[pickDifficulty(rng, 1, L3, StrategyUniform)]++
	}
	for d := L1; d <= L3; d++ {
		if counts[d] < 70 || counts[d] > 130 {
			t.Fatalf("均匀策略分布不均：%v", counts)
		}
	}
}

// 种子派生稳定且区分维度。
func TestDeriveSeedStable(t *testing.T) {
	a := DeriveSeed("m", "attack:round=1")
	b := DeriveSeed("m", "attack:round=1")
	c := DeriveSeed("m", "attack:round=2")
	d := DeriveSeed("other", "attack:round=1")
	if a != b {
		t.Fatal("同输入派生种子不一致")
	}
	if a == c || a == d {
		t.Fatal("不同维度/对局种子应区分")
	}
}

// 镜像题：同回合同实例（双方一致）。
func TestAttackMirrorSameInstance(t *testing.T) {
	f := testFactory()
	q1 := f.AttackQuestion(1)
	q2 := f.AttackQuestion(1)
	if q1.ID == "error-no-template" || q1.Statement == "" {
		t.Fatalf("AttackQuestion 失败：%v", q1.Statement)
	}
	if q1.ID != q2.ID || q1.Statement != q2.Statement {
		t.Fatal("同回合镜像题应完全一致")
	}
}

// 私有题：双方 seed 不同 → 实例不同（高概率；8 回合至少多数不同）。
func TestPrivateQuestionsDifferByPlayer(t *testing.T) {
	f := testFactory()
	diff := 0
	for round := 1; round <= 8; round++ {
		if f.MoveQuestion(round, match.P1).Statement != f.MoveQuestion(round, match.P2).Statement {
			diff++
		}
	}
	if diff < 5 {
		t.Fatalf("8 回合仅 %d 回合双方移动题不同，私有 seed 未生效", diff)
	}
}

// 连打两局：不同 MatchSeed 首回合实例不同（高概率；4 种子对至少 3 对不同）。
func TestTwoMatchesDiffer(t *testing.T) {
	mk := func(seed string) *Factory {
		f := NewFactory(Config{MaxDifficulty: L3, Strategy: StrategyUniform, MatchSeed: seed})
		for _, d := range []Difficulty{L1, L2, L3} {
			f.Register(stubTemplate{meta: Meta{ID: fmt.Sprintf("t%d", int(d)), Topic: TopicSyntax, Difficulty: d, Action: ActionAttack, Form: FormComplete}})
		}
		f.Register(stubTemplate{meta: Meta{ID: "tX", Topic: TopicArray, Difficulty: L3, Action: ActionAttack, Form: FormComplete}})
		return f
	}
	diff := 0
	for i := 0; i < 4; i++ {
		a := mk(fmt.Sprintf("match-%d", i)).AttackQuestion(1)
		b := mk(fmt.Sprintf("match-%d-other", i)).AttackQuestion(1)
		if a.ID != b.ID || a.Statement != b.Statement {
			diff++
		}
	}
	if diff < 3 {
		t.Fatalf("4 对局仅 %d 对首题不同，MatchSeed 未生效", diff)
	}
}
