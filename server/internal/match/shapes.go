// shapes.go：职业形状的覆盖格纯函数（06 号票，架构师裁决 6——本票测试重心）。
// 输入：攻击者位置、形状定义（classes 包常量）、targeting；输出：覆盖格集合。
// 判定语义：defender 结算时刻位置不在覆盖格内 = 闪避（03 号票结算位置判定）。
package match

import (
	"codewar/server/internal/classes"
)

// validDirection：方向值合法（targeting 提交与自动瞄准补齐用）。
func validDirection(dir string) bool {
	switch dir {
	case classes.DirUp, classes.DirDown, classes.DirLeft, classes.DirRight:
		return true
	}
	return false
}

// coverageCells：形状覆盖格集合（含 5×5 网格裁剪）。纯函数——形状矩阵测试逐格断言的对象。
//
//	melee（剑士普攻）：相邻正交 4 格，方向无关
//	line（法师普攻）：选定方向上距离 1..Depth 的全部格（深度 2 线扫 = 2 格）
//	snipe（游侠普攻/穿云箭/秘法汲取）：选定方向上恰 Distance 的单格（钳到 [1, RangeMax]）
//	burst8（巨力斩）：相邻 3×3 全邻域 8 格，方向无关
//	band（烈焰风暴）：选定方向上距离 1..Depth × 横向 ±1 的横带（Depth=2 → 6 格）
//	self（治疗系）：攻击者自身格
func coverageCells(from Pos, shape string, depth, rangeMax int32, t Targeting) map[Pos]bool {
	inGrid := func(p Pos) bool {
		return p.Row >= 0 && p.Row < GridSize && p.Col >= 0 && p.Col < GridSize
	}
	out := map[Pos]bool{}
	add := func(p Pos) {
		if inGrid(p) {
			out[p] = true
		}
	}
	switch shape {
	case classes.ShapeMelee:
		for _, d := range []string{classes.DirUp, classes.DirDown, classes.DirLeft, classes.DirRight} {
			dr, dc := classes.DirectionDelta(d)
			add(Pos{Row: from.Row + dr, Col: from.Col + dc})
		}
	case classes.ShapeLine:
		dr, dc := classes.DirectionDelta(t.Direction)
		for step := 1; step <= int(depth); step++ {
			add(Pos{Row: from.Row + dr*step, Col: from.Col + dc*step})
		}
	case classes.ShapeSnipe:
		dr, dc := classes.DirectionDelta(t.Direction)
		dist := classes.ClampSnipeDistance(t.Distance, rangeMax)
		add(Pos{Row: from.Row + dr*dist, Col: from.Col + dc*dist})
	case classes.ShapeBurst8:
		for dr := -1; dr <= 1; dr++ {
			for dc := -1; dc <= 1; dc++ {
				if dr != 0 || dc != 0 {
					add(Pos{Row: from.Row + dr, Col: from.Col + dc})
				}
			}
		}
	case classes.ShapeBand:
		dr, dc := classes.DirectionDelta(t.Direction)
		lr, lc := dc, dr // 横向 = 行进方向旋转 90°（up/down 的横向是左右，left/right 的横向是上下）
		for step := 1; step <= int(depth); step++ {
			base := Pos{Row: from.Row + dr*step, Col: from.Col + dc*step}
			add(base)
			add(Pos{Row: base.Row + lr, Col: base.Col + lc})
			add(Pos{Row: base.Row - lr, Col: base.Col - lc})
		}
	case classes.ShapeSelf:
		add(from)
	}
	return out
}

// autoAim：未提交 targeting 时的缺省瞄准（裁决 3）：方向取指向对手的坐标轴
// （|Δrow| ≥ |Δcol| 取行轴，否则列轴；同行/同列时取唯一非零轴），点狙距离取
// min(rangeMax, 曼哈顿距离)。非攻击者相对位形下自动瞄准可能落空——落空即闪避语义。
// 主轴裁决与决策层共用 classes.AxisDirection（27 号票单源化）。
func (e *Engine) autoAim(p Player, rangeMax int32) Targeting {
	me, opp := e.pos[p], e.pos[other(p)]
	dRow, dCol := opp.Row-me.Row, opp.Col-me.Col
	dir, _ := classes.AxisDirection(dRow, dCol)
	dist := int(rangeMax)
	if m := abs(dRow) + abs(dCol); m < dist {
		dist = m
	}
	if dist < 1 {
		dist = 1
	}
	return Targeting{Direction: dir, Distance: dist}
}

// effectiveTargeting：玩家提交的 targeting 优先，缺口用自动瞄准补齐；
// 需方向/需距离由形状语义唯一判定（classes.TargetingSemantics），距离钳到
// [1, rangeMax]（needsDistance 的形状才带距离语义）。
func (e *Engine) effectiveTargeting(p Player, sem classes.TargetingSemantics) Targeting {
	t := e.targeting[p]
	auto := e.autoAim(p, sem.RangeMax)
	if sem.NeedsDirection && !validDirection(t.Direction) {
		t.Direction = auto.Direction
	}
	if !sem.NeedsDirection {
		t.Direction = ""
	}
	if !sem.NeedsDistance {
		t.Distance = 0
		return t
	}
	if t.Distance < 1 {
		t.Distance = auto.Distance
	}
	t.Distance = classes.ClampSnipeDistance(t.Distance, sem.RangeMax)
	return t
}
