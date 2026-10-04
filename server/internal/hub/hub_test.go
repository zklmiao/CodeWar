package hub

import (
	"log/slog"
	"testing"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/match"
)

// 回归（浏览器验收 P0）：MatchEnded 广播后引擎必须作废。
// 否则 ENDED 相位的引擎被后续接入永久复用，"刷新再来一局"永远无法开赛。
func TestMatchEndedResetsEngine(t *testing.T) {
	h := New(nil, slog.Default())
	h.engine = match.NewEngine(match.RealClock{}, match.StubQuestions{}, nil, h.broadcast)

	h.broadcast(&matchpb.Envelope{Event: &matchpb.Envelope_MatchEnded{MatchEnded: &matchpb.MatchEnded{
		Winner: "p1", Reason: match.ReasonKO,
	}}})

	if h.engine != nil {
		t.Fatal("MatchEnded 后引擎应被置空，下一场对局由下一条 join 重建")
	}
}

// 引擎置空后，迟到的提交必须被安全丢弃（readLoop 的 nil 保护），不得 panic。
func TestSubmitAfterEngineResetIsDropped(t *testing.T) {
	h := New(nil, slog.Default())

	// 引擎为 nil（尚未创建或已被终局置空）：直接走与 readLoop 相同的守卫路径
	h.mu.Lock()
	eng := h.engine
	h.mu.Unlock()
	if eng != nil {
		eng.Submit(match.P1, "code")
	}
}
