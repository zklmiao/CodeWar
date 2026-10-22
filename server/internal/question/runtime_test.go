// 槽位原语回归测试（12 号票 review）。
package question

import "testing"

// TestScenarioPoolsCovered：模板使用的池键必须存在——缺失键会静默回退到 words
// （英文词混入中文题面）。12 号票 review 发现 arena/dungeon/alchemy 三键缺失。
func TestScenarioPoolsCovered(t *testing.T) {
	keys := []string{"words", "rune", "spell", "heal", "arena", "dungeon", "alchemy"}
	g := NewRNG(1)
	for _, key := range keys {
		pool, ok := scenarioPools[key]
		if !ok || len(pool) == 0 {
			t.Fatalf("情境池 %q 缺失", key)
		}
		allowed := map[string]bool{}
		for _, w := range pool {
			allowed[w] = true
		}
		got := g.Scenario(key)
		if !allowed[got] {
			t.Fatalf("情境池 %q 回退到其他池：%q", key, got)
		}
	}
}
