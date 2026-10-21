package match

// 09 号票：AnswerKey 是服务端内部字段——Problem proto 没有它，任何下发路径
// （快照/事件流）都不得携带；引擎生命周期 Stop 用于连接断开作废（幂等且静默）。

import (
	"context"
	"strings"
	"testing"

	"codewar/server/gen/matchpb"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type fixedQuestions struct{ q Question }

func (f fixedQuestions) AttackQuestion(int) Question         { return f.q }
func (f fixedQuestions) MoveQuestion(int, Player) Question   { return f.q }
func (f fixedQuestions) DefendQuestion(int, Player) Question { return f.q }
func (f fixedQuestions) SkillQuestion(int, Player) Question  { return f.q }

type okJudge struct{}

func (okJudge) Judge(context.Context, string, []TestCase) JudgeResult {
	return JudgeResult{Passed: true, Category: CategoryOK}
}

func TestAnswerKeyNeverSerialized(t *testing.T) {
	const secret = "SECRET-ANSWER-KEY-42"
	q := Question{
		ID: "q1", Statement: "题面", Signature: "def f():", StarterCode: "starter",
		AnswerKey: secret,
		Tests:     []TestCase{{Name: "t", Call: "1", Expected: "1"}},
	}
	var events []*matchpb.Envelope
	eng := NewEngine(RealClock{}, fixedQuestions{q: q}, okJudge{}, func(env *matchpb.Envelope) {
		events = append(events, env)
	})
	eng.StartMatch()
	snap := eng.Snapshot(P1)
	eng.Stop()

	if snap.GetPanel() == nil || snap.GetPanel().GetAttack().GetStatement() != "题面" {
		t.Fatal("快照应携带题面（否则本测试的泄漏检查是空集）")
	}
	var panelReady bool
	for _, env := range events {
		if env.GetPanelReady() != nil {
			panelReady = true
		}
	}
	if !panelReady {
		t.Fatal("未捕获 PanelReady 事件")
	}

	msgs := make([]proto.Message, 0, len(events)+1)
	msgs = append(msgs, snap)
	for _, env := range events {
		msgs = append(msgs, env)
	}
	for _, m := range msgs {
		b, err := protojson.Marshal(m)
		if err != nil {
			t.Fatalf("序列化失败：%v", err)
		}
		if strings.Contains(string(b), secret) {
			t.Fatalf("参考解出现在下发消息中（客户端可见）：%s", b)
		}
	}
}

func TestStopIsIdempotentAndSilent(t *testing.T) {
	ended := 0
	eng := NewEngine(RealClock{}, StubQuestions{}, okJudge{}, func(env *matchpb.Envelope) {
		if env.GetMatchEnded() != nil {
			ended++
		}
	})
	eng.StartMatch()
	eng.Stop()
	eng.Stop() // 幂等：不得 panic（done 只关一次）
	if ended != 0 {
		t.Fatalf("作废对局不得广播 MatchEnded（实际 %d 条）", ended)
	}
	if eng.Phase() != matchpb.MatchPhase_MATCH_PHASE_ENDED {
		t.Fatalf("Stop 后应为 ENDED，实际 %v", eng.Phase())
	}
	// 已结束的引擎再次 Stop 不得 panic
	eng.Stop()
}
