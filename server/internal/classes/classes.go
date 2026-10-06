// Package classes：职业数据单一来源（设计 003 §2/§4，架构师裁决 1）。
// 引擎的伤害/形状/移动距离/技能表全部读本包；客户端经 GET /api/classes
// 拉取同源 JSON（防双端漂移）。数值是设计 003 表的字面值——playtest 调参
// 修订设计表后同步本文件，禁止在引擎里另设平衡值。
package classes

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// 普攻/技能形状（覆盖格计算的输入；几何实现归 match 包，本包只存定义）。
const (
	ShapeMelee  = "melee"  // 相邻正交 4 格（剑士普攻）
	ShapeLine   = "line"   // 选定方向上距离 1..Depth 的全部格（法师普攻深度 2 线扫）
	ShapeSnipe  = "snipe"  // 选定方向上恰好 Distance 的单格（游侠点狙/穿云箭/秘法汲取）
	ShapeBurst8 = "burst8" // 相邻 3×3 全邻域 8 格，无需方向（巨力斩）
	ShapeBand   = "band"   // 选定方向上距离 1..Depth × 横向 ±1 的横带（烈焰风暴，6 格）
	ShapeSelf   = "self"   // 以自身为目标（治疗系）
)

// 技能种类（spec 开放问题 2：MVP 仅伤害/治疗双系，控制系明确排除）。
const (
	KindDamage = "damage"
	KindHeal   = "heal"
	KindDrain  = "drain" // 伤害+自疗复合（秘法汲取），两段都在本表字面值内
)

// Direction：targeting 的方向值（proto SetTargeting.direction 同值）。
const (
	DirUp    = "up"
	DirDown  = "down"
	DirLeft  = "left"
	DirRight = "right"
)

// Skill：一个主动技能（设计 003 §4 表的一行）。
type Skill struct {
	ID       string `json:"id"`       // 协议内引用名（JoinRequest.skill / 快照 skill_id）
	Name     string `json:"name"`     // 展示名
	Kind     string `json:"kind"`     // damage | heal | drain
	Power    int32  `json:"power"`    // 伤害类：字面固定伤害（不乘时间系数，裁决 5）
	Heal     int32  `json:"heal"`     // heal：治疗量；drain：命中后自疗量（溢出截断）
	Cooldown int32  `json:"cooldown"` // 冷却回合数（第 N 回合释放 → 第 N+Cooldown 回合起可用）
	Shape    string `json:"shape"`    // Shape* 常量
	Depth    int32  `json:"depth"`    // line/band 的纵深（0 = 不适用）
	RangeMax int32  `json:"rangeMax"` // snipe 的最大可选距离（0 = 不适用）
}

// Class：一个职业（设计 003 §2 表的一列）。
type Class struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	HP           int32   `json:"hp"`
	AttackDamage int32   `json:"attackDamage"`
	AttackShape  string  `json:"attackShape"` // Shape* 常量
	AttackDepth  int32   `json:"attackDepth"` // line 形状的纵深（法师 = 2）
	AttackRange  int32   `json:"attackRange"` // snipe 的最大射程（游侠 = 3）
	MoveMax      int32   `json:"moveMax"`     // 移动题可移动的最大正交距离
	Color        string  `json:"color"`       // 职业主色（设计 003 §6 占位区分规范）
	Skills       []Skill `json:"skills"`      // 可装备技能集（MVP 全解锁，局前选 1 装备）
}

// registry：设计 003 §2/§4 的字面值。
//
//	剑士 HP110 攻26 相邻1格 移动≤2｜巨力斩 伤55 邻域8格 CD4；求生本能 治30 CD3
//	法师 HP90  攻18 直线深度2 移动≤2｜烈焰风暴 伤40 横带6格 CD3；秘法汲取 伤25+自疗20 射程≤2 CD3
//	游侠 HP95  攻20 点狙射程3 移动≤3｜穿云箭 伤45 射程4 CD4；风行回复 治28 CD2
var registry = []Class{
	{
		ID: "warrior", Name: "剑士", HP: 110, AttackDamage: 26,
		AttackShape: ShapeMelee, MoveMax: 2, Color: "#B5482A",
		Skills: []Skill{
			{ID: "cleave", Name: "巨力斩", Kind: KindDamage, Power: 55, Cooldown: 4, Shape: ShapeBurst8},
			{ID: "survive", Name: "求生本能", Kind: KindHeal, Heal: 30, Cooldown: 3, Shape: ShapeSelf},
		},
	},
	{
		ID: "mage", Name: "法师", HP: 90, AttackDamage: 18,
		AttackShape: ShapeLine, AttackDepth: 2, MoveMax: 2, Color: "#7C5CBF",
		Skills: []Skill{
			{ID: "firestorm", Name: "烈焰风暴", Kind: KindDamage, Power: 40, Cooldown: 3, Shape: ShapeBand, Depth: 2},
			{ID: "drain", Name: "秘法汲取", Kind: KindDrain, Power: 25, Heal: 20, Cooldown: 3, Shape: ShapeSnipe, RangeMax: 2},
		},
	},
	{
		ID: "ranger", Name: "游侠", HP: 95, AttackDamage: 20,
		AttackShape: ShapeSnipe, AttackRange: 3, MoveMax: 3, Color: "#3E8E5A",
		Skills: []Skill{
			{ID: "pierce", Name: "穿云箭", Kind: KindDamage, Power: 45, Cooldown: 4, Shape: ShapeSnipe, RangeMax: 4},
			{ID: "windheal", Name: "风行回复", Kind: KindHeal, Heal: 28, Cooldown: 2, Shape: ShapeSelf},
		},
	},
}

// legacy：01~03 号票的既定行为锚点（HP100 / 攻22 / 相邻1格 / 移动≤2 / 无技能）。
// NewEngine 零信息构造与匿名对局的缺省——保证既有测试与匿名链路零改动全绿。
var legacy = Class{
	ID: "legacy", Name: "练习傀儡", HP: 100, AttackDamage: 22,
	AttackShape: ShapeMelee, MoveMax: 2, Color: "#8a93a8",
}

// All：全部可选拔职业（/api/classes 的数据源）。
func All() []Class { return append([]Class(nil), registry...) }

// ByID：按 ID 查职业；legacy 仅经 Legacy() 获取，不进可选清单。
func ByID(id string) (Class, bool) {
	for _, c := range registry {
		if c.ID == id {
			return c, true
		}
	}
	return Class{}, false
}

// Legacy：向后兼容锚点（裁决 2 的实现载体：既有测试与匿名对局零改动）。
func Legacy() Class { return legacy }

// SkillByID：在职业的技能集内查已装备技能；未装备（空 skillID）→ ok 且零值。
// 职业内不存在的 skillID → false（hub 在 join 时拒绝，fail-closed）。
func SkillByID(classID, skillID string) (Skill, bool, error) {
	c, ok := ByID(classID)
	if !ok {
		return Skill{}, false, fmt.Errorf("未知职业 %q", classID)
	}
	if skillID == "" {
		return Skill{}, false, nil
	}
	for _, s := range c.Skills {
		if s.ID == skillID {
			return s, true, nil
		}
	}
	return Skill{}, false, fmt.Errorf("职业 %q 无技能 %q", classID, skillID)
}

// ValidateLoadout：join 时校验职业+技能装备组合（hub fail-closed 用）。
// 空 skillID = 不装备技能位，合法（匿名/极简接入路径）。
func ValidateLoadout(classID, skillID string) error {
	_, ok, err := SkillByID(classID, skillID)
	if err != nil {
		return err
	}
	if skillID != "" && !ok {
		return fmt.Errorf("职业 %q 无技能 %q", classID, skillID)
	}
	return nil
}

// HTTPHandler：GET /api/classes —— 客户端职业选择的同源数据（裁决 1）。
func HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"classes": All()})
	}
}
