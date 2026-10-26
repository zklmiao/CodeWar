package classes

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// 表驱动核对：registry 与设计 003 §2/§4 的字面值逐格一致（数值纪律：改表先改设计文档）。
func TestRegistryMatchesDesign003(t *testing.T) {
	cases := []struct {
		id                string
		hp, atk, moveMax  int32
		skillIDs          []string
		wantSkillPower    map[string]int32 // skillID → 伤害或治疗量
		wantSkillCooldown map[string]int32
	}{
		{"warrior", 110, 26, 2, []string{"cleave", "survive"},
			map[string]int32{"cleave": 55, "survive": 30},
			map[string]int32{"cleave": 4, "survive": 3}},
		{"mage", 90, 18, 2, []string{"firestorm", "drain"},
			map[string]int32{"firestorm": 40, "drain": 25},
			map[string]int32{"firestorm": 3, "drain": 3}},
		{"ranger", 95, 20, 3, []string{"pierce", "windheal"},
			map[string]int32{"pierce": 45, "windheal": 28},
			map[string]int32{"pierce": 4, "windheal": 2}},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			got, ok := ByID(c.id)
			if !ok {
				t.Fatalf("职业 %s 不在注册表", c.id)
			}
			if got.HP != c.hp || got.AttackDamage != c.atk || got.MoveMax != c.moveMax {
				t.Fatalf("%s 基础数值漂移：got HP=%d atk=%d move=%d，期望 %d/%d/%d",
					c.id, got.HP, got.AttackDamage, got.MoveMax, c.hp, c.atk, c.moveMax)
			}
			if len(got.Skills) != len(c.skillIDs) {
				t.Fatalf("%s 技能数 %d，期望 %d", c.id, len(got.Skills), len(c.skillIDs))
			}
			for _, sid := range c.skillIDs {
				s, ok, err := SkillByID(c.id, sid)
				if err != nil || !ok {
					t.Fatalf("%s 缺技能 %s：ok=%v err=%v", c.id, sid, ok, err)
				}
				wantPower := c.wantSkillPower[sid]
				wantCD := c.wantSkillCooldown[sid]
				amount := s.Power
				if s.Kind == KindHeal {
					amount = s.Heal
				}
				if amount != wantPower || s.Cooldown != wantCD {
					t.Fatalf("%s/%s 数值漂移：量=%d CD=%d，期望 %d/%d",
						c.id, sid, amount, s.Cooldown, wantPower, wantCD)
				}
			}
		})
	}
}

// 秘法汲取是唯一 drain：伤 25 + 自疗 20（两段都在表内字面值）。
func TestDrainIsComposite(t *testing.T) {
	s, ok, err := SkillByID("mage", "drain")
	if err != nil || !ok {
		t.Fatalf("drain 应存在：ok=%v err=%v", ok, err)
	}
	if s.Kind != KindDrain || s.Power != 25 || s.Heal != 20 {
		t.Fatalf("drain 应为 伤25+自疗20，实际 kind=%s power=%d heal=%d", s.Kind, s.Power, s.Heal)
	}
}

// legacy 锚点：既有测试与匿名对局的零改动保证（HP100/攻22/相邻/移动≤2/无技能）。
func TestLegacyAnchor(t *testing.T) {
	l := Legacy()
	if l.ID != "legacy" || l.HP != 100 || l.AttackDamage != 22 || l.AttackShape != ShapeMelee || l.MoveMax != 2 {
		t.Fatalf("legacy 锚点漂移：%+v", l)
	}
	if len(l.Skills) != 0 {
		t.Fatalf("legacy 不应有技能（无技能位语义）：%+v", l.Skills)
	}
	if _, ok := ByID("legacy"); ok {
		t.Fatal("legacy 不应出现在可选拔清单（仅作引擎缺省锚点）")
	}
}

// 装备校验：职业内技能合法；跨职业技能、未知职业、未知技能一律拒绝（hub fail-closed）。
func TestValidateLoadout(t *testing.T) {
	if err := ValidateLoadout("warrior", "cleave"); err != nil {
		t.Fatalf("本职业装备应合法：%v", err)
	}
	if err := ValidateLoadout("warrior", ""); err != nil {
		t.Fatalf("空技能 = 不装备技能位，应合法：%v", err)
	}
	if err := ValidateLoadout("warrior", "firestorm"); err == nil {
		t.Fatal("跨职业技能必须被拒")
	}
	if err := ValidateLoadout("nope", "cleave"); err == nil {
		t.Fatal("未知职业必须被拒")
	}
	if err := ValidateLoadout("mage", "nope"); err == nil {
		t.Fatal("未知技能必须被拒")
	}
}

// 形状语义唯一判定（27 号票）：六形状 × (需方向/需距离)；未知形状 fail-closed。
func TestShapeSemantics(t *testing.T) {
	cases := []struct {
		shape string
		dir   bool
		dist  bool
	}{
		{ShapeMelee, false, false},  // 相邻正交 4 格：方向无关
		{ShapeBurst8, false, false}, // 3×3 全邻域：方向无关
		{ShapeSelf, false, false},   // 自身：无目标选择
		{ShapeLine, true, false},    // 线扫：仅方向（纵深由 depth 提供）
		{ShapeBand, true, false},    // 横带：仅方向（纵深由 depth 提供）
		{ShapeSnipe, true, true},    // 点狙：方向 + 距离
		{"unknown", false, false},   // 未知形状：无目标选择（fail-closed）
	}
	for _, c := range cases {
		if got := NeedsDirection(c.shape); got != c.dir {
			t.Fatalf("%s NeedsDirection=%v，期望 %v", c.shape, got, c.dir)
		}
		if got := NeedsDistance(c.shape); got != c.dist {
			t.Fatalf("%s NeedsDistance=%v，期望 %v", c.shape, got, c.dist)
		}
		sem := SemanticsForShape(c.shape)
		if sem.NeedsDirection != c.dir || sem.NeedsDistance != c.dist {
			t.Fatalf("%s SemanticsForShape 与便捷判定不一致：%+v", c.shape, sem)
		}
	}
}

// 语义携带几何参数：普攻/技能从各自表取 depth/range（engine 消费口径）。
func TestSemanticsCarryDepthAndRange(t *testing.T) {
	mage, _ := ByID("mage")
	if sem := AttackSemantics(mage); !sem.NeedsDirection || sem.NeedsDistance ||
		sem.Depth != 2 || sem.RangeMax != 0 {
		t.Fatalf("法师普攻语义应为 方向+深度2，实际 %+v", sem)
	}
	ranger, _ := ByID("ranger")
	if sem := AttackSemantics(ranger); !sem.NeedsDirection || !sem.NeedsDistance ||
		sem.Depth != 0 || sem.RangeMax != 3 {
		t.Fatalf("游侠普攻语义应为 方向+距离(≤3)，实际 %+v", sem)
	}
	pierce, ok, err := SkillByID("ranger", "pierce")
	if err != nil || !ok {
		t.Fatalf("pierce 应存在：ok=%v err=%v", ok, err)
	}
	if sem := SkillSemantics(pierce); !sem.NeedsDirection || !sem.NeedsDistance ||
		sem.Depth != 0 || sem.RangeMax != 4 {
		t.Fatalf("穿云箭语义应为 方向+距离(≤4)，实际 %+v", sem)
	}
	firestorm, ok, err := SkillByID("mage", "firestorm")
	if err != nil || !ok {
		t.Fatalf("firestorm 应存在：ok=%v err=%v", ok, err)
	}
	if sem := SkillSemantics(firestorm); !sem.NeedsDirection || sem.NeedsDistance ||
		sem.Depth != 2 || sem.RangeMax != 0 {
		t.Fatalf("烈焰风暴语义应为 方向+深度2，实际 %+v", sem)
	}
	windheal, ok, err := SkillByID("ranger", "windheal")
	if err != nil || !ok {
		t.Fatalf("windheal 应存在：ok=%v err=%v", ok, err)
	}
	if sem := SkillSemantics(windheal); sem.NeedsDirection || sem.NeedsDistance {
		t.Fatalf("风行回复不应需要目标选择，实际 %+v", sem)
	}
}

// ClampSnipeDistance：唯一钳制实现（低于 1 → 1；超射程 → 射程；rangeMax=0 不设上限）。
func TestClampSnipeDistance(t *testing.T) {
	cases := []struct {
		dist, want int
		rangeMax   int32
	}{
		{0, 1, 3},
		{-2, 1, 3},
		{2, 2, 3},
		{4, 3, 3},
		{9, 4, 4},
		{5, 5, 0},
	}
	for _, c := range cases {
		if got := ClampSnipeDistance(c.dist, c.rangeMax); got != c.want {
			t.Fatalf("ClampSnipeDistance(%d,%d)=%d，期望 %d", c.dist, c.rangeMax, got, c.want)
		}
	}
}

// AxisDirection / DirectionDelta：主轴裁决与方向增量的唯一实现。
func TestAxisDirectionAndDelta(t *testing.T) {
	cases := []struct {
		dRow, dCol int
		wantDir    string
		wantSteps  int
	}{
		{2, 1, DirDown, 2},
		{-2, 1, DirUp, 2},
		{1, 2, DirRight, 2},
		{1, -2, DirLeft, 2},
		{2, 2, DirDown, 2}, // 平局取行轴
		{0, 0, DirRight, 0},
	}
	for _, c := range cases {
		dir, steps := AxisDirection(c.dRow, c.dCol)
		if dir != c.wantDir || steps != c.wantSteps {
			t.Fatalf("AxisDirection(%d,%d)=(%s,%d)，期望 (%s,%d)",
				c.dRow, c.dCol, dir, steps, c.wantDir, c.wantSteps)
		}
	}
	for _, c := range []struct {
		dir    string
		dr, dc int
	}{
		{DirUp, -1, 0},
		{DirDown, 1, 0},
		{DirLeft, 0, -1},
		{DirRight, 0, 1},
		{"", 0, 0},
		{"diagonal", 0, 0},
	} {
		dr, dc := DirectionDelta(c.dir)
		if dr != c.dr || dc != c.dc {
			t.Fatalf("DirectionDelta(%q)=(%d,%d)，期望 (%d,%d)", c.dir, dr, dc, c.dr, c.dc)
		}
	}
}

// /api/classes：JSON 形状（客户端同源数据契约）。
func TestHTTPHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/classes", nil)
	rec := httptest.NewRecorder()
	HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("状态码 %d", rec.Code)
	}
	var body struct {
		Classes []struct {
			ID                   string `json:"id"`
			HP                   int32  `json:"hp"`
			AttackNeedsDirection bool   `json:"attackNeedsDirection"`
			AttackNeedsDistance  bool   `json:"attackNeedsDistance"`
			Skills               []struct {
				ID             string `json:"id"`
				Cooldown       int32  `json:"cooldown"`
				NeedsDirection bool   `json:"needsDirection"`
				NeedsDistance  bool   `json:"needsDistance"`
			} `json:"skills"`
		} `json:"classes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应非合法 JSON：%v", err)
	}
	if len(body.Classes) != 3 {
		t.Fatalf("应恰好 3 个可选拔职业，实际 %d", len(body.Classes))
	}
	if body.Classes[0].ID != "warrior" || body.Classes[0].Skills[0].ID != "cleave" {
		t.Fatalf("首个职业应为剑士且首个技能为巨力斩：%+v", body.Classes[0])
	}
	// 形状语义字段（27 号票）：客户端 aimShape 直接读，不再自行按形状名判断。
	byID := map[string]int{}
	for i, c := range body.Classes {
		byID[c.ID] = i
	}
	warrior, mage, ranger := body.Classes[byID["warrior"]], body.Classes[byID["mage"]], body.Classes[byID["ranger"]]
	if warrior.AttackNeedsDirection || warrior.AttackNeedsDistance {
		t.Fatalf("剑士普攻（melee）不应需要方向/距离：%+v", warrior)
	}
	if !mage.AttackNeedsDirection || mage.AttackNeedsDistance {
		t.Fatalf("法师普攻（line）应仅需方向：%+v", mage)
	}
	if !ranger.AttackNeedsDirection || !ranger.AttackNeedsDistance {
		t.Fatalf("游侠普攻（snipe）应需方向+距离：%+v", ranger)
	}
	skillByID := map[string]int{}
	for i, s := range ranger.Skills {
		skillByID[s.ID] = i
	}
	if s := ranger.Skills[skillByID["pierce"]]; !s.NeedsDirection || !s.NeedsDistance {
		t.Fatalf("穿云箭应需方向+距离：%+v", s)
	}
	if s := ranger.Skills[skillByID["windheal"]]; s.NeedsDirection || s.NeedsDistance {
		t.Fatalf("风行回复不应需要方向/距离：%+v", s)
	}
}
