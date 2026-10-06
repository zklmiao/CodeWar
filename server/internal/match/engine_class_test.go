package match

import (
	"testing"
	"time"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/classes"
)

// ===== 06 号票：职业形状矩阵 + 技能流 + 冷却 + 盲选隐私 =====

// newClassHarness：带职业配置的对局夹具（先绑职业再开赛——与 hub 的 join 时序一致）。
func newClassHarness(t *testing.T, setup Setup, results ...JudgeResult) *harness {
	clk := &fakeClock{now: baseTime}
	jg := &scriptJudge{results: results}
	var envs []*matchpb.Envelope
	eng := NewEngineWithSetup(clk, StubQuestions{}, jg, func(e *matchpb.Envelope) { envs = append(envs, e) }, setup)
	eng.StartMatch()
	return &harness{t: t, clk: clk, jg: jg, eng: eng, envs: &envs}
}

// TestShapeCoverageMatrix：形状覆盖格逐格断言（裁决 6 的测试重心）。
// 3 类普攻 + 4 技能形状 × 方向/距离/网格裁剪。
func TestShapeCoverageMatrix(t *testing.T) {
	cases := []struct {
		name      string
		from      Pos
		shape     string
		depth     int32
		rangeMax  int32
		t         Targeting
		wantCells []Pos
	}{
		{"剑士普攻-相邻4格", Pos{3, 2}, classes.ShapeMelee, 0, 0, Targeting{},
			[]Pos{{2, 2}, {4, 2}, {3, 1}, {3, 3}}},
		{"法师线扫-向上深度2", Pos{3, 2}, classes.ShapeLine, 2, 0, Targeting{Direction: "up"},
			[]Pos{{2, 2}, {1, 2}}},
		{"法师线扫-向下网格裁剪", Pos{3, 2}, classes.ShapeLine, 2, 0, Targeting{Direction: "down"},
			[]Pos{{4, 2}}},
		{"游侠点狙-向上距离2", Pos{3, 2}, classes.ShapeSnipe, 0, 3, Targeting{Direction: "up", Distance: 2},
			[]Pos{{1, 2}}},
		{"游侠点狙-距离钳到射程", Pos{3, 2}, classes.ShapeSnipe, 0, 3, Targeting{Direction: "up", Distance: 9},
			[]Pos{{0, 2}}},
		{"游侠点狙-向右距离4出界裁剪", Pos{2, 0}, classes.ShapeSnipe, 0, 4, Targeting{Direction: "right", Distance: 4},
			[]Pos{{2, 4}}},
		{"巨力斩-邻域8格且裁剪", Pos{0, 0}, classes.ShapeBurst8, 0, 0, Targeting{},
			[]Pos{{0, 1}, {1, 0}, {1, 1}}},
		{"烈焰风暴-向上横带6格", Pos{2, 2}, classes.ShapeBand, 2, 0, Targeting{Direction: "up"},
			[]Pos{{1, 1}, {1, 2}, {1, 3}, {0, 1}, {0, 2}, {0, 3}}},
		{"巨力斩-场中满8格", Pos{2, 2}, classes.ShapeBurst8, 0, 0, Targeting{},
			[]Pos{{1, 1}, {1, 2}, {1, 3}, {2, 1}, {2, 3}, {3, 1}, {3, 2}, {3, 3}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := coverageCells(c.from, c.shape, c.depth, c.rangeMax, c.t)
			if len(got) != len(c.wantCells) {
				t.Fatalf("覆盖格数 %d，期望 %d：%v", len(got), len(c.wantCells), got)
			}
			for _, want := range c.wantCells {
				if !got[want] {
					t.Fatalf("覆盖格缺 %v：实际 %v", want, got)
				}
			}
		})
	}
}

// 职业初始 HP 与 legacy 锚点：装备 warrior/mage 后 HP 110/90；零信息构造 100/100。
func TestClassHPInit(t *testing.T) {
	h := newClassHarness(t, Setup{P1Class: "warrior", P1Skill: "cleave", P2Class: "mage", P2Skill: "firestorm"},
		passOK())
	snap := h.eng.Snapshot(P1)
	if snap.Players[0].Hp != 110 || snap.Players[1].Hp != 90 {
		t.Fatalf("职业初始 HP 应为 110/90，实际 %d/%d", snap.Players[0].Hp, snap.Players[1].Hp)
	}
	if snap.Players[0].ClassId != "warrior" || snap.Players[1].ClassId != "mage" {
		t.Fatalf("开赛后职业应揭示：%q/%q", snap.Players[0].ClassId, snap.Players[1].ClassId)
	}
	plain := NewEngine(&fakeClock{now: baseTime}, StubQuestions{}, nil, nil)
	ps := plain.Snapshot(P1)
	if ps.Players[0].Hp != 100 || ps.Players[0].ClassId != "legacy" {
		t.Fatalf("零信息构造应保持 legacy 锚点 100/legacy，实际 %d/%q",
			ps.Players[0].Hp, ps.Players[0].ClassId)
	}
}

// 盲选隐私（设计 003 §1）：对局开始前快照不下发对方职业；开始后才揭示。
func TestBlindPickPrivacy(t *testing.T) {
	clk := &fakeClock{now: baseTime}
	var envs []*matchpb.Envelope
	eng := NewEngineWithSetup(clk, StubQuestions{}, &scriptJudge{results: []JudgeResult{passOK()}},
		func(e *matchpb.Envelope) { envs = append(envs, e) },
		Setup{P1Class: "warrior", P1Skill: "cleave", P2Class: "mage", P2Skill: "firestorm"})
	snap := eng.Snapshot(P1) // 仍 WAITING
	if snap.Players[1].ClassId != "" || snap.Players[1].SkillId != "" {
		t.Fatalf("开赛前对方职业必须隐藏，实际 %q/%q", snap.Players[1].ClassId, snap.Players[1].SkillId)
	}
	if snap.Players[0].ClassId != "warrior" {
		t.Fatalf("自己的职业始终可见：%q", snap.Players[0].ClassId)
	}
	eng.StartMatch()
	snap = eng.Snapshot(P1)
	if snap.Players[1].ClassId != "mage" {
		t.Fatalf("开赛后对方职业应揭示：mage，实际 %q", snap.Players[1].ClassId)
	}
}

// 剑士普攻 26：相邻命中；法师线扫打距离 2（超出剑士相邻射程）；
// 游侠点狙显式 targeting 与自动瞄准两分支都命中。
func TestClassAttackShapes(t *testing.T) {
	t.Run("剑士相邻命中26", func(t *testing.T) {
		h := newClassHarness(t, Setup{P1Class: "warrior", P2Class: "mage"}, passOK())
		h.eng.Submit(P1, ActionAttack, "good", Pos{}) // (3,2)→(2,2) 相邻
		h.timeoutRound()
		acts := actions(h.events())
		if len(acts) != 1 || acts[0].Damage != 26 {
			t.Fatalf("剑士普攻应 26 伤害：%+v", acts)
		}
	})
	t.Run("法师线扫深2命中", func(t *testing.T) {
		// 实际站位：法师 P1 (3,2)、剑士 P2 (2,2)。剑士先上移 1 到 (1,2)（与法师距离 2，
		// 超出剑士相邻射程），法师向上线扫（深度 2 覆盖 (2,2)(1,2)）命中
		h := newClassHarness(t, Setup{P1Class: "mage", P1Skill: "firestorm", P2Class: "warrior", P2Skill: "cleave"}, passOK())
		h.eng.Submit(P2, ActionMove, "good", Pos{Row: 1, Col: 2})
		h.eng.SetTargeting(P1, Targeting{Direction: "up"})
		h.eng.Submit(P1, ActionAttack, "good", Pos{})
		h.timeoutRound()
		acts := actions(h.events())
		if len(acts) != 2 || acts[0].Action != ActionMove {
			t.Fatalf("应先手移动后线扫：%+v", acts)
		}
		if acts[1].Damage != 18 || acts[1].Dodged {
			t.Fatalf("法师线扫应 18 伤害且命中距离 2 目标：%+v", acts[1])
		}
	})
	t.Run("剑士打距离2目标落空", func(t *testing.T) {
		// 实际站位：剑士 P1 (3,2)、法师 P2 (2,2)。法师先上移 2 到 (0,2)（相邻 4 格外），
		// 剑士向上攻击覆盖 (2,2)——法师已不在 → 闪避
		h := newClassHarness(t, Setup{P1Class: "warrior", P2Class: "mage"}, passOK())
		h.eng.Submit(P2, ActionMove, "good", Pos{Row: 0, Col: 2})
		h.eng.Submit(P1, ActionAttack, "good", Pos{})
		h.timeoutRound()
		acts := actions(h.events())
		if len(acts) != 2 || !acts[1].Dodged {
			t.Fatalf("对手移出相邻 4 格应闪避：%+v", acts)
		}
	})
	t.Run("游侠点狙-显式targeting", func(t *testing.T) {
		// 实际站位：游侠 P1 (3,2)、剑士 P2 (2,2)。显式 up+1 命中；下一轮显式 up+3
		// 落到 (0,2)（对手在 (2,2)）→ 距离语义真实生效
		h := newClassHarness(t, Setup{P1Class: "ranger", P2Class: "warrior", P2Skill: "cleave"}, passOK())
		h.eng.SetTargeting(P1, Targeting{Direction: "up", Distance: 1})
		h.eng.Submit(P1, ActionAttack, "good", Pos{})
		h.timeoutRound()
		acts := actions(h.events())
		if len(acts) != 1 || acts[0].Damage != 20 || acts[0].Dodged {
			t.Fatalf("游侠点狙 up+1 应 20 伤害且命中：%+v", acts)
		}
		h.eng.SetTargeting(P1, Targeting{Direction: "up", Distance: 3})
		h.eng.Submit(P1, ActionAttack, "good", Pos{})
		h.timeoutRound()
		acts = actions(h.events())
		if len(acts) != 2 || !acts[1].Dodged {
			t.Fatalf("游侠点狙 up+3 应回到落空分支：%+v", acts[1])
		}
	})
	t.Run("游侠点狙-自动瞄准缺省", func(t *testing.T) {
		h := newClassHarness(t, Setup{P1Class: "ranger", P2Class: "warrior", P2Skill: "cleave"}, passOK())
		h.eng.Submit(P1, ActionAttack, "good", Pos{}) // 不提交 targeting
		h.timeoutRound()
		acts := actions(h.events())
		if len(acts) != 1 || acts[0].Dodged {
			t.Fatalf("自动瞄准应命中相邻对手：%+v", acts)
		}
	})
}

// 技能流：巨力斩字面值 55（不乘时间系数——慢解同值）+ 冷却跨回合
// （CD4：第 1 回合释放 → 第 2/3/4 回合灰置拒绝 → 第 5 回合恢复）。
func TestSkillDamageAndCooldown(t *testing.T) {
	h := newClassHarness(t, Setup{P1Class: "warrior", P1Skill: "cleave", P2Class: "mage"}, passOK())

	// R1：慢解（30s）技能——伤害仍为字面 55（普攻此时只有 ~18.5，系数差异即证词）
	h.clk.Advance(30 * time.Second)
	h.eng.Submit(P1, ActionSkill, "good", Pos{})
	h.timeoutRound()
	acts := actions(h.events())
	if len(acts) != 1 || acts[0].Action != ActionSkill || acts[0].Damage != 55 {
		t.Fatalf("巨力斩应为字面 55 伤害（不乘时间系数）：%+v", acts)
	}
	if hp := lastHp(h.events(), "p2"); hp != 35 {
		t.Fatalf("法师 90-55=35，实际 %d", hp)
	}

	// R2~R4：冷却中——提交被拒（skill-cooldown），不判题不惩罚
	for want := 1; want <= 3; want++ {
		h.eng.Submit(P1, ActionSkill, "good", Pos{})
		rej := rejections(h.events())
		if len(rej) != want || rej[len(rej)-1].Reason != "skill-cooldown" {
			t.Fatalf("第 %d 回合技能提交应被 skill-cooldown 拒绝：%+v", want, rej)
		}
		h.timeoutRound()
	}

	// R5：冷却结束（1+4=5）→ 再次释放
	h.eng.Submit(P1, ActionSkill, "good", Pos{})
	h.timeoutRound()
	if n := len(actions(h.events())); n != 2 {
		t.Fatalf("冷却结束后应可再次释放（共 2 次技能结算），实际 %d", n)
	}
	ends := endings(h.events())
	if len(ends) != 1 || ends[0].Winner != P1 || ends[0].Reason != ReasonKO {
		t.Fatalf("第二发巨力斩应击杀法师（35-55<0）P1 胜 ko：%+v", ends)
	}
}

// 治疗不溢出（设计 003 §1）：满血治疗量为 0（无 HpChanged）；残血治疗截断到最大 HP。
func TestHealOverflowCapped(t *testing.T) {
	h := newClassHarness(t, Setup{P1Class: "warrior", P1Skill: "survive", P2Class: "mage"}, passOK())
	// R1：满血治疗 → ActionResolved.heal=0，无 HpChanged
	h.eng.Submit(P1, ActionSkill, "good", Pos{})
	h.timeoutRound()
	for _, e := range h.events() {
		if hp := e.GetHpChanged(); hp != nil && hp.Cause == "heal" {
			t.Fatalf("满血不应产生治疗 HpChanged：%+v", hp)
		}
	}
	heals := 0
	for _, a := range actions(h.events()) {
		heals += int(a.Heal)
	}
	if heals != 0 {
		t.Fatalf("满血治疗量应为 0，实际 %d", heals)
	}
	// R2：法师普攻打掉 18（110→92）；R3 空回合过冷却（R1 已施放，CD3 → R2/R3 灰置）
	h.eng.Submit(P2, ActionAttack, "good", Pos{})
	h.timeoutRound()
	h.timeoutRound()
	// R4：冷却结束（1+3=4），治疗 +30 截断到最大 HP 110（92+18）
	h.clk.Advance(10 * time.Second)
	h.eng.Submit(P1, ActionSkill, "good", Pos{})
	h.timeoutRound()
	if hp := lastHp(h.events(), "p1"); hp != 110 {
		t.Fatalf("治疗应截断到最大 HP 110，实际 %d", hp)
	}
}

// 秘法汲取：命中 → 对手 -25、自己 +20（截断）；对手满血时自疗为 0 仍成立。
func TestDrainHeals(t *testing.T) {
	h := newClassHarness(t, Setup{P1Class: "mage", P1Skill: "drain", P2Class: "warrior", P2Skill: "cleave"}, passOK())
	// R1：剑士普攻法师 26（90→64）；法师解出汲取（对手相邻，自动瞄准命中）
	h.eng.Submit(P2, ActionAttack, "good", Pos{})
	h.clk.Advance(5 * time.Second)
	h.eng.Submit(P1, ActionSkill, "good", Pos{})
	h.timeoutRound()
	if hp := lastHp(h.events(), "p2"); hp != 85 {
		t.Fatalf("汲取应伤 25（110→85），实际 %d", hp)
	}
	if hp := lastHp(h.events(), "p1"); hp != 84 {
		t.Fatalf("汲取自疗 20（64→84），实际 %d", hp)
	}
}

// 盾 × 技能伤害：巨力斩 55 打在护盾上 → 55×0.4=22。
func TestSkillDamageShielded(t *testing.T) {
	h := newClassHarness(t, Setup{P1Class: "warrior", P1Skill: "cleave", P2Class: "mage", P2Skill: "firestorm"}, passOK())
	h.eng.Submit(P2, ActionDefend, "good", Pos{}) // 法师先手开盾
	h.eng.Submit(P1, ActionSkill, "good", Pos{})  // 剑士巨力斩（相邻在 8 格邻域内）
	h.timeoutRound()
	acts := actions(h.events())
	for _, a := range acts {
		if a.Action == ActionSkill && a.Attacker == P1 {
			if a.Damage != 22 || !a.Shielded {
				t.Fatalf("巨力斩受盾应 55×0.4=22：%+v", a)
			}
		}
	}
}

// 工具：某玩家最后一次 HpChanged 后的 HP。
func lastHp(envs []*matchpb.Envelope, player Player) int32 {
	out := int32(-999)
	for _, e := range envs {
		if hp := e.GetHpChanged(); hp != nil && hp.Player == player {
			out = hp.Hp
		}
	}
	return out
}
