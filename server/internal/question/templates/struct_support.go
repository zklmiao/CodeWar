// 结构体类题支持（12 号票批 7）：TestCase 四件套（call/expected/setup/compare）、
// 期望值计算（链表/树规范化 repr）与 class 型 starter。
// 契约见 ADR-0004 / 设计 002 附录 E：setup 构造入参、compare 规范化 got、expected 永不进容器。
package templates

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"codewar/server/internal/match"
	"codewar/server/internal/question"
)

// structCase：带 setup/compare 的判题用例（空串 = 走现行 eval/repr 路径）。
func structCase(name, call, expected, setup, compare string) match.TestCase {
	return match.TestCase{Name: name, Call: call, Expected: expected, Setup: setup, Compare: compare}
}

// nodeCompare：链表题比较器（got 经 nodes_repr 规范化；环按 id 有界遍历）。
func nodeCompare() string { return "def _normalize(got):\n    return nodes_repr(got)" }

// treeCompare：树题比较器（got 经 tree_repr 层序规范化）。
func treeCompare() string { return "def _normalize(got):\n    return tree_repr(got)" }

// customInstance：complete-only 题的 Instance 组装（starter 由模板给出，
// 用于 class 型题：默认 RenderComplete 只适合单函数签名）。
func customInstance(meta question.Meta, statement, signature, starter, answer string, visible, hidden []match.TestCase) (question.Instance, error) {
	return question.Instance{
		Meta:            meta,
		Statement:       statement,
		Signature:       signature,
		CompleteStarter: starter,
		VisibleTests:    visible,
		HiddenTests:     hidden,
		AnswerKey:       answer,
	}, nil
}

// listSetup：链表构造声明（pos >= 0 时尾节点连回第 pos 个节点）。
func listSetup(name string, nums []int, pos int) string {
	s := fmt.Sprintf("%s = list_to_nodes(%s)", name, pyList(nums))
	if pos >= 0 {
		s += fmt.Sprintf("\nmake_cycle(%s, %d)", name, pos)
	}
	return s
}

// treeSetup：树构造声明。
func treeSetup(name string, vals []any) string {
	return fmt.Sprintf("%s = build_tree(%s)", name, pyTreeVals(vals))
}

// nodesListRepr：无环链表的规范化 repr（nodes_repr 紧凑风格，无空格）。
func nodesListRepr(nums []int) string {
	parts := make([]string, len(nums))
	for i, v := range nums {
		parts[i] = strconv.Itoa(v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// cycleRepr：环入口的规范化 repr（nodes_repr：从入口出发的环序）。
func cycleRepr(nums []int, pos int) string {
	if pos < 0 {
		return "None"
	}
	return "cycle:[" + strings.Join(intStrs(nums[pos:]), ",") + "]"
}

func intStrs(nums []int) []string {
	out := make([]string, len(nums))
	for i, v := range nums {
		out[i] = strconv.Itoa(v)
	}
	return out
}

// pyTreeVals：层序数组的 Python repr（None 占位）。
func pyTreeVals(vals []any) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		if v == nil {
			parts[i] = "None"
		} else {
			parts[i] = fmt.Sprintf("%d", v.(int))
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// pyAnyListRepr：混合类型列表的 Python repr（None/bool/int；用于调用序列期望值）。
func pyAnyListRepr(vals []any) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		switch x := v.(type) {
		case nil:
			parts[i] = "None"
		case bool:
			if x {
				parts[i] = "True"
			} else {
				parts[i] = "False"
			}
		case int:
			parts[i] = strconv.Itoa(x)
		default:
			parts[i] = fmt.Sprintf("%v", x)
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// ---- 链表期望值计算（与参考解同语义） ----

func reverseListGo(nums []int) []int {
	out := make([]int, len(nums))
	for i, v := range nums {
		out[len(nums)-1-i] = v
	}
	return out
}

func dedupeSortedGo(nums []int) []int {
	out := []int{}
	for i, v := range nums {
		if i == 0 || v != nums[i-1] {
			out = append(out, v)
		}
	}
	return out
}

func mergeSortedGo(a, b []int) []int {
	out := []int{}
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] <= b[j] {
			out = append(out, a[i])
			i++
		} else {
			out = append(out, b[j])
			j++
		}
	}
	out = append(out, a[i:]...)
	out = append(out, b[j:]...)
	return out
}

// addDigitsGo：数字逆序表示的链表加法（每位 0-9，最高位非 0）。
func addDigitsGo(a, b []int) []int {
	out := []int{}
	i, j, carry := 0, 0, 0
	for i < len(a) || j < len(b) || carry > 0 {
		x, y := 0, 0
		if i < len(a) {
			x = a[i]
			i++
		}
		if j < len(b) {
			y = b[j]
			j++
		}
		total := x + y + carry
		out = append(out, total%10)
		carry = total / 10
	}
	return out
}

func removeNthGo(nums []int, n int) []int {
	idx := len(nums) - n
	out := append([]int{}, nums[:idx]...)
	return append(out, nums[idx+1:]...)
}

func reorderGo(nums []int) []int {
	out := []int{}
	l, r := 0, len(nums)-1
	for l <= r {
		out = append(out, nums[l])
		l++
		if l <= r {
			out = append(out, nums[r])
			r--
		}
	}
	return out
}

// ---- 树结构（层序数组 ⇄ 树；与沙箱辅助库 build_tree/tree_repr 同语义） ----

type structTree struct {
	val         int
	left, right *structTree
}

func buildTreeGo(vals []any) *structTree {
	if len(vals) == 0 || vals[0] == nil {
		return nil
	}
	root := &structTree{val: vals[0].(int)}
	queue := []*structTree{root}
	i := 1
	for len(queue) > 0 && i < len(vals) {
		node := queue[0]
		queue = queue[1:]
		if i < len(vals) {
			if vals[i] != nil {
				node.left = &structTree{val: vals[i].(int)}
				queue = append(queue, node.left)
			}
			i++
		}
		if i < len(vals) {
			if vals[i] != nil {
				node.right = &structTree{val: vals[i].(int)}
				queue = append(queue, node.right)
			}
			i++
		}
	}
	return root
}

// treeReprGo：与沙箱 tree_repr 同格式（层序 JSON、null 占位、尾部裁剪）。
func treeReprGo(root *structTree) string {
	out := []any{}
	queue := []*structTree{root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if node == nil {
			out = append(out, nil)
			continue
		}
		out = append(out, node.val)
		queue = append(queue, node.left, node.right)
	}
	for len(out) > 0 && out[len(out)-1] == nil {
		out = out[:len(out)-1]
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func invertTreeGo(node *structTree) *structTree {
	if node == nil {
		return nil
	}
	node.left, node.right = invertTreeGo(node.right), invertTreeGo(node.left)
	return node
}

func maxDepthGo(node *structTree) int {
	if node == nil {
		return 0
	}
	l, r := maxDepthGo(node.left), maxDepthGo(node.right)
	if l > r {
		return l + 1
	}
	return r + 1
}

func inorderGo(node *structTree, out *[]int) {
	if node == nil {
		return
	}
	inorderGo(node.left, out)
	*out = append(*out, node.val)
	inorderGo(node.right, out)
}

func levelOrderGo(root *structTree) [][]int {
	out := [][]int{}
	current := []*structTree{}
	if root != nil {
		current = append(current, root)
	}
	for len(current) > 0 {
		row := make([]int, 0, len(current))
		nxt := []*structTree{}
		for _, n := range current {
			row = append(row, n.val)
			if n.left != nil {
				nxt = append(nxt, n.left)
			}
			if n.right != nil {
				nxt = append(nxt, n.right)
			}
		}
		out = append(out, row)
		current = nxt
	}
	return out
}

func rightViewGo(root *structTree) []int {
	out := []int{}
	current := []*structTree{}
	if root != nil {
		current = append(current, root)
	}
	for len(current) > 0 {
		out = append(out, current[len(current)-1].val)
		nxt := []*structTree{}
		for _, n := range current {
			if n.left != nil {
				nxt = append(nxt, n.left)
			}
			if n.right != nil {
				nxt = append(nxt, n.right)
			}
		}
		current = nxt
	}
	return out
}

// pyNestedInts：二维整数列表的 Python repr（用于层序遍历期望值）。
func pyNestedInts(rows [][]int) string {
	parts := make([]string, len(rows))
	for i, row := range rows {
		parts[i] = pyList(row)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
