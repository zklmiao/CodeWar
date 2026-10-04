// 槽位原语运行库（设计 002 §1）：种子化 RNG、命名池、数据构造器、
// 情境池、双形式渲染、不变量校验。全库唯一随机来源是 RNG 流。
package question

import (
	"fmt"
	"math/rand"
	"strings"
)

// RNG：种子化随机流（原语 rng.int / pick / sample / shuffle 的唯一载体）。
type RNG struct {
	r *rand.Rand
}

// NewRNG：从 int64 种子构造流。
func NewRNG(seed int64) *RNG { return &RNG{r: rand.New(rand.NewSource(seed))} }

// Int：闭区间 [min, max] 随机整数。
func (g *RNG) Int(min, max int) int {
	if max <= min {
		return min
	}
	return min + g.r.Intn(max-min+1)
}

// Pick：从候选中随机取 1 个。
func (g *RNG) Pick(cands []string) string {
	if len(cands) == 0 {
		return ""
	}
	return cands[g.r.Intn(len(cands))]
}

// PickN：从候选中随机取 n 个（可重复）。
func (g *RNG) PickN(cands []string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = g.Pick(cands)
	}
	return out
}

// Sample：从候选中不放回取 n 个（n 超限则全取后打乱）。
func (g *RNG) Sample(cands []string, n int) []string {
	cp := append([]string(nil), cands...)
	g.Shuffle(cp)
	if n > len(cp) {
		n = len(cp)
	}
	return cp[:n]
}

// Shuffle：原地打乱字符串切片。
func (g *RNG) Shuffle(s []string) {
	g.r.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
}

// ShuffleInts：原地打乱整数切片。
func (g *RNG) ShuffleInts(s []int) {
	g.r.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
}

// Bool：按概率返回 true。
func (g *RNG) Bool(p float64) bool { return g.r.Float64() < p }

// ---- 命名池（names.variable / names.function，防实例趋同） ----

var varPools = map[string][]string{
	"int":    {"n", "x", "k", "total", "count", "num", "val", "m"},
	"list":   {"nums", "arr", "items", "data", "vals", "seq"},
	"string": {"s", "text", "word", "line", "msg", "code"},
	"dict":   {"freq", "counts", "table", "mapping", "seen", "bag"},
}

var funcPool = []string{"solve", "calc", "compute", "process", "evaluate", "transform"}

// Variable：按品类取随机变量名。
func (g *RNG) Variable(kind string) string {
	pool, ok := varPools[kind]
	if !ok || len(pool) == 0 {
		pool = varPools["int"]
	}
	name := g.Pick(pool)
	// 约 1/3 概率加数字后缀，扩大实例差异。
	if g.Bool(0.33) {
		name = fmt.Sprintf("%s%d", name, g.Int(2, 9))
	}
	return name
}

// Function：取随机函数名（模板一般固定签名，本原语供仿写变体用）。
func (g *RNG) Function() string { return g.Pick(funcPool) }

// ---- 数据构造器（data.list / matrix / string） ----

// ListOpts：data.list 选项（distinct / sorted / mustContain）。
type ListOpts struct {
	Distinct    bool
	Sorted      bool
	MustContain []int
}

// IntList：构造 [size] 个 [min, max] 整数列表。
func (g *RNG) IntList(size, min, max int, opts ListOpts) []int {
	out := make([]int, 0, size)
	seen := map[int]bool{}
	for len(out) < size {
		v := g.Int(min, max)
		if opts.Distinct && seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	for _, m := range opts.MustContain {
		if !containsInt(out, m) {
			out[g.r.Intn(len(out))] = m
		}
	}
	if opts.Sorted {
		sortInts(out)
	}
	return out
}

// Matrix：构造 rows×cols 二维整数矩阵。
func (g *RNG) Matrix(rows, cols, min, max int) [][]int {
	m := make([][]int, rows)
	for i := range m {
		m[i] = g.IntList(cols, min, max, ListOpts{})
	}
	return m
}

var strAlphabets = map[string]string{
	"lower":  "abcdefghijklmnopqrstuvwxyz",
	"alpha":  "abcde",
	"binary": "ab",
	"digit":  "0123456789",
}

// Str：构造定长随机字符串。
func (g *RNG) Str(length int, alphabet string) string {
	alpha, ok := strAlphabets[alphabet]
	if !ok || alpha == "" {
		alpha = strAlphabets["lower"]
	}
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteByte(alpha[g.r.Intn(len(alpha))])
	}
	return b.String()
}

// ---- 情境池（text.scenario，MVP 静态池；LLM 皮肤只替换本原语） ----

var scenarioPools = map[string][]string{
	"words": {"arena", "battle", "code", "spell", "quest", "tower", "blade", "storm"},
	"names": {"Aria", "Bolt", "Cinder", "Dusk", "Ember", "Frost"},
}

// Scenario：取随机情境词。
func (g *RNG) Scenario(poolKey string) string {
	pool, ok := scenarioPools[poolKey]
	if !ok || len(pool) == 0 {
		pool = scenarioPools["words"]
	}
	return g.Pick(pool)
}

// ---- 双形式渲染（code.render，设计 002 §2） ----

// RenderComplete：签名 + 完整函数体骨架（待补全标记）。
func RenderComplete(signature string) string {
	return signature + "\n    # 在这里补全函数体\n    raise NotImplementedError\n"
}

// RenderFillBlank：把参考解 bodyLines 中 idx 集合挖空为 ____ 占位。
// 返回骨架代码与挖空行数；调用方保证 idx 为核心行且 ≤3。
func RenderFillBlank(signature string, bodyLines []string, blankIdx map[int]bool) (string, int) {
	var b strings.Builder
	b.WriteString(signature + "\n")
	n := 0
	for i, ln := range bodyLines {
		if blankIdx[i] {
			indent := leadingSpaces(ln)
			fmt.Fprintf(&b, "%s# ____ 填空：补全本行逻辑 ____\n", indent)
			n++
			continue
		}
		b.WriteString(ln + "\n")
	}
	return b.String(), n
}

// IsCoreLine：核心行判定（工具校验用：非空、非纯注释、非 trivial）。
// 拒绝挖花括号式假空：长度过短或仅 return 常量/字面量视为非核心。
func IsCoreLine(ln string) bool {
	t := strings.TrimSpace(ln)
	if t == "" || strings.HasPrefix(t, "#") {
		return false
	}
	if len(t) < 8 {
		return false
	}
	trivial := []string{"pass", "return None", "return True", "return False", "return 0", "return []", "return \"\""}
	for _, tr := range trivial {
		if t == tr {
			return false
		}
	}
	return true
}

func leadingSpaces(s string) string {
	n := 0
	for n < len(s) && (s[n] == ' ' || s[n] == '\t') {
		n++
	}
	return s[:n]
}

func containsInt(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func sortInts(s []int) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
