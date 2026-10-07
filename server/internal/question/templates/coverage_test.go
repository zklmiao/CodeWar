// 12 号票覆盖工程（本票首批）：模板矩阵覆盖地图 + rubric 机械门禁。
// 口径（设计 002 §7）：6 题型 × 5 难度 × 4 动作 = 120 格；语义空格 12 格（emptyRows）→ 有效 108；
// 关键格 = 攻击 × L3-L5 的有效格（16 个）各配 ≥2 模板；目标总量 108 + 16 = 124（≈ §7 的 ≈126）。
//
// 分批推进协议：每批新增模板必须在同一提交内收缩 expectedGaps（棘轮）——
// 测试双向断言「无计划外缺口」与「无过期条目」；全部批次完成后 expectedGaps 清空，
// 本测试即验收 1（每格 ≥1 模板、关键格 ≥2）的常驻证据。
package templates

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"codewar/server/internal/question"
)

// cellKey：矩阵格（题型 × 难度 × 动作）。攻击模板 Form=both 一模板双渲染（设计 002 §2），
// 私有动作模板 Form=complete；形式维不单独计格。
type cellKey struct {
	topic  question.Topic
	diff   question.Difficulty
	action question.Action
}

func (c cellKey) String() string {
	return fmt.Sprintf("%s-%s-%s", c.topic, c.diff, c.action)
}

// emptyRows：语义空格白名单——Rubric（设计 002 §4）无法给出语义的 (题型 × 难度) 行，
// 对 4 种动作同等成立（3 行 × 4 动作 = 12 格，120 − 12 = 108）：
//   - (recursion, L1)：递归题至少需要「基例 + 递归步」两个结构，超出 L1「1 个语法点 / ≤3 行」上限
//     （设计 002 §7 点名；04 号票 MANIFEST 已声明）。
//   - (recursion, L5)：「递归与简单 DP」题型定位为单一递归/入门 DP 思想；L5 要求 ≥2 思想组合、
//     参考解 12-18 行，超出「简单 DP」定位（04 号票 MANIFEST 声明，本票维持）。
//   - (syntax, L5)：纯语法无法承载 ≥2 算法思想组合（设计 002 §7 点名）。
//
// 新增白名单条目必须在此写明理由并同步设计文档（测试纪律：语义空格白名单须注释理由）。
var emptyRows = map[question.Topic]map[question.Difficulty]string{
	question.TopicRecursion: {
		question.L1: "递归需基例+递归步两个结构，超出 L1「1 语法点/≤3 行」上限（设计 002 §7 点名）",
		question.L5: "「简单 DP」定位为单一思想；L5 需 ≥2 思想组合 12-18 行，超出题型定位（04 号票 MANIFEST 声明）",
	},
	question.TopicSyntax: {
		question.L5: "纯语法无法承载 ≥2 算法思想组合（设计 002 §7 点名）",
	},
}

var allDifficulties = []question.Difficulty{question.L1, question.L2, question.L3, question.L4, question.L5}

var allActions = []question.Action{question.ActionAttack, question.ActionMove, question.ActionDefend, question.ActionSkill}

// validCells：合法格全集 = 120 − 12 语义空格 = 108。
func validCells(t *testing.T) map[cellKey]bool {
	t.Helper()
	cells := map[cellKey]bool{}
	for _, topic := range question.AllTopics {
		for _, d := range allDifficulties {
			if _, empty := emptyRows[topic][d]; empty {
				continue
			}
			for _, action := range allActions {
				cells[cellKey{topic, d, action}] = true
			}
		}
	}
	if len(cells) != 108 {
		t.Fatalf("有效格口径漂移：应为 108，实际 %d", len(cells))
	}
	return cells
}

// keyCells：关键格 = 攻击 × L3-L5 的有效格（语法/递归无 L5）→ 16 个，要求 ≥2 模板。
func keyCells(t *testing.T) map[cellKey]bool {
	t.Helper()
	keys := map[cellKey]bool{}
	for cell := range validCells(t) {
		if cell.action == question.ActionAttack && cell.diff >= question.L3 {
			keys[cell] = true
		}
	}
	if len(keys) != 16 {
		t.Fatalf("关键格口径漂移：应为 16，实际 %d", len(keys))
	}
	return keys
}

// coverageByCell：注册表按格计数。
func coverageByCell(t *testing.T) map[cellKey]int {
	t.Helper()
	byCell := map[cellKey]int{}
	for _, tm := range All() {
		m := tm.Meta()
		byCell[cellKey{m.Topic, m.Difficulty, m.Action}]++
	}
	return byCell
}

// expectedGaps：棘轮清单——当前尚未填上的格与关键格副本（12 号票分批收缩至空）。
// 格缺口 = "<topic>-<L#>-<action>"；关键格副本缺口 = 其后缀 "#2"。
var expectedGaps = []string{
	// 攻击格缺口（批3a 后 6）
	"array-L1-attack", "array-L5-attack",
	"hash-L1-attack", "hash-L5-attack",
	"sort-L1-attack", "sort-L5-attack",
	// 攻击关键格副本缺口（批3a 后 11）
	"array-L3-attack#2", "array-L4-attack#2", "array-L5-attack#2",
	"hash-L3-attack#2", "hash-L4-attack#2", "hash-L5-attack#2",
	"sort-L3-attack#2", "sort-L4-attack#2", "sort-L5-attack#2",
	"recursion-L3-attack#2", "recursion-L4-attack#2",
	// 移动格缺口（批3a 后 17）
	"array-L1-move", "array-L3-move", "array-L4-move", "array-L5-move",
	"hash-L1-move", "hash-L2-move", "hash-L3-move", "hash-L4-move", "hash-L5-move",
	"sort-L1-move", "sort-L2-move", "sort-L3-move", "sort-L4-move", "sort-L5-move",
	"recursion-L2-move", "recursion-L3-move", "recursion-L4-move",
	// 防御格缺口（批3b 后 16）
	"array-L1-defend", "array-L3-defend", "array-L4-defend", "array-L5-defend",
	"hash-L1-defend", "hash-L2-defend", "hash-L4-defend", "hash-L5-defend",
	"sort-L1-defend", "sort-L2-defend", "sort-L3-defend", "sort-L4-defend", "sort-L5-defend",
	"recursion-L2-defend", "recursion-L3-defend", "recursion-L4-defend",
	// 技能格缺口（批3b 后 17）
	"array-L1-skill", "array-L3-skill", "array-L4-skill", "array-L5-skill",
	"hash-L1-skill", "hash-L2-skill", "hash-L4-skill", "hash-L5-skill",
	"sort-L1-skill", "sort-L2-skill", "sort-L3-skill", "sort-L4-skill", "sort-L5-skill",
	"recursion-L2-skill", "recursion-L3-skill", "recursion-L4-skill",
}

// TestCoverageGapReport：缺口报告 + 棘轮（验收 1 的证据链）。
// 实际缺口必须 ⊆ expectedGaps（无计划外缺口），expectedGaps 必须 ⊆ 实际缺口（无过期条目）。
func TestCoverageGapReport(t *testing.T) {
	valid := validCells(t)
	keys := keyCells(t)
	byCell := coverageByCell(t)

	expected := map[string]bool{}
	for _, g := range expectedGaps {
		if expected[g] {
			t.Fatalf("expectedGaps 重复条目：%s", g)
		}
		expected[g] = true
	}

	var actual []string
	coveredCells, doubledKeys := 0, 0
	for cell := range valid {
		n := byCell[cell]
		switch {
		case n == 0:
			actual = append(actual, cell.String())
		default:
			coveredCells++
		}
		if keys[cell] {
			if n >= 2 {
				doubledKeys++
			} else {
				actual = append(actual, cell.String()+"#2")
			}
		}
	}
	sort.Strings(actual)
	actualSet := map[string]bool{}
	for _, g := range actual {
		actualSet[g] = true
	}

	var unexpected, stale []string
	for _, g := range actual {
		if !expected[g] {
			unexpected = append(unexpected, g)
		}
	}
	for g := range expected {
		if !actualSet[g] {
			stale = append(stale, g)
		}
	}
	for _, g := range unexpected {
		t.Errorf("计划外缺口：%s（该格无模板或关键格未配双模板——若属白名单调整，先更新 emptyRows 并注释理由）", g)
	}
	for _, g := range stale {
		t.Errorf("棘轮过期：%s 已填上，请从 expectedGaps 移除（每批同提交收缩）", g)
	}

	t.Logf("矩阵覆盖：%d/108 格 | 关键格双模板 %d/16 | 模板总数 %d/124 | 剩余缺口 %d",
		coveredCells, doubledKeys, len(All()), len(actual))
}

// TestRubricAndFormGates：rubric 机械门禁（设计 002 §4 T 维下限）+ 形式约定 + 空格禁区。
func TestRubricAndFormGates(t *testing.T) {
	for _, tm := range All() {
		m := tm.Meta()
		// 形式约定：攻击 = both（双渲染），私有动作 = complete（引擎以 complete 消费）。
		if m.Action == question.ActionAttack && m.Form != question.FormBoth {
			t.Errorf("模板 %s：攻击题必须 Form=both", m.ID)
		}
		if m.Action != question.ActionAttack && m.Form != question.FormComplete {
			t.Errorf("模板 %s：私有动作题必须 Form=complete", m.ID)
		}
		// 陷阱下限：L4 ≥2、L5 ≥3（上界不机械化——L3 历史模板存在 3 陷阱，rubric 上界仅作参考）。
		minTraps := 0
		switch m.Difficulty {
		case question.L4:
			minTraps = 2
		case question.L5:
			minTraps = 3
		}
		if len(m.Traps) < minTraps {
			t.Errorf("模板 %s：%v 声明陷阱 %d 个，rubric 要求 ≥%d", m.ID, m.Difficulty, len(m.Traps), minTraps)
		}
		// 语义空格禁区：白名单格不得出现模板。
		if why, empty := emptyRows[m.Topic][m.Difficulty]; empty {
			t.Errorf("模板 %s 落在语义空格（%s×%s）：%s", m.ID, m.Topic, m.Difficulty, why)
		}
	}
}

// TestFactoryDrawAllFilledCells：全矩阵出题抽样（验收 3 的「出题」半边）——
// 每个已填格经 Factory.Create 真实出题 × 3 seed；「判题」半边由
// TestSmokeAnswerKeyPassesOwnTests 全模板参考解冒烟覆盖。
func TestFactoryDrawAllFilledCells(t *testing.T) {
	f := question.NewFactory(question.Config{
		MaxDifficulty: question.L5, Strategy: question.StrategyUniform, MatchSeed: "coverage-draw",
	})
	f.RegisterAll(All()...)
	skip := map[string]bool{}
	for _, g := range expectedGaps {
		if !strings.HasSuffix(g, "#2") { // 仅格缺口跳过；副本缺口说明格已可抽题
			skip[g] = true
		}
	}
	n := 0
	for cell := range validCells(t) {
		if skip[cell.String()] {
			continue
		}
		spec := question.Spec{Topic: cell.topic, Difficulty: cell.diff, Action: cell.action}
		for _, s := range []string{"draw-1", "draw-2", "draw-3"} {
			if _, err := f.Create(spec, s); err != nil {
				t.Fatalf("格 %s 出题失败（seed=%s）：%v", cell, s, err)
			}
		}
		n++
	}
	if n == 0 {
		t.Fatal("无已填格可抽样")
	}
	t.Logf("全矩阵出题抽样：%d 格 × 3 seed 通过", n)
}
