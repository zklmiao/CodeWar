package hub

// 31 号票：投降认输——hub 接线端到端（ws → 引擎 Resign → 终局广播 → settle）。
// PvP：走既有 settle 全链（stub 门控 + 真 DB 门控两档断言）；PvE：bot 胜且零落库
//（广播路径的单人模式守卫兜住）；教程局：拒绝且无事件、对局照常活跃。

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/match"
	"codewar/server/internal/store"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// resignMsg：客户端投降消息（请求体无字段——身份由连接席位权威决定）。
func resignMsg() []byte {
	b, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Resign{
		Resign: &matchpb.ResignRequest{},
	}})
	return b
}

// waitEnded：从 ws 读到 MatchEnded（跳过快照与其他事件），超时 fail。
func waitEnded(t *testing.T, ws *websocket.Conn, timeout time.Duration) *matchpb.MatchEnded {
	t.Helper()
	_ = ws.SetReadDeadline(time.Now().Add(timeout))
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("等终局事件失败：%v", err)
		}
		var msg matchpb.ServerMessage
		if err := (protojson.UnmarshalOptions{}).Unmarshal(data, &msg); err != nil {
			continue
		}
		if env := msg.GetEvent(); env != nil && env.GetMatchEnded() != nil {
			return env.GetMatchEnded()
		}
	}
}

// TestPvpResignEndsMatchAndSettles（stub 门控）：双认证 PvP 中 p1 投降——
// 双方收到同一条 MatchEnded（p2 胜 / reason=resign / 双方 ELO 附加），
// 既有 settle 全链以 end_reason='resign' 落库，引擎照常作废。
func TestPvpResignEndsMatchAndSettles(t *testing.T) {
	a, b := testUsers()
	now := time.Now()
	st := &stubStore{
		userByToken: map[string]store.User{"tok-a": a, "tok-b": b},
		d1:          store.EloDelta{Player: "p1", UserID: a.ID, Before: 1000, After: 984},
		d2:          store.EloDelta{Player: "p2", UserID: b.ID, Before: 1000, After: 1016},
		// 10 号票起 PvP 入口要求教程已完成（本测试只关心投降结算链路）。
		tutorial: map[uuid.UUID]store.TutorialProgress{
			a.ID: {Level: 4, Completed: true, CompletedAt: &now},
			b.ID: {Level: 4, Completed: true, CompletedAt: &now},
		},
	}
	h := New(passJudge{}, slog.Default(), nil)
	h.st = st
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ws1 := dialJoin(t, srv.URL, &matchpb.JoinRequest{Player: "p1", Token: "tok-a"})
	defer ws1.Close()
	readSnapshot(t, ws1) // WAITING
	ws2 := dialJoin(t, srv.URL, &matchpb.JoinRequest{Player: "p2", Token: "tok-b"})
	defer ws2.Close()
	readSnapshot(t, ws2) // ROUND_ACTIVE（双方就位即开赛）

	if err := ws1.WriteMessage(websocket.TextMessage, resignMsg()); err != nil {
		t.Fatalf("发送投降失败：%v", err)
	}
	me := waitEnded(t, ws1, 5*time.Second)
	if me.GetWinner() != "p2" || me.GetReason() != match.ReasonResign {
		t.Fatalf("p1 投降应判 p2 胜（reason=resign）：%v", me)
	}
	if len(me.GetElo()) != 2 {
		t.Fatalf("双认证投降终局应附双方 ELO：%v", me.GetElo())
	}
	// 第二窗口收到同一条终局（共享事件流）
	if me2 := waitEnded(t, ws2, 5*time.Second); !proto.Equal(me, me2) {
		t.Fatalf("双方应收到同一条 MatchEnded：%v vs %v", me, me2)
	}
	// 既有 settle 全链沿用：落库 reason/winner 透传 resign
	if st.lastRes == nil || st.lastRes.Reason != match.ReasonResign || st.lastRes.Winner != "p2" {
		t.Fatalf("投降应按 reason=resign 走既有结算落库：%+v", st.lastRes)
	}
	// 终局即作废引擎（下一场由下一条 join 重建）
	h.mu.Lock()
	eng := h.engine
	h.mu.Unlock()
	if eng != nil {
		t.Fatal("投降终局后引擎应被置空")
	}
}

// TestPveResignBotWinsNoRows：PvE 投降 → bot（p2）胜；终局广播零落库（stub 未被
// 调用、ELO 字段为空），Bot 与引擎随终局停机清场。
func TestPveResignBotWinsNoRows(t *testing.T) {
	a, _ := testUsers()
	st := &stubStore{userByToken: map[string]store.User{"tok-a": a}}
	h := New(passJudge{}, slog.Default(), nil)
	h.st = st
	h.botClock = idleClock{}
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ws := dialJoin(t, srv.URL, &matchpb.JoinRequest{Player: "p1", Token: "tok-a", Mode: "pve", Bot: "easy"})
	defer ws.Close()
	readSnapshot(t, ws) // 单连接即开赛

	if err := ws.WriteMessage(websocket.TextMessage, resignMsg()); err != nil {
		t.Fatalf("发送投降失败：%v", err)
	}
	me := waitEnded(t, ws, 5*time.Second)
	if me.GetWinner() != "p2" || me.GetReason() != match.ReasonResign {
		t.Fatalf("PvE 投降应判 bot（p2）胜、reason=resign：%v", me)
	}
	if len(me.GetElo()) != 0 {
		t.Fatalf("PvE 终局不得携带 ELO：%v", me.GetElo())
	}
	if st.lastRes != nil {
		t.Fatalf("PvE 投降不得落库：%+v", st.lastRes)
	}
	h.mu.Lock()
	eng, ctl := h.engine, h.botCtl
	h.mu.Unlock()
	if eng != nil || ctl != nil {
		t.Fatalf("PvE 投降终局后引擎/Bot 应置空：engine=%v bot=%v", eng != nil, ctl != nil)
	}
}

// TestTutorialResignRejected：教程局投降被引擎拒绝——无 MatchEnded、连接不断、
// 对局照常活跃（接受后续正常提交的引擎仍在 ROUND_ACTIVE）。
func TestTutorialResignRejected(t *testing.T) {
	a, _ := testUsers()
	st := &stubStore{
		userByToken: map[string]store.User{"tok-a": a},
		// 教程进度：已完成第 1 关 → 可进入第 2 关
		tutorial: map[uuid.UUID]store.TutorialProgress{a.ID: {Level: 1}},
	}
	h := New(passJudge{}, slog.Default(), nil)
	h.st = st
	h.botClock = idleClock{}
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ws := dialJoin(t, srv.URL, &matchpb.JoinRequest{Player: "p1", Token: "tok-a", Mode: "tutorial", Level: "2"})
	defer ws.Close()
	readSnapshot(t, ws)

	if err := ws.WriteMessage(websocket.TextMessage, resignMsg()); err != nil {
		t.Fatalf("发送投降失败：%v", err)
	}
	// 拒绝 = 零新事件：窗口内读到的任何事件流都不得出现 MatchEnded；
	// 读超时（无更多事件）即为通过。连接本身保持打开。
	_ = ws.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			if ne, ok := err.(net.Error); !ok || !ne.Timeout() {
				t.Fatalf("教程连接不应被投降拒绝断开：%v", err)
			}
			break // 读超时：窗口内无新事件
		}
		var msg matchpb.ServerMessage
		if err := (protojson.UnmarshalOptions{}).Unmarshal(data, &msg); err != nil {
			continue
		}
		if env := msg.GetEvent(); env != nil && env.GetMatchEnded() != nil {
			t.Fatal("教程局投降不得产生 MatchEnded")
		}
	}
	// 拒绝不是作废：对局仍活跃
	h.mu.Lock()
	eng := h.engine
	h.mu.Unlock()
	if eng == nil {
		t.Fatal("教程投降被拒后引擎不得被作废")
	}
	if eng.Phase() != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE {
		t.Fatalf("教程投降被拒后对局应仍活跃，实际 %v", eng.Phase())
	}
}

// TestRealStoreResignRecordsReason（CODEWAR_TEST_DSN 门控）：真实 store 落库断言
// ——matches 一行 end_reason='resign' + elo_history 恰两行 + users.elo 随结算
// 变动（胜涨败跌、零和）。
func TestRealStoreResignRecordsReason(t *testing.T) {
	dsn := integrationDSN(t)
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("打开 store 失败（含迁移）：%v", err)
	}
	defer st.Close()

	ua, ub, tokA, tokB := registerPair(t, st, "resign")
	completeTutorial(t, st, ua, ub)

	h := New(passJudge{}, slog.Default(), st)
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ws1 := dialWS(t, srv.URL, "p1", tokA)
	defer ws1.Close()
	ws2 := dialWS(t, srv.URL, "p2", tokB)
	defer ws2.Close()

	if err := ws1.WriteMessage(websocket.TextMessage, resignMsg()); err != nil {
		t.Fatalf("发送投降失败：%v", err)
	}
	me := waitEnded(t, ws1, 10*time.Second)
	if me.GetWinner() != "p2" || me.GetReason() != match.ReasonResign {
		t.Fatalf("投降终局应为 p2 胜（resign）：%v", me)
	}
	if len(me.GetElo()) != 2 {
		t.Fatalf("真实结算应附双方 ELO：%v", me.GetElo())
	}
	waitEnded(t, ws2, 10*time.Second)

	matchID := h.matchID
	var dbWinner *string
	var dbReason string
	if err := st.DB().QueryRowContext(ctx,
		`SELECT winner, end_reason FROM matches WHERE id = $1`, matchID,
	).Scan(&dbWinner, &dbReason); err != nil {
		t.Fatalf("matches 行缺失：%v", err)
	}
	if dbWinner == nil || *dbWinner != "p2" || dbReason != match.ReasonResign {
		t.Fatalf("matches 行应为 winner=p2 / end_reason='resign'：%v %v", dbWinner, dbReason)
	}
	var histCount int
	if err := st.DB().QueryRowContext(ctx,
		`SELECT count(*) FROM elo_history WHERE match_id = $1`, matchID).Scan(&histCount); err != nil {
		t.Fatal(err)
	}
	if histCount != 2 {
		t.Fatalf("elo_history 应恰两行，实际 %d", histCount)
	}
	var afterA, afterB int32
	if err := st.DB().QueryRowContext(ctx,
		`SELECT elo_after FROM elo_history WHERE match_id = $1 AND player = 'p1'`, matchID).Scan(&afterA); err != nil {
		t.Fatal(err)
	}
	if err := st.DB().QueryRowContext(ctx,
		`SELECT elo_after FROM elo_history WHERE match_id = $1 AND player = 'p2'`, matchID).Scan(&afterB); err != nil {
		t.Fatal(err)
	}
	if afterA >= afterB {
		t.Fatalf("投降败方 ELO 应低于胜方：p1=%d p2=%d", afterA, afterB)
	}
	var eloA, eloB int32
	if err := st.DB().QueryRowContext(ctx, `SELECT elo FROM users WHERE id = $1`, ua.ID).Scan(&eloA); err != nil {
		t.Fatal(err)
	}
	if err := st.DB().QueryRowContext(ctx, `SELECT elo FROM users WHERE id = $1`, ub.ID).Scan(&eloB); err != nil {
		t.Fatal(err)
	}
	if eloA != afterA || eloB != afterB {
		t.Fatalf("users.elo 未随投降结算更新：users(%d,%d) history(%d,%d)", eloA, eloB, afterA, afterB)
	}
}
