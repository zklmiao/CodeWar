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
			ID     string `json:"id"`
			HP     int32  `json:"hp"`
			Skills []struct {
				ID       string `json:"id"`
				Cooldown int32  `json:"cooldown"`
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
}
