// Package elo：ELO 结算纯函数（07 号票）。
// 参数集中于此：spec 附注初稿，调参改这里。
package elo

import "math"

const (
	// 初始段位分（spec 附注初稿，调参改这里）。
	Initial = 1000
	// K 系数（spec 附注初稿，调参改这里）。
	K = 32
	// 分母档（标准期望分公式常量，400）。
	Scale = 400.0
)

// Expected：标准期望分公式，a 对 b 的预期得分 ∈ (0,1)。
func Expected(eloA, eloB int) float64 {
	return 1.0 / (1.0 + math.Pow(10.0, float64(eloB-eloA)/Scale))
}

// Settle：结算一场对局，返回双方新分。
// winner 取 "p1" | "p2" | ""（空=平局）。
// 平局双方 +0（spec 附注硬规则，不做小数位摆动）：直接返回原分，
// 即使双方分差巨大也不产生小数位漂移。
func Settle(p1Elo, p2Elo int, winner string) (p1After, p2After int) {
	if winner != "p1" && winner != "p2" {
		return p1Elo, p2Elo
	}
	expP1 := Expected(p1Elo, p2Elo)
	var scoreP1 float64
	if winner == "p1" {
		scoreP1 = 1.0
	}
	p1After = int(math.Round(float64(p1Elo) + K*(scoreP1-expP1)))
	p2After = int(math.Round(float64(p2Elo) + K*((1.0-scoreP1)-(1.0-expP1))))
	return p1After, p2After
}
