package bot

// 19 号票回归：PvE Bot 超射程自我放逐（随机走位后永久 miss）。
//
// 场景按架构师裁决 4 构造：fake clock + 脚本化对手，事件流断言外部行为。
// 修复前 Bot 超距后移动不偏好接近（随机方向）、攻击被自动瞄准的距离钳制
// 永久落空（连续 dodged=true）；修复后超距移动确定性接近、恢复命中。
// 引擎零改动：闪避/移动规则由 03 号票覆盖，这里只断言 Bot 决策层的行为。

import (
	"testing"
	"time"

	"codewar/server/internal/classes"
	"codewar/server/internal/match"
)

// testManhattan：棋盘距离（与引擎 coverage/移动规则同量纲）。
// 测试本地实现，不依赖被测修复的任何新符号。
func testManhattan(a, b match.Pos) int {
	dr, dc := a.Row-b.Row, a.Col-b.Col
	if dr < 0 {
		dr = -dr
	}
	if dc < 0 {
		dc = -dc
	}
	return dr + dc
}

// fleeOrElseDefend：脚本化人类——每回合移到"与 Bot 距离最远"的合法格，
// 无能拉开距离的落点（被逼进死角）时转防御站桩。确定性：同距取 LegalMoves
// 扫描序首个。人类先提交 → solveSeq 靠前 → 同回合先于 Bot 结算（走位闪避
// 是引擎语义，命中断言只在人类站桩回合成立，与真实博弈一致）。
func fleeOrElseDefend(h *matchHarness, round int) {
	snap := h.eng.Snapshot(match.P1)
	me, ok1 := posOf(snap, match.P1)
	opp, ok2 := posOf(snap, match.P2)
	if !ok1 || !ok2 {
		return
	}
	bestDist := testManhattan(me, opp)
	var to match.Pos
	fled := false
	for _, m := range LegalMoves(me, opp, 3) {
		if d := testManhattan(m, opp); d > bestDist {
			bestDist, to, fled = d, m, true
		}
	}
	if fled {
		h.count.Submit(match.P1, match.ActionMove, h.factory.MoveQuestion(round, match.P1).AnswerKey, to)
		return
	}
	h.count.Submit(match.P1, match.ActionDefend, h.factory.DefendQuestion(round, match.P1).AnswerKey, match.Pos{})
}

// playFleeScenario：驱动整局。每回合人类先提交、再推进时钟触发 Bot 的 0ms
// 提交并结算；同时按回合记录结算前的双方站位（= Bot 本回合 act 时看到的
// 快照——Bot 上一回合结算后的 Advance 批次里就已提交，位置未被改动）。
type fleeRun struct {
	positions map[int][2]match.Pos // round → {bot, human}（Bot act 时的快照）
	botMoves  []botMoveEvent       // Bot 的移动结算事件（按发生序）
}

type botMoveEvent struct {
	round int
	to    match.Pos
}

func playFleeScenario(h *matchHarness, rounds int) fleeRun {
	run := fleeRun{positions: map[int][2]match.Pos{}}
	record := func(round int) {
		snap := h.eng.Snapshot(match.P1)
		bot, _ := posOf(snap, match.P2)
		human, _ := posOf(snap, match.P1)
		run.positions[round] = [2]match.Pos{bot, human}
	}
	mark := len(h.events)
	h.eng.StartMatch()
	for round := 1; round <= rounds && h.ended == nil; round++ {
		if !h.active() || h.round() != round {
			break
		}
		record(round)
		fleeOrElseDefend(h, round)
		h.clk.Advance(time.Millisecond)
		for _, env := range h.events[mark:] {
			if ar := env.GetActionResolved(); ar != nil &&
				ar.GetAttacker() == string(match.P2) && ar.GetAction() == match.ActionMove {
				run.botMoves = append(run.botMoves, botMoveEvent{round: round, to: match.Pos{
					Row: int(ar.GetMoveTo().GetRow()), Col: int(ar.GetMoveTo().GetCol()),
				}})
			}
		}
		mark = len(h.events)
	}
	h.ctl.Stop()
	h.eng.Stop()
	return run
}

// 场景配置：双方游侠（Bot 装风行回复——治疗技能不放大攻击射程，reach=3），
// 行动权重覆盖移动/攻击两相，0ms 解题让整局在毫秒级跑完。

// 结构断言：Bot 在超距回合的移动必须严格接近对手（裁决 1：接近目标使距离
// 落入射程内；一格进不了射程时沿主轴取最大进度）。删除接近逻辑后，随机
// 移动迟早出现"远离"样本，本测试转红。
func TestBotOutOfRangeMoveClosesDistance(t *testing.T) {
	h := newConfiguredHarness(t, "pve-out-of-range", "ranger", "", "ranger", "windheal",
		Params{MinSolveMs: 0, MaxSolveMs: 0, Accuracy: 1, Weights: Weights{Move: 100}})
	run := playFleeScenario(h, 10)

	// 游侠普攻射程 3（设计 003 §2 字面值；风行回复为自身治疗，不放大射程）。
	const rangerReach = 3
	checks := 0
	for _, mv := range run.botMoves {
		p := run.positions[mv.round]
		before := testManhattan(p[0], p[1])
		if before <= rangerReach {
			continue // 射程内移动维持现行随机决策（裁决 1），不在本断言范围
		}
		after := testManhattan(mv.to, p[1])
		if after >= before {
			t.Fatalf("超距回合 %d 的移动未接近对手：bot=%v opp=%v moveTo=%v 距离 %d→%d",
				mv.round, p[0], p[1], mv.to, before, after)
		}
		checks++
	}
	if checks == 0 {
		t.Fatal("场景无效：整局未出现超距移动回合，无法约束接近行为")
	}
	if h.rejected != 0 {
		t.Fatalf("Bot 出现 %d 次参数拒绝（接近落点必须经引擎合法性规则）", h.rejected)
	}
}

// 端到端断言（裁决 4 + 票据 L22）：修复后 Bot 先接近再命中——事件流呈现
// "dodged 由 true 转为命中"。修复前（种子 exile-08）该局面下 Bot 的 8 次攻击
// 全部 dodged=true、伤害 0 次（超距放逐到回合上限），本测试红——即票据实录
// 的缺陷现场。三重守卫防空转：必须先出现闪避阶段、后出现命中、且命中在
// 事件序上位于闪避之后（否则种子局面退化为开局即射程内，测试虚假通过）。
func TestBotApproachesThenLandsHit(t *testing.T) {
	h := newConfiguredHarness(t, "exile-08", "ranger", "", "ranger", "windheal",
		Params{MinSolveMs: 0, MaxSolveMs: 0, Accuracy: 1, Weights: Weights{Attack: 60, Move: 40}})
	playFleeScenario(h, 10)

	attacks, dodged, hit := 0, 0, 0
	firstDodge, firstHit := -1, -1
	for i, env := range h.events {
		ar := env.GetActionResolved()
		if ar == nil || ar.GetAttacker() != string(match.P2) || ar.GetAction() != match.ActionAttack {
			continue
		}
		attacks++
		switch {
		case ar.GetDamage() > 0:
			hit++
			if firstHit < 0 {
				firstHit = i
			}
		case ar.GetDodged():
			dodged++
			if firstDodge < 0 {
				firstDodge = i
			}
		}
	}
	if attacks == 0 {
		t.Fatal("场景无效：Bot 未发起任何攻击")
	}
	if dodged == 0 {
		t.Fatal("场景无效：无超距闪避阶段（开局即命中），未复现放逐→恢复的转移")
	}
	if hit == 0 {
		t.Fatalf("Bot 从未命中（超距放逐）：P2 攻击 dodged=%d 次、伤害 0 次", dodged)
	}
	if firstHit < firstDodge {
		t.Fatalf("事件序异常：首次命中(事件#%d)早于首次闪避(事件#%d)，未呈现 dodged→命中转移",
			firstHit, firstDodge)
	}
	if h.rejected != 0 {
		t.Fatalf("Bot 出现 %d 次参数拒绝", h.rejected)
	}
}

// ===== 纯函数契约：射程口径与接近落点选择 =====

// classWithSkill：取职业与其技能（测试辅助）。
func classWithSkill(t *testing.T, classID, skillID string) (classes.Class, classes.Skill, bool) {
	t.Helper()
	c, ok := classes.ByID(classID)
	if !ok {
		t.Fatalf("未知职业 %q", classID)
		return c, classes.Skill{}, false
	}
	sk, has, err := classes.SkillByID(classID, skillID)
	if err != nil {
		t.Fatalf("职业 %q 查技能 %q 失败：%v", classID, skillID, err)
	}
	return c, sk, has
}

// 射程口径（裁决 1）：普攻按职业——剑士 1、法师 2、游侠 3；技能就绪取 max，
// 穿云箭（4）放大游侠射程，治疗/邻域技能不放大；冷却中不放大。
func TestEffectiveReachByClass(t *testing.T) {
	cases := []struct {
		classID, skillID string
		skillReady       bool
		want             int
	}{
		{"warrior", "", false, 1},
		{"warrior", "cleave", true, 1},  // 巨力斩邻域无距离语义
		{"warrior", "survive", true, 1}, // 治疗不放大
		{"mage", "", false, 2},
		{"mage", "firestorm", true, 2}, // 横带纵深 2
		{"mage", "drain", true, 2},     // 秘法汲取射程 2
		{"ranger", "", false, 3},
		{"ranger", "windheal", true, 3}, // 治疗不放大
		{"ranger", "pierce", false, 3},  // 冷却中不放大
		{"ranger", "pierce", true, 4},   // 穿云箭就绪 → 4
	}
	for _, tc := range cases {
		c, sk, has := classWithSkill(t, tc.classID, tc.skillID)
		if got := EffectiveReach(c, sk, has, tc.skillReady); got != tc.want {
			t.Fatalf("%s+%s(ready=%v) 射程 = %d，want %d", tc.classID, tc.skillID, tc.skillReady, got, tc.want)
		}
	}
}

// 接近落点选择（裁决 1/2）：
//   - 角落对角（裁决 4 的 (0,0)/(4,4) 游侠局面）：一格进不了射程 → 沿主轴
//     最大进度，跨回合收敛；
//   - 一格能进射程 → 停在射程边缘（rim），不贴脸；
//   - 轴向被封死 → false（调用方退化随机移动，裁决 1 的负例分支）。
func TestApproachMove(t *testing.T) {
	cases := []struct {
		name           string
		from, opp      match.Pos
		moveMax, reach int
		want           match.Pos
		ok             bool
	}{
		{"角落对角取最大进度（裁决4局面）", match.Pos{Row: 0, Col: 0}, match.Pos{Row: 4, Col: 4}, 3, 3, match.Pos{Row: 3, Col: 0}, true},
		{"对角反向同构", match.Pos{Row: 4, Col: 4}, match.Pos{Row: 0, Col: 0}, 3, 3, match.Pos{Row: 1, Col: 4}, true},
		{"一格进射程停在边缘", match.Pos{Row: 0, Col: 0}, match.Pos{Row: 4, Col: 0}, 3, 3, match.Pos{Row: 1, Col: 0}, true},
		{"法师两格进射程", match.Pos{Row: 2, Col: 2}, match.Pos{Row: 0, Col: 4}, 2, 2, match.Pos{Row: 0, Col: 2}, true},
		{"剑士斜向相邻接近", match.Pos{Row: 0, Col: 0}, match.Pos{Row: 1, Col: 1}, 2, 1, match.Pos{Row: 1, Col: 0}, true},
		{"边缘优先于更近格", match.Pos{Row: 0, Col: 0}, match.Pos{Row: 1, Col: 4}, 3, 3, match.Pos{Row: 0, Col: 2}, true},
		{"轴向封死退化", match.Pos{Row: 0, Col: 0}, match.Pos{Row: 0, Col: 1}, 2, 1, match.Pos{}, false},
	}
	for _, tc := range cases {
		got, ok := ApproachMove(tc.from, tc.opp, tc.moveMax, tc.reach)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("%s：got %v,%v want %v,%v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

// 射程内随机决策保持不变（裁决 1"射程内时维持现行权重决策"+ 裁决 3 随机流
// 不变）：ChooseMove 在射程内与 PickMove 消费同一随机序列、产生同一落点；
// 接近落点为确定性计算，不消耗随机流。
func TestChooseMoveRandomStreamUntouched(t *testing.T) {
	from, opp := match.Pos{Row: 2, Col: 2}, match.Pos{Row: 2, Col: 3} // 距离 1 ≤ 射程
	r1 := &seqRNG{ints: []int{7}}
	r2 := &seqRNG{ints: []int{7}}
	a, okA := ChooseMove(r1, from, opp, 3, 3)
	b, okB := PickMove(r2, from, opp, 3)
	if !okA || !okB || a != b {
		t.Fatalf("射程内应与现行随机移动一致：%v,%v vs %v,%v", a, okA, b, okB)
	}
	if r1.ii != r2.ii || r1.fi != r2.fi {
		t.Fatalf("射程内随机流消费应一致：%+v vs %+v", r1, r2)
	}

	// 超距：确定性接近，不消耗任何随机流。
	r3 := &seqRNG{ints: []int{7}}
	to, ok := ChooseMove(r3, match.Pos{Row: 0, Col: 0}, match.Pos{Row: 4, Col: 4}, 3, 3)
	if !ok || to != (match.Pos{Row: 3, Col: 0}) {
		t.Fatalf("超距应确定性接近：got %v,%v", to, ok)
	}
	if r3.ii != 0 || r3.fi != 0 {
		t.Fatalf("接近落点不应消耗随机流：%+v", r3)
	}
}
