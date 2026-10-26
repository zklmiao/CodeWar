package bot

import (
	"codewar/server/internal/classes"
	"codewar/server/internal/match"
)

// 方向常量按固定顺序遍历（同 seed 的候选列表稳定，移动选择可复现）。
var directions = []string{classes.DirUp, classes.DirDown, classes.DirLeft, classes.DirRight}

// PlanTargeting：按形状规划 targeting（方向/距离瞄准对手当前位置）。
// 返回 ok=false 表示该形状不需要目标选择（近战/邻域/自身）。
//
// 需方向/需距离由 classes 包唯一判定（与引擎同源，27 号票）；主轴方向复用
// classes.AxisDirection（与引擎自动瞄准同一实现）。与真人客户端相同的约束：
// 直线形状在对手斜向时只能沿主轴出手，是否命中由引擎覆盖格在结算时裁决。
// 点狙距离 = 主轴格距，钳到 [1, rangeMax]（与引擎同一钳制实现）。
func PlanTargeting(from, opp match.Pos, shape string, rangeMax int32) (match.Targeting, bool) {
	switch {
	case classes.NeedsDistance(shape):
		dir, steps := classes.AxisDirection(opp.Row-from.Row, opp.Col-from.Col)
		return match.Targeting{Direction: dir, Distance: classes.ClampSnipeDistance(steps, rangeMax)}, true
	case classes.NeedsDirection(shape):
		dir, _ := classes.AxisDirection(opp.Row-from.Row, opp.Col-from.Col)
		return match.Targeting{Direction: dir}, true
	default:
		// melee（方向无关）、burst8（无方向）、self（自身）、未知形状：走引擎自动瞄准
		return match.Targeting{}, false
	}
}

// LegalMoves：与引擎 validateMove 同语义的合法移动终点（直线正交 1..moveMax、
// 终点在网格内且不被对方占据；距离 2 时中间格不得是对方）。
// 只做规划，最终合法性仍由引擎裁决（Bot 没有绕过校验的路径）。
func LegalMoves(from, opp match.Pos, moveMax int) []match.Pos {
	var out []match.Pos
	for _, dir := range directions {
		dr, dc := classes.DirectionDelta(dir)
		for step := 1; step <= moveMax; step++ {
			to := match.Pos{Row: from.Row + dr*step, Col: from.Col + dc*step}
			if to.Row < 0 || to.Row >= match.GridSize || to.Col < 0 || to.Col >= match.GridSize {
				break // 越过边界后同方向更远的格也非法
			}
			if to == opp {
				continue // 终点被占：该格非法，但同方向更远仍可尝试（与引擎一致）
			}
			if step == 2 {
				mid := match.Pos{Row: from.Row + dr, Col: from.Col + dc}
				if mid == opp {
					continue
				}
			}
			out = append(out, to)
		}
	}
	return out
}

// PickMove：从合法终点中均匀抽取一个（同 seed 可复现）。
func PickMove(r decisionRNG, from, opp match.Pos, moveMax int) (match.Pos, bool) {
	moves := LegalMoves(from, opp, moveMax)
	if len(moves) == 0 {
		return match.Pos{}, false
	}
	return moves[r.Intn(len(moves))], true
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
