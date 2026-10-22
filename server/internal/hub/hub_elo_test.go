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
	// 10 号票：教程进度（零值 = 新账号：level 0、未通关）。
	tutorial map[uuid.UUID]store.TutorialProgress
	tutErr   error
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

func (s *stubStore) GetTutorialProgress(_ context.Context, userID uuid.UUID) (store.TutorialProgress, error) {
	if s.tutErr != nil {
		return store.TutorialProgress{}, s.tutErr
	}
	return s.tutorial[userID], nil
}

func (s *stubStore) AdvanceTutorial(_ context.Context, userID uuid.UUID, level, total int32) (store.TutorialProgress, error) {
	if s.tutErr != nil {
		return store.TutorialProgress{}, s.tutErr
	}
	p := s.tutorial[userID]
	if level > p.Level {
		p.Level = level
	}
	if level >= total && !p.Completed {
		now := time.Now()
		p.CompletedAt = &now
		p.Completed = true
	}
	if s.tutorial == nil {
		s.tutorial = map[uuid.UUID]store.TutorialProgress{}
	}
	s.tutorial[userID] = p
	return p, nil
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

// passJudge：全过桩。双方秒解 → 每回合互扣 22 → 5 回合双双归零平局，数秒打满。
type passJudge struct{}

func (passJudge) Judge(_ context.Context, _ string, _ []match.TestCase) match.JudgeResult {
	return match.JudgeResult{Passed: true, Category: match.CategoryOK}
}

// 完整回路：双认证接入 → 打满 → 终局附 ELO（07 号票主干，CI 可跑，无需真 DB/judge）。
func TestAuthenticatedMatchSettlesEloEndToEnd(t *testing.T) {
	a, b := testUsers()
	now := time.Now()
	st := &stubStore{
		userByToken: map[string]store.User{"tok-a": a, "tok-b": b},
		d1:          store.EloDelta{Player: "p1", UserID: a.ID, Before: 1000, After: 1000},
		d2:          store.EloDelta{Player: "p2", UserID: b.ID, Before: 1000, After: 1000},
		// 10 号票起 PvP 入口要求教程已完成（本测试只关心结算链路）。
		tutorial: map[uuid.UUID]store.TutorialProgress{
			a.ID: {Level: 4, Completed: true, CompletedAt: &now},
			b.ID: {Level: 4, Completed: true, CompletedAt: &now},
		},
	}
	h := New(passJudge{}, slog.Default(), nil)
	h.st = st
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ends := make(chan *matchpb.MatchEnded, 2)
	for _, tc := range []struct{ player, token string }{{"p1", "tok-a"}, {"p2", "tok-b"}} {
		ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
		if err != nil {
			t.Fatalf("拨号失败：%v", err)
		}
		defer ws.Close()
		join, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Join{
			Join: &matchpb.JoinRequest{Player: tc.player, Token: tc.token},
		}})
		if err := ws.WriteMessage(websocket.TextMessage, join); err != nil {
			t.Fatalf("发送 join 失败：%v", err)
		}
		go playLoop(t, ws, tc.player, ends)
	}

	collected := make([]*matchpb.MatchEnded, 0, 2)
	timeout := time.After(30 * time.Second)
	for len(collected) < 2 {
		select {
		case me := <-ends:
			if me == nil {
				t.Fatal("对局中连接断裂（playLoop 已报明细）")
			}
			collected = append(collected, me)
		case <-timeout:
			t.Fatal("30s 内对局未终局")
		}
	}
	for _, me := range collected {
		if me.GetWinner() != "" {
			t.Fatalf("对称秒解应判平局，实际 winner=%q reason=%q", me.GetWinner(), me.GetReason())
		}
		if len(me.GetElo()) != 2 || me.GetElo()[0].GetPlayer() != "p1" || me.GetElo()[1].GetPlayer() != "p2" {
			t.Fatalf("终局应附双方 ELO：%v", me.GetElo())
		}
		if me.GetElo()[0].GetUserId() != a.ID.String() || me.GetElo()[1].GetUserId() != b.ID.String() {
			t.Fatalf("ELO 应绑定双方用户：%v", me.GetElo())
		}
	}
	if st.lastRes == nil || st.lastRes.Winner != "" || st.lastRes.P1UserID != a.ID || st.lastRes.P2UserID != b.ID {
		t.Fatalf("落库参数不符：%+v", st.lastRes)
	}
	if d := st.lastRes.DurationMs; d < 0 || d > 30000 {
		t.Fatalf("真实对局时长应合理，实际 %d", d)
	}
}

// playLoop：测试客户端——见面板即秒交，全程只为把对局推向终局。
// 出错只 E（不断言），调用方以 ends==nil 或超时判 fail，保证快速失败。
func playLoop(t *testing.T, ws *websocket.Conn, player string, ends chan<- *matchpb.MatchEnded) {
	t.Helper()
	_ = ws.SetReadDeadline(time.Now().Add(400 * time.Second)) // 跨多回合的集成对局需要更长读窗
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			t.Errorf("[%s] 读事件失败: %v", player, err)
			ends <- nil
			return
		}
		var msg matchpb.ServerMessage
		if err := (protojson.UnmarshalOptions{}).Unmarshal(data, &msg); err != nil {
			t.Errorf("[%s] 解析失败: %v", player, err)
			ends <- nil
			return
		}
		env := msg.GetEvent()
		if env == nil {
			continue // 快照
		}
		if env.GetPanelReady() != nil {
			sub, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Submit{
				Submit: &matchpb.SubmitRequest{Code: "code", Action: "attack"},
			}})
			if err := ws.WriteMessage(websocket.TextMessage, sub); err != nil {
				t.Errorf("[%s] 提交失败: %v", player, err)
				ends <- nil
				return
			}
		}
		if me := env.GetMatchEnded(); me != nil {
			ends <- me
			return
		}
	}
}

// 有效 token 建连绑定席位；断开后连接与席位一并清除（H1：
// 否则同席位换身份重连会沿用旧身份结算）。
func TestJoinBindsUserAndCleansUpOnDisconnect(t *testing.T) {
	a, _ := testUsers()
	now := time.Now()
	st := &stubStore{
		userByToken: map[string]store.User{"tok-a": a},
		tutorial:    map[uuid.UUID]store.TutorialProgress{a.ID: {Level: 4, Completed: true, CompletedAt: &now}},
	}
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

// 首条消息非法 → 关连接，不建引擎（readJoin 错误分支）。
func TestJoinRejectsMalformedMessage(t *testing.T) {
	h := New(nil, slog.Default(), nil)
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatalf("拨号失败：%v", err)
	}
	defer ws.Close()
	if err := ws.WriteMessage(websocket.TextMessage, []byte("not-json{{{")); err != nil {
		t.Fatalf("发送失败：%v", err)
	}
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("非法首条消息应被拒连")
	}
	if h.engine != nil {
		t.Fatal("被拒的连接不得创建引擎")
	}
}

// 同席位重复接入 → 后者被拒，前者不受影响（taken 分支）。
func TestJoinRejectsDuplicateSeat(t *testing.T) {
	h := New(nil, slog.Default(), nil)
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()
	dial := func() *websocket.Conn {
		t.Helper()
		ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
		if err != nil {
			t.Fatalf("拨号失败：%v", err)
		}
		return ws
	}
	joinOf := func(player string) []byte {
		b, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Join{
			Join: &matchpb.JoinRequest{Player: player},
		}})
		return b
	}

	first := dial()
	defer first.Close()
	if err := first.WriteMessage(websocket.TextMessage, joinOf("p1")); err != nil {
		t.Fatal(err)
	}
	_ = first.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, _, err := first.ReadMessage(); err != nil {
		t.Fatalf("首个 p1 应建连成功：%v", err)
	}

	second := dial()
	defer second.Close()
	if err := second.WriteMessage(websocket.TextMessage, joinOf("p1")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := second.ReadMessage(); err == nil {
		t.Fatal("重复席位应被拒连")
	}
	h.mu.Lock()
	n := len(h.conns)
	h.mu.Unlock()
	if n != 1 {
		t.Fatalf("前者连接应不受影响，conns=%d", n)
	}
}

// 回合外非法消息不断连（readLoop 继续服务，而非退出）。
func TestLoopSurvivesGarbageMessage(t *testing.T) {
	h := New(nil, slog.Default(), nil)
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatalf("拨号失败：%v", err)
	}
	defer ws.Close()
	join, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Join{
		Join: &matchpb.JoinRequest{Player: "p1"},
	}})
	if err := ws.WriteMessage(websocket.TextMessage, join); err != nil {
		t.Fatal(err)
	}
	_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, _, err := ws.ReadMessage(); err != nil {
		t.Fatalf("建连应成功：%v", err)
	}
	if err := ws.WriteMessage(websocket.TextMessage, []byte("garbage{{{")); err != nil {
		t.Fatal(err)
	}
	// 给服务端一个处理时间片；若 readLoop 退出，conns 会被清空
	time.Sleep(200 * time.Millisecond)
	h.mu.Lock()
	n := len(h.conns)
	h.mu.Unlock()
	if n != 1 {
		t.Fatal("非法后续消息不应断开连接")
	}
}

// 无效 token 建连被拒（客户端应重登后重试，不静默降级为匿名）。
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
