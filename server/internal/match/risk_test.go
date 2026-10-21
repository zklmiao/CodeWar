package match

import (
	"math"
	"testing"

	"codewar/server/gen/matchpb"
)

// ===== 难度风险系数（05 号票）=====
// spec 001 附注：难度系数 L1 ×0.85 → L5 ×1.20；L3=1.0 为向后兼容锚点
// （无配置对局与 05 之前逐字节一致，05 裁决 5）。

func TestRiskCoeffTable(t *testing.T) {
	cases := map[int]float64{
		1: 0.85, 2: 0.925, 3: 1.0, 4: 1.1, 5: 1.2,
	}
	for tier, want := range cases {
		if got := RiskCoeff(tier); math.Abs(got-want) > 1e-9 {
			t.Errorf("RiskCoeff(L%d) = %v, want %v", tier, got, want)
		}
	}
	// 未配置（0/负）= L3 锚点；越界钳制到端点
	if RiskCoeff(0) != 1.0 || RiskCoeff(-2) != 1.0 {
		t.Errorf("未配置档应锚定 L3=1.0")
	}
	if RiskCoeff(9) != 1.2 || RiskCoeff(99) != 1.2 {
		t.Errorf("超界档应钳制到 L5=1.2")
	}
}

// newRiskHarness：带难度档的引擎 harness（其余与 newHarness 一致；
// class/skill 绑定须在 StartMatch 之前完成，故不自动开赛）。
func newRiskHarness(t *testing.T, tier int, classID, skillID string) *harness {
	t.Helper()
	clk := &fakeClock{now: baseTime}
	jg := &scriptJudge{results: []JudgeResult{passOK()}}
	var envs []*matchpb.Envelope
	eng := NewEngineWithSetup(clk, StubQuestions{}, jg, func(e *matchpb.Envelope) { envs = append(envs, e) },
		Setup{RiskTier: tier})
	if classID != "" {
		if err := eng.BindClass(P1, classID, skillID); err != nil {
			t.Fatalf("职业绑定失败：%v", err)
		}
	}
	eng.StartMatch()
	return &harness{t: t, clk: clk, jg: jg, eng: eng, envs: &envs}
}

// attackDamages：双方同刻秒解普攻（solveMs=0 → 时间系数 1.0），返回攻击
// 造成的伤害值（两人对称，取首个）。只经公共边界观察事件流（spec 测试决策）。
func attackDamages(t *testing.T, tier int) []int32 {
	t.Helper()
	h := newRiskHarness(t, tier, "", "")
	h.attack(P1, "code")
	h.attack(P2, "code")
	var out []int32
	for _, e := range h.events() {
		if ar := e.GetActionResolved(); ar != nil && ar.GetAction() == ActionAttack {
			out = append(out, ar.GetDamage())
		}
	}
	if len(out) == 0 {
		t.Fatal("无攻击结算事件")
	}
	return out
}

// 验收 3/4：同操作对照（同 solveMs）——L1/L5 伤害比 ≈ 0.85:1.20，L5 显著高于
// L1（≥1.35 倍）；L3=22 为 05 之前的字节级锚点。
func TestAttackDamageScalesWithRiskTier(t *testing.T) {
	d1 := attackDamages(t, 1)
	d3 := attackDamages(t, 3)
	d5 := attackDamages(t, 5)

	for _, got := range d1 {
		if got != 19 { // round(22 × 0.85) = round(18.7)
			t.Fatalf("L1 普攻伤害应为 19，实际 %d", got)
		}
	}
	for _, got := range d3 {
		if got != 22 { // L3 锚点：系数 1.0，与 05 之前一致
			t.Fatalf("L3 普攻伤害应为 22（向后兼容锚点），实际 %d", got)
		}
	}
	for _, got := range d5 {
		if got != 26 { // round(22 × 1.2) = round(26.4)
			t.Fatalf("L5 普攻伤害应为 26，实际 %d", got)
		}
	}
	ratio := float64(d5[0]) / float64(d1[0])
	if ratio < 1.35 {
		t.Fatalf("L5 伤害应显著高于 L1（≥1.35 倍），实际比值 %v", ratio)
	}
	// 伤害比（L5/L1）≈ 系数比（四舍五入允许 ≤0.05 漂移）：0.85:1.20 的倒数形式
	want := RiskCoeffMax / RiskCoeffMin
	if math.Abs(ratio-want) > 0.05 {
		t.Fatalf("伤害比 %v 与系数比 %v 偏差超容差", ratio, want)
	}
}

// 05 裁决 2：技能伤害为设计 003 字面值，不乘风险系数（L1 与 L5 同值）。
func TestSkillDamageIgnoresRiskCoeff(t *testing.T) {
	for _, tier := range []int{1, 5} {
		h := newRiskHarness(t, tier, "warrior", "cleave") // 巨力斩：字面 55，邻域 8 格
		h.eng.Submit(P1, ActionSkill, "code", Pos{})
		h.attack(P2, "code")
		var skillDmg int32
		for _, e := range h.events() {
			if ar := e.GetActionResolved(); ar != nil && ar.GetAction() == ActionSkill {
				skillDmg = ar.GetDamage()
			}
		}
		if skillDmg != 55 {
			t.Fatalf("L%d 技能伤害应为字面值 55（不乘系数），实际 %d", tier, skillDmg)
		}
	}
}

// 快照携带风险系数档位（对局内配置展示的服务端权威链路的引擎侧载体）——
// 验证 Setup.RiskTier 不改变其余快照语义（字段由 hub 填充，引擎不感知题型/策略）。
func TestSnapshotWithRiskTierUnchanged(t *testing.T) {
	h := newRiskHarness(t, 5, "", "")
	snap := h.eng.Snapshot(P1)
	if snap.GetPhase() != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE {
		t.Fatalf("RiskTier 不应影响快照相位：%v", snap.GetPhase())
	}
}
