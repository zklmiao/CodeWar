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
// 回归锚点：05 时期 round1 曾因哈希缺 L1 走就近/跨题型回退（该断言钉住
// 不泄出题型）；12 号票矩阵收官后哈希 L1 存在，round1 直接命中。
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

// topic=sort + cap L5 + 均匀：40 次抽题全部命中排序题型；排序 L1-L5 齐备
// （12 号票矩阵收官，就近平移不再触发），难度应铺满多档（不钉死 5/5——
// 模板增补会改变随机流，4/5 足以证明均匀铺开且不越出题型）。
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
	if len(seen) < 4 {
		t.Fatalf("40 次均匀抽题应覆盖 ≥4/5 档（排序 L1-L5 齐备），实际 %v", seen)
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

// cap=L5 + 技能位：贴上限语义直取上限档——技能模板 L1-L5 已齐备（12 号票
// 批 1 补 L1/L2、批 6b/7 补 L4/L5），cap=L5 不再回退；回退机制本身由
// factory_test.go 的 TestSkillQuestionCapTierAndFallback（桩注册表留空档）覆盖。
func TestSkillQuestionExactCapTierAtL5(t *testing.T) {
	f := newTopicFactory(question.Config{MaxDifficulty: question.L5, Strategy: question.StrategyClimb, MatchSeed: "skill-cap5"})
	q := f.SkillQuestion(1, "p1")
	if strings.HasPrefix(templateID(q.ID), "error") {
		t.Fatalf("cap=L5 技能题应直取上限档，实际 %q", q.ID)
	}
	if !strings.Contains(templateID(q.ID), "-L5-") {
		t.Fatalf("技能模板 L5 已存在，cap=L5 应直取 L5：%q", q.ID)
	}
}

// 真空格保底：topic=recursion + cap L1——递归 L1 是设计 002 勘误认定的
// 永久语义空格（递归至少需要基例+递归步，装不进 L1"≤3 行/1 个语法点"），
// 整个 [L1,L1] 区间无递归攻击模板 → 退回跨题型（行动题同语义），必须给出
// 可用题而非错误占位。（05 时期用 hash×L1 触发，该格自 12 号票矩阵收官起已填充。）
func TestAttackTopicEmptyPoolFallsBackCrossTopic(t *testing.T) {
	f := newTopicFactory(question.Config{Topics: []question.Topic{question.TopicRecursion}, MaxDifficulty: question.L1, Strategy: question.StrategyClimb, MatchSeed: "topic-empty"})
	for round := 1; round <= 3; round++ {
		q := f.AttackQuestion(round)
		tpl := templateID(q.ID)
		if strings.HasPrefix(tpl, "error") {
			t.Fatalf("真空格应跨题型保底而非错误占位：%q", q.ID)
		}
		if idsOf(question.TopicRecursion, question.ActionAttack)[tpl] {
			t.Fatalf("cap=L1 不可能存在递归攻击模板（递归 L1 为永久语义空格）：%q", tpl)
		}
	}
}
