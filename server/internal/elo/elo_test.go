package elo

import "testing"

// ELO 纯函数全表驱动：胜/负/平 × 分差边界（同分、大分差、小分差）；
// 平局严格 +0（spec 附注硬规则：即使分差巨大也不摆动）。
func TestSettle(t *testing.T) {
	cases := []struct {
		name             string
		p1, p2           int
		winner           string
		wantP1, wantP2   int
	}{
		{"同分-p1胜", 1000, 1000, "p1", 1016, 984},
		{"同分-p2胜", 1000, 1000, "p2", 984, 1016},
		{"同分-平局", 1000, 1000, "", 1000, 1000},
		{"小分差-高分者胜", 1100, 1000, "p1", 1112, 988},
		{"小分差-低分者胜", 1100, 1000, "p2", 1080, 1020},
		{"大分差-高分者胜", 1600, 1000, "p1", 1601, 999},
		{"大分差-低分者爆冷", 1600, 1000, "p2", 1569, 1031},
		{"大分差-平局严格+0", 1600, 1000, "", 1600, 1000},
		{"低分-平局严格+0", 800, 1200, "", 800, 1200},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotP1, gotP2 := Settle(c.p1, c.p2, c.winner)
			if gotP1 != c.wantP1 || gotP2 != c.wantP2 {
				t.Fatalf("Settle(%d,%d,%q)=(%d,%d)，期望 (%d,%d)",
					c.p1, c.p2, c.winner, gotP1, gotP2, c.wantP1, c.wantP2)
			}
		})
	}
}

// 小分差低分者胜的期望值：exp(1100 vs 1000)=0.640065…，
// p1 失 32×0.640065≈20.48→20，p2 得 20。上面表格已覆盖，此处只锁期望分公式本身。
func TestExpected(t *testing.T) {
	if got := Expected(1000, 1000); got != 0.5 {
		t.Fatalf("同分期望应为 0.5，实际 %v", got)
	}
	if got := Expected(1600, 1000); got < 0.96 || got > 0.98 {
		t.Fatalf("大分差高分方期望应在 0.96~0.98，实际 %v", got)
	}
	// 对称性：双方期望之和为 1
	a, b := Expected(1234, 987), Expected(987, 1234)
	if a+b < 0.999999 || a+b > 1.000001 {
		t.Fatalf("期望分应对称求和为 1，实际 %v+%v", a, b)
	}
}
