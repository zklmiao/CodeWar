// 槽位原语运行库（设计 002 §1）：种子化 RNG、数据构造器、
// 情境池、双形式渲染、不变量校验。全库唯一随机来源是 RNG 流。
package question

import (
	"fmt"
	"math/rand"
	"strings"
)

// RNG：种子化随机流（原语 rng.int / pick 的唯一载体）。
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

// Bool：按概率返回 true。
func (g *RNG) Bool(p float64) bool { return g.r.Float64() < p }

// ---- 数据构造器（data.list / string） ----

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

var strAlphabets = map[string]string{
	"lower":  "abcdefghijklmnopqrstuvwxyz",
	"alpha":  "abcde",
	"binary": "ab",
	"digit":  "0123456789",
	"moves":  "UDLR",
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
	// 12 号票扩容：按题材域扩充（text.scenario 原语不变，MVP 静态池；LLM 皮肤后续只替换本池）。
	"rune":  {"符文", "咒文", "铭文"},
	"spell": {"法力", "奥术", "元素"},
	"heal":  {"圣光", "生命之泉", "治疗波"},
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
