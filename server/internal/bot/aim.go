package bot

import (
	"codewar/server/internal/classes"
	"codewar/server/internal/match"
)

// 方向常量按固定顺序遍历（同 seed 的候选列表稳定，移动选择可复现）。
var directions = []string{classes.DirUp, classes.DirDown, classes.DirLeft, classes.DirRight}

// aimAxis：把相对位置折算为"朝对手的主轴方向 + 该轴格距"。
// 语义与引擎自动瞄准一致（|Δrow| ≥ |Δcol| 取行轴，否则列轴；同轴时取唯一非零轴）。
func aimAxis(from, opp match.Pos) (dir string, steps int) {
	dr, dc := opp.Row-from.Row, opp.Col-from.Col
	switch {
	case dr > 0 && abs(dr) >= abs(dc):
		return classes.DirDown, abs(dr)
	case dr < 0 && abs(dr) >= abs(dc):
		return classes.DirUp, abs(dr)
	case dc > 0:
		return classes.DirRight, abs(dc)
	case dc < 0:
		return classes.DirLeft, abs(dc)
	}
	return classes.DirRight, 0
}

// PlanTargeting：按形状规划 targeting（方向/距离瞄准对手当前位置）。
// 返回 ok=false 表示该形状不需要目标选择（近战/邻域/自身）。
//
// 与真人客户端相同的约束：直线形状在对手斜向时只能沿主轴出手（自动瞄准同语义，
// 是否命中由引擎覆盖格在结算时裁决）。点狙距离钳到 [1, rangeMax]。
func PlanTargeting(from, opp match.Pos, shape string, rangeMax int32) (match.Targeting, bool) {
	dir, steps := aimAxis(from, opp)
	switch shape {
	case classes.ShapeSnipe:
		dist := steps
		if dist < 1 {
			dist = 1
		}
		if rangeMax > 0 && dist > int(rangeMax) {
			dist = int(rangeMax)
		}
		return match.Targeting{Direction: dir, Distance: dist}, true
	case classes.ShapeLine, classes.ShapeBand:
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
		dr, dc := delta(dir)
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

func delta(dir string) (int, int) {
	switch dir {
	case classes.DirUp:
		return -1, 0
	case classes.DirDown:
		return 1, 0
	case classes.DirLeft:
		return 0, -1
	case classes.DirRight:
		return 0, 1
	}
	return 0, 0
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
