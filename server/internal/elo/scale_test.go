package elo

import "testing"

// ===== 难度风险系数 × ELO delta（05 号票）=====

// delta × coeff 四舍五入：胜方 delta 随档位放大/缩小，负 delta 对称缩放。
func TestScaleDelta(t *testing.T) {
	cases := []struct {
		delta int
		coeff float64
		want  int
	}{
		{16, 1.0, 16},    // 缺省锚点：不变
		{16, 0.85, 14},   // round(13.6)
		{16, 1.2, 19},    // round(19.2)
		{-16, 0.85, -14}, // 负 delta 对称（half away from zero）
		{-16, 1.2, -19},
		{0, 1.2, 0},     // 平局 delta=0：与任何系数相乘仍 +0（spec 附注硬规则）
		{0, 0.85, 0},
		{7, 0.85, 6}, // round(5.95) = 6
	}
	for _, c := range cases {
		if got := ScaleDelta(c.delta, c.coeff); got != c.want {
			t.Errorf("ScaleDelta(%d, %v) = %d, want %d", c.delta, c.coeff, got, c.want)
		}
	}
}

// Settle 本身不动（07 号票语义保持），系数在 delta 层施加——
// 回归锚：同分 p1 胜 = +16/−16（K=32, exp=0.5）。
func TestSettleUnchangedBaseline(t *testing.T) {
	a, b := Settle(1000, 1000, "p1")
	if a != 1016 || b != 984 {
		t.Fatalf("Settle 回归破坏：%d / %d，应为 1016 / 984", a, b)
	}
	a, b = Settle(1000, 1000, "")
	if a != 1000 || b != 1000 {
		t.Fatalf("平局应 +0：%d / %d", a, b)
	}
}
