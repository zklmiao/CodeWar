package question_test

import (
	"strings"
	"testing"

	"codewar/server/internal/match"
	"codewar/server/internal/question"
	"codewar/server/internal/question/templates"
)

// ===== 题型过滤端到端语义（05 号票验收 2）=====
// 攻击题只从所选题型抽取（策略抽出的难度格为空时就近平移难度，不放弃题型）；
// 移动/防御题跨题型不受限。注册全量真实模板断言（非桩）。
// 注：templates 依赖 question，本文件用外部测试包规避 import 环。

// idsOf：全量注册表中指定题型+动作的模板 ID 集合。
func idsOf(topic question.Topic, action question.Action) map[string]bool {
	out := map[string]bool{}
	for _, tpl := range templates.All() {
		m := tpl.Meta()
		if m.Topic == topic && m.Action == action {
			out[m.ID] = true
		}
	}
	return out
}

// templateID：Question ID 去掉 @seed[:form] 后缀的模板段。
func templateID(qid string) string {
	if i := strings.IndexByte(qid, '@'); i >= 0 {
		return qid[:i]
	}
	return qid
}

// newTopicFactory：注册全量真实模板、固定 seed（断言可复现）。
func newTopicFactory(cfg question.Config) *question.Factory {
	f := question.NewFactory(cfg)
	f.RegisterAll(templates.All()...)
	return f
}

// topic=hash + cap L3 + 爬坡：10 回合攻击题全部命中哈希题型——
// 回归锚点：round1 爬坡落 L1（哈希无 L1 攻击模板），修复前会回退为跨题型
// 泄出非哈希题（05 验收 2 的遍历断言在此钉住就近平移难度语义）。
func TestAttackTopicFilterStrict(t *testing.T) {
	f := newTopicFactory(question.Config{Topics: []question.Topic{question.TopicHash}, MaxDifficulty: question.L3, Strategy: question.StrategyClimb, MatchSeed: "topic-strict"})
	hashAttacks := idsOf(question.TopicHash, question.ActionAttack)
	if len(hashAttacks) == 0 {
		t.Fatal("注册表缺少哈希攻击模板")
	}
	for round := 1; round <= 10; round++ {
		tpl := templateID(f.AttackQuestion(round).ID)
		if !hashAttacks[tpl] {
			t.Fatalf("round %d 攻击题 %q 越出哈希题型（允许集 %v）", round, tpl, hashAttacks)
		}
	}
}

// topic=sort + cap L5 + 均匀：40 次抽题全部命中排序题型；难度格空（排序无
// L1/L5 模板）时就近平移落在 [L2, L4]，绝不越出题型、也绝不报错占位题。
func TestAttackTopicFilterUniformCapL5(t *testing.T) {
	f := newTopicFactory(question.Config{Topics: []question.Topic{question.TopicSort}, MaxDifficulty: question.L5, Strategy: question.StrategyUniform, MatchSeed: "topic-uniform"})
	sortAttacks := idsOf(question.TopicSort, question.ActionAttack)
	seen := map[question.Difficulty]bool{}
	for i := 1; i <= 40; i++ {
		q := f.AttackQuestion(i)
		tpl := templateID(q.ID)
		if !sortAttacks[tpl] {
			t.Fatalf("第 %d 次攻击题 %q 越出排序题型", i, tpl)
		}
		// 从模板 ID 尾段还原难度（ID 约定 <topic>-<name>-L<n>-attack）
		for d := question.L1; d <= question.L5; d++ {
			if strings.Contains(tpl, "-"+d.String()+"-") {
				seen[d] = true
			}
		}
	}
	if seen[question.L1] || seen[question.L5] {
		t.Fatalf("排序题只应在 [L2, L4] 落档（空格就近平移），实际命中 %v", seen)
	}
	if len(seen) < 2 {
		t.Fatalf("40 次抽题难度应覆盖多档（均匀语义），实际 %v", seen)
	}
}

// 题型过滤只约束共享攻击题：hash 过滤下移动/防御题跨题型抽取（10 回合内
// 覆盖 ≥2 个题型，固定 seed 确定性成立）。
func TestActionQuestionsCrossTopic(t *testing.T) {
	f := newTopicFactory(question.Config{Topics: []question.Topic{question.TopicHash}, MaxDifficulty: question.L3, Strategy: question.StrategyClimb, MatchSeed: "topic-cross"})
	moveTopics, defendTopics := map[question.Topic]bool{}, map[question.Topic]bool{}
	byID := map[string]question.Meta{}
	for _, tpl := range templates.All() {
		byID[tpl.Meta().ID] = tpl.Meta()
	}
	for round := 1; round <= 10; round++ {
		moveTopics[byID[templateID(f.MoveQuestion(round, "p1").ID)].Topic] = true
		defendTopics[byID[templateID(f.DefendQuestion(round, "p1").ID)].Topic] = true
	}
	if len(moveTopics) < 2 || len(defendTopics) < 2 {
		t.Fatalf("行动题应跨题型（不受过滤限制）：move=%v defend=%v", moveTopics, defendTopics)
	}
}

// cap=L5 时行动题不得落空成错误占位题：移动/防御模板只到 L3，策略抽到
// L4/L5 时就近平移到最近可用档（05 号票浏览器验收发现的回归——cap 真实
// 生效后行动题首次能抽到 L4/L5，空格必须回退而非把占位题推进题面板）。
func TestActionQuestionsNeverPlaceholderAtCapL5(t *testing.T) {
	for _, strategy := range []string{question.StrategyUniform, question.StrategyClimb} {
		f := newTopicFactory(question.Config{MaxDifficulty: question.L5, Strategy: strategy, MatchSeed: "action-cap5-" + strategy})
		for round := 1; round <= 20; round++ {
			for _, p := range []match.Player{"p1", "p2"} {
				for _, q := range []match.Question{
					f.MoveQuestion(round, p),
					f.DefendQuestion(round, p),
				} {
					if strings.HasPrefix(templateID(q.ID), "error") {
						t.Fatalf("strategy=%s round=%d player=%s 行动题落空：%q", strategy, round, p, q.ID)
					}
				}
			}
		}
	}
}

// cap=L5 + 技能位：贴上限语义下就近回退——现役技能模板仅 L3，cap=L5 应回
// 落 L3 而非错误占位（与 06 号票 pinned 行为一致的端到端面）。
func TestSkillQuestionNearestFallbackAtCapL5(t *testing.T) {
	f := newTopicFactory(question.Config{MaxDifficulty: question.L5, Strategy: question.StrategyClimb, MatchSeed: "skill-cap5"})
	q := f.SkillQuestion(1, "p1")
	if strings.HasPrefix(templateID(q.ID), "error") {
		t.Fatalf("cap=L5 技能题应回退到最近可用档，实际 %q", q.ID)
	}
	if !strings.Contains(templateID(q.ID), "-L3-") {
		t.Fatalf("现役技能模板仅 L3，应回退到 L3：%q", q.ID)
	}
}

// 真空格保底：topic=hash + cap L1——整个区间无哈希攻击模板（12 号票补齐前
// 的语义空格），退回跨题型（行动题同语义），且必须给出可用题而非错误占位。
func TestAttackTopicEmptyPoolFallsBackCrossTopic(t *testing.T) {
	f := newTopicFactory(question.Config{Topics: []question.Topic{question.TopicHash}, MaxDifficulty: question.L1, Strategy: question.StrategyClimb, MatchSeed: "topic-empty"})
	for round := 1; round <= 3; round++ {
		q := f.AttackQuestion(round)
		tpl := templateID(q.ID)
		if strings.HasPrefix(tpl, "error") {
			t.Fatalf("真空格应跨题型保底而非错误占位：%q", q.ID)
		}
		if idsOf(question.TopicHash, question.ActionAttack)[tpl] {
			t.Fatalf("cap=L1 不可能存在哈希攻击模板：%q", tpl)
		}
	}
}
