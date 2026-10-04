package hub

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/match"
	"codewar/server/internal/store"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
)

// stubStore：accountStore 桩（记录调用，预设结算结果/错误）。
type stubStore struct {
	userByToken map[string]store.User
	authErr     error
	recordErr   error
	lastRes     *store.MatchResult
	gotDeadline bool
	d1, d2      store.EloDelta
}

func (s *stubStore) Authenticate(_ context.Context, token string) (store.User, error) {
	if s.authErr != nil {
		return store.User{}, s.authErr
	}
	if u, ok := s.userByToken[token]; ok {
		return u, nil
	}
	return store.User{}, store.ErrUnauthorized
}

func (s *stubStore) RecordMatch(ctx context.Context, res store.MatchResult) (store.EloDelta, store.EloDelta, error) {
	if s.recordErr != nil {
		return store.EloDelta{}, store.EloDelta{}, s.recordErr
	}
	if _, ok := ctx.Deadline(); ok {
		s.gotDeadline = true
	}
	cp := res
	s.lastRes = &cp
	return s.d1, s.d2, nil
}

func testUsers() (a, b store.User) {
	a = store.User{ID: uuid.New(), Username: "alice", Elo: 1000}
	b = store.User{ID: uuid.New(), Username: "bob", Elo: 1000}
	return a, b
}

// 双认证终局：MatchEnded 附上双方 ELO（同分 p1 胜 → 1016/984），引擎照常作废。
func TestBroadcastAttachesEloWhenBothAuthenticated(t *testing.T) {
	a, b := testUsers()
	st := &stubStore{
		d1: store.EloDelta{Player: "p1", UserID: a.ID, Before: 1000, After: 1016},
		d2: store.EloDelta{Player: "p2", UserID: b.ID, Before: 1000, After: 984},
	}
	h := New(nil, slog.Default(), nil)
	h.st = st
	h.seats[match.P1] = a
	h.seats[match.P2] = b
	h.engine = match.NewEngine(match.RealClock{}, match.StubQuestions{}, nil, h.broadcast)

	env := &matchpb.Envelope{Event: &matchpb.Envelope_MatchEnded{MatchEnded: &matchpb.MatchEnded{
		Winner: "p1", Reason: match.ReasonKO,
	}}}
	h.broadcast(env)

	me := env.GetMatchEnded()
	if len(me.GetElo()) != 2 {
		t.Fatalf("双认证终局应附 2 条 ELO，实际 %d", len(me.GetElo()))
	}
	if me.GetElo()[0].GetEloBefore() != 1000 || me.GetElo()[0].GetEloAfter() != 1016 {
		t.Fatalf("p1 ELO 不符：%v", me.GetElo()[0])
	}
	if me.GetElo()[1].GetUserId() != b.ID.String() || me.GetElo()[1].GetEloAfter() != 984 {
		t.Fatalf("p2 ELO 不符：%v", me.GetElo()[1])
	}
	if st.lastRes == nil || st.lastRes.Winner != "p1" || st.lastRes.P1UserID != a.ID {
		t.Fatalf("落库参数不符：%+v", st.lastRes)
	}
	// H2：测试桩未走开赛记账（matchStart 零值），时长也应 sane，不能是回绕垃圾
	if d := st.lastRes.DurationMs; d < 0 || d > 5000 {
		t.Fatalf("对局时长应为合理毫秒数，实际 %d", d)
	}
	if h.engine != nil {
		t.Fatal("MatchEnded 后引擎仍应被置空")
	}
}

// 结算落库在带超时的 context 下执行（H3：DB 挂起不得永久卡住持锁的广播路径）。
func TestSettleUsesBoundedContext(t *testing.T) {
	a, b := testUsers()
	st := &stubStore{
		d1: store.EloDelta{Player: "p1", UserID: a.ID, Before: 1000, After: 1016},
		d2: store.EloDelta{Player: "p2", UserID: b.ID, Before: 1000, After: 984},
	}
	h := New(nil, slog.Default(), nil)
	h.st = st
	h.seats[match.P1] = a
	h.seats[match.P2] = b

	env := &matchpb.Envelope{Event: &matchpb.Envelope_MatchEnded{MatchEnded: &matchpb.MatchEnded{
		Winner: "p1", Reason: match.ReasonKO,
	}}}
	h.broadcast(env)

	if !st.gotDeadline {
		t.Fatal("结算落库应在带超时的 context 下执行")
	}
	if len(env.GetMatchEnded().GetElo()) != 2 {
		t.Fatal("带超时结算成功时仍应附 ELO")
	}
}
func TestBroadcastNoEloWhenAnonymous(t *testing.T) {
	a, _ := testUsers()
	st := &stubStore{}
	h := New(nil, slog.Default(), nil)
	h.st = st
	h.seats[match.P1] = a // 仅一方认证

	env := &matchpb.Envelope{Event: &matchpb.Envelope_MatchEnded{MatchEnded: &matchpb.MatchEnded{
		Winner: "p1", Reason: match.ReasonKO,
	}}}
	h.broadcast(env)

	if len(env.GetMatchEnded().GetElo()) != 0 {
		t.Fatalf("匿名对局不得带 elo：%v", env.GetMatchEnded().GetElo())
	}
	if st.lastRes != nil {
		t.Fatal("匿名对局不得落库")
	}
}

// 无库模式（st=nil）：纯匿名热座，终局无 elo 且不断连。
func TestBroadcastNoEloWithoutStore(t *testing.T) {
	h := New(nil, slog.Default(), nil)
	env := &matchpb.Envelope{Event: &matchpb.Envelope_MatchEnded{MatchEnded: &matchpb.MatchEnded{
		Reason: match.ReasonRoundsCap, // 平局
	}}}
	h.broadcast(env) // 不得 panic
	if len(env.GetMatchEnded().GetElo()) != 0 {
		t.Fatal("无库模式不得带 elo")
	}
}

// 落库失败：降级为无 ELO 广播，对局照常结束（不断连、不重发）。
func TestBroadcastDegradesOnRecordError(t *testing.T) {
	a, b := testUsers()
	st := &stubStore{recordErr: context.DeadlineExceeded}
	h := New(nil, slog.Default(), nil)
	h.st = st
	h.seats[match.P1] = a
	h.seats[match.P2] = b
	h.engine = match.NewEngine(match.RealClock{}, match.StubQuestions{}, nil, h.broadcast)

	env := &matchpb.Envelope{Event: &matchpb.Envelope_MatchEnded{MatchEnded: &matchpb.MatchEnded{
		Winner: "p2", Reason: match.ReasonKO,
	}}}
	h.broadcast(env)

	if len(env.GetMatchEnded().GetElo()) != 0 {
		t.Fatal("落库失败时不得带 elo")
	}
	if h.engine != nil {
		t.Fatal("落库失败引擎仍应作废（对局照常结束）")
	}
}

// 同用户分饰双方：跳过结算（防自刷分），无落库无 elo。
func TestBroadcastSkipsSelfPlay(t *testing.T) {
	a, _ := testUsers()
	st := &stubStore{}
	h := New(nil, slog.Default(), nil)
	h.st = st
	h.seats[match.P1] = a
	h.seats[match.P2] = a

	env := &matchpb.Envelope{Event: &matchpb.Envelope_MatchEnded{MatchEnded: &matchpb.MatchEnded{
		Winner: "p1", Reason: match.ReasonKO,
	}}}
	h.broadcast(env)

	if st.lastRes != nil || len(env.GetMatchEnded().GetElo()) != 0 {
		t.Fatal("自对弈不得结算落库")
	}
}

// 有效 token 建连绑定席位；断开后连接与席位一并清除（H1：
// 否则同席位换身份重连会沿用旧身份结算）。
func TestJoinBindsUserAndCleansUpOnDisconnect(t *testing.T) {
	a, _ := testUsers()
	st := &stubStore{userByToken: map[string]store.User{"tok-a": a}}
	h := New(nil, slog.Default(), nil)
	h.st = st
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatalf("拨号失败：%v", err)
	}
	join, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Join{
		Join: &matchpb.JoinRequest{Player: "p1", Token: "tok-a"},
	}})
	if err := ws.WriteMessage(websocket.TextMessage, join); err != nil {
		t.Fatalf("发送 join 失败：%v", err)
	}
	// 建连被接受的证据：收到快照（被拒则直接关连接，读失败）
	_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, data, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("有效 token 应建连成功并下发快照：%v", err)
	}
	var msg matchpb.ServerMessage
	if err := (protojson.UnmarshalOptions{}).Unmarshal(data, &msg); err != nil || msg.GetSnapshot() == nil {
		t.Fatalf("首条下行应为快照：err=%v msg=%v", err, &msg)
	}

	h.mu.Lock()
	_, bound := h.seats[match.P1]
	nconns := len(h.conns)
	h.mu.Unlock()
	if !bound || nconns != 1 {
		t.Fatalf("席位应已绑定：bound=%v conns=%d", bound, nconns)
	}

	ws.Close()
	// readLoop 的 defer 异步清理，轮询等它（2s 上限防挂死）
	deadline := time.Now().Add(2 * time.Second)
	for {
		h.mu.Lock()
		empty := len(h.conns) == 0 && len(h.seats) == 0
		h.mu.Unlock()
		if empty {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("断开后连接与席位应在 2s 内清空")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// 有 token 但服务端无库：拒连（会话无法验证，不静默降级为匿名）。
func TestJoinWithTokenButNoStoreIsRejected(t *testing.T) {
	h := New(nil, slog.Default(), nil)
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatalf("拨号失败：%v", err)
	}
	defer ws.Close()
	join, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Join{
		Join: &matchpb.JoinRequest{Player: "p1", Token: "tok"},
	}})
	if err := ws.WriteMessage(websocket.TextMessage, join); err != nil {
		t.Fatalf("发送 join 失败：%v", err)
	}
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("无库模式带 token 应被拒连")
	}
	if h.engine != nil {
		t.Fatal("被拒的连接不得创建引擎")
	}
}
func TestJoinRejectsInvalidToken(t *testing.T) {
	st := &stubStore{authErr: store.ErrUnauthorized}
	h := New(nil, slog.Default(), nil)
	h.st = st
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatalf("拨号失败：%v", err)
	}
	defer ws.Close()
	join, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Join{
		Join: &matchpb.JoinRequest{Player: "p1", Token: "bogus"},
	}})
	if err := ws.WriteMessage(websocket.TextMessage, join); err != nil {
		t.Fatalf("发送 join 失败：%v", err)
	}
	// 服务端拒连即关连接：读应失败（CloseError 或 EOF）。
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("无效 token 应被拒连（读应失败）")
	}
	if h.engine != nil {
		t.Fatal("被拒的连接不得创建引擎")
	}
}
