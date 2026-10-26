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
		// 27 号票：横向旋转分支（left/right）与向下方向逐格断言——横带最易错的几何。
		{"烈焰风暴-向下横带6格", Pos{2, 2}, classes.ShapeBand, 2, 0, Targeting{Direction: "down"},
			[]Pos{{3, 1}, {3, 2}, {3, 3}, {4, 1}, {4, 2}, {4, 3}}},
		{"烈焰风暴-向左横带6格", Pos{2, 2}, classes.ShapeBand, 2, 0, Targeting{Direction: "left"},
			[]Pos{{1, 1}, {2, 1}, {3, 1}, {1, 0}, {2, 0}, {3, 0}}},
		{"烈焰风暴-向右横带6格", Pos{2, 2}, classes.ShapeBand, 2, 0, Targeting{Direction: "right"},
			[]Pos{{1, 3}, {2, 3}, {3, 3}, {1, 4}, {2, 4}, {3, 4}}},
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
	// 血量同样泄露职业（110/90/95 三档互异）：盲选期对手行按初始 100 上报
	// （review 发现：WAITING 快照 hp 可反推职业）
	if snap.Players[1].Hp != 100 {
		t.Fatalf("开赛前对手血量必须按初始 100 上报（防职业反推），实际 %d", snap.Players[1].Hp)
	}
	if snap.Players[0].Hp != 110 {
		t.Fatalf("自己的真实血量（剑士 110）始终可见：%d", snap.Players[0].Hp)
	}
	eng.StartMatch()
	snap = eng.Snapshot(P1)
	if snap.Players[1].ClassId != "mage" {
		t.Fatalf("开赛后对方职业应揭示：mage，实际 %q", snap.Players[1].ClassId)
	}
	if snap.Players[1].Hp != 90 {
		t.Fatalf("开赛后对手真实血量（法师 90）应揭示：%d", snap.Players[1].Hp)
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

// 秘法汲取落空（27 号票，TestDrainHeals 的反向断言）：Dodged 事件、
// 无伤害 HpChanged、无自疗——未命中时伤害与汲取两段都不发生（设计 003 §4）。
func TestDrainMiss(t *testing.T) {
	h := newClassHarness(t, Setup{P1Class: "mage", P1Skill: "drain", P2Class: "warrior", P2Skill: "cleave"}, passOK())
	// P2 先手右移 2 到 (2,4)：曼哈顿 3 > 汲取射程 2；法师自动瞄准 right+2 覆盖 (3,4)
	h.eng.Submit(P2, ActionMove, "good", Pos{Row: 2, Col: 4})
	h.clk.Advance(5 * time.Second)
	h.eng.Submit(P1, ActionSkill, "good", Pos{})
	h.timeoutRound()
	acts := actions(h.events())
	if len(acts) != 2 || acts[0].Action != ActionMove || acts[1].Action != ActionSkill {
		t.Fatalf("应先手移动后施法：%+v", acts)
	}
	if !acts[1].Dodged || acts[1].Damage != 0 || acts[1].Heal != 0 {
		t.Fatalf("汲取未命中应 Dodged 且无伤害/自疗：%+v", acts[1])
	}
	if hps := hpChanges(h.events()); len(hps) != 0 {
		t.Fatalf("落空不应产生任何 HpChanged（含自疗）：%+v", hps)
	}
	snap := h.eng.Snapshot(P1)
	if snap.Players[0].Hp != 90 || snap.Players[1].Hp != 110 {
		t.Fatalf("落空后双方血量应不变（90/110），实际 %d/%d", snap.Players[0].Hp, snap.Players[1].Hp)
	}
}

// 烈焰风暴事件流（27 号票）：横向旋转分支的产品级证据——对手在横带覆盖格内
// 命中、带外闪避。矩阵逐格断言之外的结算 seam。
func TestFirestormBandEventStream(t *testing.T) {
	t.Run("横向带内命中40", func(t *testing.T) {
		h := newClassHarness(t, Setup{P1Class: "mage", P1Skill: "firestorm", P2Class: "warrior"}, passOK())
		// 剑士先手移到 (2,3)：向 up 施法时该格是横带的横向覆盖格
		h.eng.Submit(P2, ActionMove, "good", Pos{Row: 2, Col: 3})
		h.clk.Advance(5 * time.Second)
		h.eng.SetTargeting(P1, Targeting{Direction: "up"})
		h.eng.Submit(P1, ActionSkill, "good", Pos{})
		h.timeoutRound()
		acts := actions(h.events())
		if len(acts) != 2 || acts[0].Action != ActionMove ||
			acts[0].MoveTo.Row != 2 || acts[0].MoveTo.Col != 3 {
			// review 加固：走位必须真实落到 (2,3)，否则"横向格命中"退化为中心格命中
			t.Fatalf("应先手移动到 (2,3) 后施法：%+v", acts)
		}
		if acts[1].Action != ActionSkill || acts[1].Dodged || acts[1].Damage != 40 {
			t.Fatalf("横向带内应命中 40：%+v", acts[1])
		}
		if hp := lastHp(h.events(), "p2"); hp != 70 {
			t.Fatalf("剑士 110-40=70，实际 %d", hp)
		}
	})
	t.Run("横向带外闪避", func(t *testing.T) {
		h := newClassHarness(t, Setup{P1Class: "mage", P1Skill: "firestorm", P2Class: "warrior"}, passOK())
		// 剑士先手移到 (2,4)：在 up 横带（列 1..3）之外
		h.eng.Submit(P2, ActionMove, "good", Pos{Row: 2, Col: 4})
		h.clk.Advance(5 * time.Second)
		h.eng.SetTargeting(P1, Targeting{Direction: "up"})
		h.eng.Submit(P1, ActionSkill, "good", Pos{})
		h.timeoutRound()
		acts := actions(h.events())
		if len(acts) != 2 || acts[0].Action != ActionMove ||
			acts[0].MoveTo.Row != 2 || acts[0].MoveTo.Col != 4 {
			// review 加固：走位必须真实落到 (2,4)，否则"带外闪避"的几何前提不成立
			t.Fatalf("应先手移动到 (2,4) 后施法：%+v", acts)
		}
		if acts[1].Action != ActionSkill || !acts[1].Dodged {
			t.Fatalf("横向带外应闪避：%+v", acts)
		}
		if hps := hpChanges(h.events()); len(hps) != 0 {
			t.Fatalf("闪避不应产生 HpChanged：%+v", hps)
		}
		if hp := h.eng.Snapshot(P1).Players[1].Hp; hp != 110 {
			t.Fatalf("闪避后剑士应仍为 110，实际 %d", hp)
		}
	})
}

// 穿云箭端到端（27 号票）：命中伤害 45（字面值不乘时间系数）、CD4 灰置窗口、
// 距离 1/4 边界命中与超距落空。
func TestPierceEndToEnd(t *testing.T) {
	t.Run("距离1命中45与CD4", func(t *testing.T) {
		h := newClassHarness(t, Setup{P1Class: "ranger", P1Skill: "pierce", P2Class: "warrior"}, passOK())
		h.clk.Advance(30 * time.Second) // 慢解仍为字面 45（技能不乘时间系数）
		h.eng.SetTargeting(P1, Targeting{Direction: "up", Distance: 1})
		h.eng.Submit(P1, ActionSkill, "good", Pos{})
		h.timeoutRound()
		acts := actions(h.events())
		if len(acts) != 1 || acts[0].Dodged || acts[0].Damage != 45 {
			t.Fatalf("穿云箭距离1应命中 45：%+v", acts)
		}
		if hp := lastHp(h.events(), "p2"); hp != 65 {
			t.Fatalf("剑士 110-45=65，实际 %d", hp)
		}
		// CD4 登记：R1 释放 → 第 2 回合快照剩 3 回合（第 5 回合起可用）
		if cd := h.eng.Snapshot(P1).Players[0].SkillCdRounds; cd != 3 {
			t.Fatalf("CD4 在第 2 回合应剩 3 回合，实际 %d", cd)
		}
		// R2~R4 灰置拒绝（三次 skill-cooldown）；R5 恢复可释放并再次命中
		for want := 1; want <= 3; want++ {
			h.eng.Submit(P1, ActionSkill, "good", Pos{})
			rej := rejections(h.events())
			if len(rej) != want || rej[len(rej)-1].Reason != "skill-cooldown" {
				t.Fatalf("第 %d 次冷却提交应被 skill-cooldown 拒绝：%+v", want, rej)
			}
			h.timeoutRound()
		}
		h.eng.SetTargeting(P1, Targeting{Direction: "up", Distance: 1})
		h.eng.Submit(P1, ActionSkill, "good", Pos{})
		h.timeoutRound()
		acts = actions(h.events())
		if len(acts) != 2 || acts[1].Dodged || acts[1].Damage != 45 {
			t.Fatalf("CD4 结束后应可再次命中 45：%+v", acts)
		}
		if hp := lastHp(h.events(), "p2"); hp != 20 {
			t.Fatalf("第二发后剑士 65-45=20，实际 %d", hp)
		}
	})
	t.Run("距离4边界命中（超上限距离钳到4）", func(t *testing.T) {
		h := newClassHarness(t, Setup{P1Class: "ranger", P1Skill: "pierce", P2Class: "warrior"}, passOK())
		// R1：游侠上移 3 到 (0,2)；R2：剑士再下移 2 到 (4,2)——轴向距离恰为射程上限 4
		// （分两回合走位：剑士下移 2 的中间格 (3,2) 初始被游侠占据，不能同回合穿越）
		h.eng.Submit(P1, ActionMove, "good", Pos{Row: 0, Col: 2})
		h.timeoutRound()
		h.eng.Submit(P2, ActionMove, "good", Pos{Row: 4, Col: 2})
		h.timeoutRound()
		// review 加固：钉住走位落点，防止"移动被静默拒绝后仍算边界命中"的假 pass
		if mvs := actions(h.events()); len(mvs) != 2 ||
			mvs[0].MoveTo.Row != 0 || mvs[0].MoveTo.Col != 2 ||
			mvs[1].MoveTo.Row != 4 || mvs[1].MoveTo.Col != 2 {
			t.Fatalf("边界场景走位应落 (0,2)/(4,2)：%+v", mvs)
		}
		// R3：提交超上限距离 9 → 钳到 4，命中边界格
		h.eng.SetTargeting(P1, Targeting{Direction: "down", Distance: 9})
		h.eng.Submit(P1, ActionSkill, "good", Pos{})
		h.timeoutRound()
		var skill *matchpb.ActionResolved
		for _, a := range actions(h.events()) {
			if a.Action == ActionSkill {
				skill = a
			}
		}
		if skill == nil || skill.Dodged || skill.Damage != 45 {
			t.Fatalf("距离4边界应命中 45（距离钳到4）：%+v", skill)
		}
		if hp := lastHp(h.events(), "p2"); hp != 65 {
			t.Fatalf("剑士 110-45=65，实际 %d", hp)
		}
	})
	t.Run("超距落空", func(t *testing.T) {
		h := newClassHarness(t, Setup{P1Class: "ranger", P1Skill: "pierce", P2Class: "warrior"}, passOK())
		// R1：游侠左移 2 到 (3,0)，剑士右移 2 到 (2,4)——曼哈顿距离 5 > 射程 4
		h.eng.Submit(P1, ActionMove, "good", Pos{Row: 3, Col: 0})
		h.eng.Submit(P2, ActionMove, "good", Pos{Row: 2, Col: 4})
		h.timeoutRound()
		// review 加固：钉住走位落点——任一移动被静默拒绝时，本用例的"超距"前提
		// 不成立（如游侠留在 (3,2)，right+4 出界同样闪避，会以假 pass 混过）
		if mvs := actions(h.events()); len(mvs) != 2 ||
			mvs[0].MoveTo.Row != 3 || mvs[0].MoveTo.Col != 0 ||
			mvs[1].MoveTo.Row != 2 || mvs[1].MoveTo.Col != 4 {
			t.Fatalf("超距场景走位应落 (3,0)/(2,4)：%+v", mvs)
		}
		// R2：right+4 覆盖 (3,4)，对手在 (2,4)——超距落空
		h.eng.SetTargeting(P1, Targeting{Direction: "right", Distance: 4})
		h.eng.Submit(P1, ActionSkill, "good", Pos{})
		h.timeoutRound()
		var skill *matchpb.ActionResolved
		for _, a := range actions(h.events()) {
			if a.Action == ActionSkill {
				skill = a
			}
		}
		if skill == nil || !skill.Dodged || skill.Damage != 0 {
			t.Fatalf("超距应落空：%+v", skill)
		}
		if hps := hpChanges(h.events()); len(hps) != 0 {
			t.Fatalf("落空不应产生 HpChanged：%+v", hps)
		}
	})
}

// 风行回复（27 号票）：治疗 28、满血溢出封顶 0、CD2；自身治疗不经过瞄准/命中路径。
func TestWindheal(t *testing.T) {
	t.Run("满血溢出封顶0", func(t *testing.T) {
		h := newClassHarness(t, Setup{P1Class: "ranger", P1Skill: "windheal", P2Class: "warrior"}, passOK())
		h.eng.Submit(P1, ActionSkill, "good", Pos{})
		h.timeoutRound()
		acts := actions(h.events())
		if len(acts) != 1 || acts[0].Action != ActionSkill || acts[0].Dodged || acts[0].Heal != 0 {
			t.Fatalf("满血治疗量应为 0 且不闪避：%+v", acts)
		}
		for _, hp := range hpChanges(h.events()) {
			if hp.Cause == "heal" {
				t.Fatalf("满血不应产生治疗 HpChanged：%+v", hp)
			}
		}
		// HP 不变量（review 加固）：封顶为 0 意味着满血治疗不得实际改动血量——
		// 只看上报的 Heal=0 抓不住"上报 0 却暗中加血"的回归。
		snap := h.eng.Snapshot(P1)
		if snap.Players[0].Hp != 95 || snap.Players[1].Hp != 110 {
			t.Fatalf("满血治疗后血量必须不变（95/110），实际 %d/%d",
				snap.Players[0].Hp, snap.Players[1].Hp)
		}
	})
	t.Run("治疗28与CD2", func(t *testing.T) {
		h := newClassHarness(t, Setup{P1Class: "ranger", P1Skill: "windheal", P2Class: "warrior"}, passOK())
		// R1/R2：剑士普攻 26×2 → 游侠 95→43（治疗空间 ≥28）
		h.eng.Submit(P2, ActionAttack, "good", Pos{})
		h.timeoutRound()
		h.eng.Submit(P2, ActionAttack, "good", Pos{})
		h.timeoutRound()
		// R3：对手远离（移到 (0,2)），并提交一个方向 targeting——自身治疗不受影响。
		// 双方本回合都行动 → 引擎在第二次 Submit 内立即结算，不再拨钟（否则会多跳一回合）。
		h.eng.Submit(P2, ActionMove, "good", Pos{Row: 0, Col: 2})
		h.clk.Advance(5 * time.Second)
		h.eng.SetTargeting(P1, Targeting{Direction: "left", Distance: 1})
		h.eng.Submit(P1, ActionSkill, "good", Pos{})
		acts := actions(h.events())
		var heal *matchpb.ActionResolved
		for _, a := range acts {
			if a.Action == ActionSkill {
				heal = a
			}
		}
		if heal == nil || heal.Dodged || heal.Heal != 28 {
			t.Fatalf("风行回复应治疗 28 且不经命中判定：%+v", heal)
		}
		if hp := lastHp(h.events(), "p1"); hp != 71 {
			t.Fatalf("游侠 43+28=71，实际 %d", hp)
		}
		// CD2：R3 释放 → R4 灰置拒绝；R5 恢复可释放
		h.eng.Submit(P1, ActionSkill, "good", Pos{})
		rej := rejections(h.events())
		if len(rej) != 1 || rej[0].Reason != "skill-cooldown" {
			t.Fatalf("CD2 第 4 回合应拒绝释放：%+v", rej)
		}
		h.timeoutRound()
		h.eng.Submit(P1, ActionSkill, "good", Pos{})
		h.timeoutRound()
		if rej := rejections(h.events()); len(rej) != 1 {
			t.Fatalf("R5 不应新增拒绝：%+v", rej)
		}
		skillActs := 0
		for _, a := range actions(h.events()) {
			if a.Action == ActionSkill {
				skillActs++
			}
		}
		if skillActs != 2 {
			t.Fatalf("R3+R5 应有两次技能结算，实际 %d", skillActs)
		}
	})
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

// 20 号票回归：PanelReady 事件必须逐回合携带当回合技能题（PrivatePanel.skill）。
// 缺陷史：privatePanelToPb 只填 move/defend，事件路径的技能题恒空——客户端每回合
// 用事件刷新面板，技能题从第 2 回合起消失；快照路径（Snapshot.panel.skill）一直
// 正确，两边测试各自绿、proto 契约在 seam 交叉处断了无人看。
// 红证据（2026-10-24 修复前首跑）：R1 即红——
//
//	"PanelReadyCarriesSkillQuestion: 第 1 回合 PanelReady.p1.skill 应携带当回合技能题（非 nil 且 ID 非空）"
//	（PrivatePanel.skill 恒为 nil，与票面根因证据一致）。
func TestPanelReadyCarriesSkillQuestion(t *testing.T) {
	h := newClassHarness(t, Setup{P1Class: "warrior", P1Skill: "cleave"}, passOK())

	// 回合 1..4：装备玩家（剑士+巨力斩，CD=4）每回合都携带新技能题；ID 随回合变化
	// （SkillQuestion seed 含 round——用回合间变化断言，不从实现反抄具体 ID）；
	// 未装备玩家（P2 legacy）不携带——与快照路径的 ok-跳过对齐。
	prevID := ""
	for round := 1; round <= 4; round++ {
		ps := panels(h.events())
		if len(ps) != round {
			t.Fatalf("第 %d 回合应有 %d 条 PanelReady，实际 %d", round, round, len(ps))
		}
		cur := ps[round-1]
		if cur.P1.Skill == nil || cur.P1.Skill.Id == "" {
			t.Fatalf("第 %d 回合 PanelReady.p1.skill 应携带当回合技能题（非 nil 且 ID 非空）", round)
		}
		if round > 1 && cur.P1.Skill.Id == prevID {
			t.Fatalf("技能题 ID 应随回合变化：第 %d 与第 %d 回合同为 %q", round-1, round, cur.P1.Skill.Id)
		}
		prevID = cur.P1.Skill.Id
		if cur.P2.Skill != nil {
			t.Fatalf("未装备玩家（legacy）的 PanelReady.p2.skill 不应携带，实际 ID %q", cur.P2.Skill.Id)
		}
		if round == 1 {
			h.eng.Submit(P1, ActionSkill, "good", Pos{}) // R1 释放：冷却登记（结算语义归 16 号票，此处不断言）
		}
		h.timeoutRound()
	}

	// 真实链路收尾（验收 2 最低要求）：冷却结束（1+4=5）后技能题仍在且可再次提交——
	// R5 提交无 rejection、判题通过（R1+R5 共两次技能通过）。
	h.eng.Submit(P1, ActionSkill, "good", Pos{})
	h.timeoutRound()
	if rej := rejections(h.events()); len(rej) != 0 {
		t.Fatalf("冷却结束后技能提交不应被拒：%+v", rej)
	}
	passedSkill := 0
	for _, j := range judged(h.events()) {
		if j.Player == P1 && j.Action == ActionSkill && j.Passed {
			passedSkill++
		}
	}
	if passedSkill != 2 {
		t.Fatalf("R1 与 R5 应各有一次技能判题通过，实际 %d", passedSkill)
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
