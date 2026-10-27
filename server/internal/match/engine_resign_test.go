package match

// 31 号票：投降认输——引擎事件流 seam 断言（spec「测试决策」：只测外部行为，
// 不触碰引擎内部状态）。覆盖：终局事件流形状 / 重复投降幂等 / WAITING 与已终局
// 拒绝 / 教程局拒绝（拒绝一律零事件）。

import (
	"testing"

	"codewar/server/gen/matchpb"
)

// TestResignEndsMatchWithOpponentWin：活跃对局投降 → 对方胜、reason=resign、
// MatchEnded 恰好一次；投降是认输不是受伤——不得扣血（无 HpChanged）。
func TestResignEndsMatchWithOpponentWin(t *testing.T) {
	h := newHarness(t, passOK())
	if err := h.eng.Resign(P1); err != nil {
		t.Fatalf("活跃对局投降应被接受：%v", err)
	}
	ends := endings(h.events())
	if len(ends) != 1 {
		t.Fatalf("投降应产生恰好一次 MatchEnded，实际 %d 次", len(ends))
	}
	if ends[0].GetWinner() != P2 || ends[0].GetReason() != ReasonResign {
		t.Fatalf("终局应为对方胜（reason=resign）：%v", ends[0])
	}
	if n := len(hpChanges(h.events())); n != 0 {
		t.Fatalf("投降不得扣血（无 HpChanged），实际 %d 条", n)
	}
	if phase := h.eng.Snapshot(P1).GetPhase(); phase != matchpb.MatchPhase_MATCH_PHASE_ENDED {
		t.Fatalf("投降后对局应已终局，实际相位 %v", phase)
	}
}

// TestResignTwiceIsIdempotent：重复投降幂等——第一次终局后，任何一方的第二次
// 投降都落入"已终局"拒绝，事件流不得出现第二个 MatchEnded。
// （已终局拒绝与引擎以何原因终局无关：投降/ko/rounds-cap 后 phase 同为 ENDED，
// 是 Resign 的同一分支，不另设"常规终局后拒绝"的重复样本。）
func TestResignTwiceIsIdempotent(t *testing.T) {
	h := newHarness(t, passOK())
	if err := h.eng.Resign(P1); err != nil {
		t.Fatalf("首次投降应被接受：%v", err)
	}
	if err := h.eng.Resign(P2); err == nil {
		t.Fatal("对方在已终局对局上的投降应被拒绝")
	}
	if err := h.eng.Resign(P1); err == nil {
		t.Fatal("重复投降应被拒绝")
	}
	if len(endings(h.events())) != 1 {
		t.Fatalf("重复投降不得产生第二个 MatchEnded，实际 %d 次", len(endings(h.events())))
	}
}

// TestResignRejectedBeforeStart：未开赛（WAITING）不存在的对局无从认输，拒绝。
func TestResignRejectedBeforeStart(t *testing.T) {
	clk := &fakeClock{now: baseTime}
	var envs []*matchpb.Envelope
	eng := NewEngine(clk, StubQuestions{}, &scriptJudge{results: []JudgeResult{passOK()}},
		func(e *matchpb.Envelope) { envs = append(envs, e) })
	if err := eng.Resign(P1); err == nil {
		t.Fatal("未开赛（WAITING）不得投降")
	}
	if len(envs) != 0 {
		t.Fatalf("拒绝不得产生任何事件，实际 %d 条", len(envs))
	}
}

// TestResignRejectedWhenLocked：禁投对局（接入层对教学引导局等设置的
// ResignLocked）投降一律拒绝，拒绝零事件，且对局照常活跃——拒绝不是作废，
// 后续提交仍被正常受理。（引擎不知道"教程"概念——10 号票零侵入边界；
// hub 在教学局设置此标记的端到端断言在 hub_resign_test.go。）
func TestResignRejectedWhenLocked(t *testing.T) {
	clk := &fakeClock{now: baseTime}
	var envs []*matchpb.Envelope
	eng := NewEngineWithSetup(clk, StubQuestions{}, &scriptJudge{results: []JudgeResult{passOK()}},
		func(e *matchpb.Envelope) { envs = append(envs, e) }, Setup{ResignLocked: true})
	eng.StartMatch()
	baseline := len(envs)
	if err := eng.Resign(P1); err == nil {
		t.Fatal("禁投对局不得投降")
	}
	if len(envs) != baseline {
		t.Fatalf("禁投对局的拒绝不得产生任何事件，实际新增 %d 条", len(envs)-baseline)
	}
	eng.Submit(P1, ActionDefend, "code", Pos{})
	if len(judged(envs)) == 0 {
		t.Fatal("拒绝投降后对局应照常进行（提交仍被受理）")
	}
	if len(endings(envs)) != 0 {
		t.Fatal("拒绝投降不得触发终局")
	}
}
