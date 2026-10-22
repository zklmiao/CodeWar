package hub

// A3 集成测试（07 号票审查补齐）：真实 *store.Store 串真实 hub——
// 此前 hub_elo_test 全部走 stubStore，只能断言"桩被怎么调"，真实 SQL 事务、
// elo_history、users.elo 与 MatchEnded 事件的一致性从未验证过。
//
// 门控与隔离：CODEWAR_TEST_DSN 必填且库名须含 test；本测试用**专属库**
// codewar_test_hub（缺失则自动 CREATE DATABASE），与 store 包测试的
// TRUNCATE 互不干扰（go test 并行跑不同包的二进制）。

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/match"
	"codewar/server/internal/store"
	_ "github.com/lib/pq"
)

func integrationDSN(t *testing.T) string {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv("CODEWAR_TEST_DSN"))
	if raw == "" {
		t.Skip("CODEWAR_TEST_DSN 未设置，跳过 hub×store 集成测试")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Path == "" {
		t.Skipf("DSN 无法解析，跳过：%q", raw)
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	if !strings.Contains(strings.ToLower(dbName), "test") {
		t.Skip("测试库名必须含 test（防误清共享库）")
	}
	// 专属库：与 store 包测试的 TRUNCATE 隔离。
	hubDB := "codewar_test_hub"
	admin, err := sql.Open("postgres", raw)
	if err != nil {
		t.Fatalf("管理连接失败：%v", err)
	}
	defer admin.Close()
	var exists bool
	if err := admin.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, hubDB,
	).Scan(&exists); err != nil {
		t.Fatalf("查询库存在性失败：%v", err)
	}
	if !exists {
		if _, err := admin.Exec(
			fmt.Sprintf(`CREATE DATABASE %s OWNER codewar`, hubDB)); err != nil {
			t.Fatalf("创建专属库失败：%v", err)
		}
	}
	u.Path = "/" + hubDB
	return u.String()
}

func dialWS(t *testing.T, srvURL, player, token string) *websocket.Conn {
	t.Helper()
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srvURL, "http")+"/ws", nil)
	if err != nil {
		t.Fatalf("[%s] 拨号失败：%v", player, err)
	}
	join, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Join{
		Join: &matchpb.JoinRequest{Player: player, Token: token},
	}})
	if err := ws.WriteMessage(websocket.TextMessage, join); err != nil {
		t.Fatalf("[%s] join 发送失败：%v", player, err)
	}
	return ws
}

// registerPair：注册两名真实用户（真 bcrypt + SQL），返回用户与令牌。
func registerPair(t *testing.T, st *store.Store, tag string) (ua, ub store.User, tokA, tokB string) {
	t.Helper()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1e13)
	ua, tokA, err := st.Register(ctx, "hub-intg-a-"+tag+"-"+suffix, "password-123")
	if err != nil {
		t.Fatalf("注册 A 失败：%v", err)
	}
	ub, tokB, err = st.Register(ctx, "hub-intg-b-"+tag+"-"+suffix, "password-123")
	if err != nil {
		t.Fatalf("注册 B 失败：%v", err)
	}
	return ua, ub, tokA, tokB
}

func collectEnds(t *testing.T, ends chan *matchpb.MatchEnded, n int, timeout time.Duration) []*matchpb.MatchEnded {
	t.Helper()
	collected := make([]*matchpb.MatchEnded, 0, n)
	deadline := time.After(timeout)
	for len(collected) < n {
		select {
		case me := <-ends:
			if me == nil {
				t.Fatal("对局中连接断裂")
			}
			collected = append(collected, me)
		case <-deadline:
			t.Fatalf("%ds 内未收齐 %d 个终局事件（已收 %d）", int(timeout.Seconds()), n, len(collected))
		}
	}
	return collected
}

// contentJudge：只对 okCode 判过——制造不对称伤害（一方秒解得手、另一方
// 连续错答自损）的确定性工具；按提交内容区分，Judge 接口本身无玩家语义。
type contentJudge struct{ okCode string }

func (j contentJudge) Judge(_ context.Context, code string, _ []match.TestCase) match.JudgeResult {
	if code == j.okCode {
		return match.JudgeResult{Passed: true, Category: match.CategoryOK}
	}
	return match.JudgeResult{Passed: false, Category: match.CategoryWrongAnswer}
}

// selfDestructLoop：P2 策略——血量制导的自损：仅当 lastHP > 22 时提交毒码
// （降入 P1 一击必杀区间 [1,22] 即停手），使 P1 的回合超时攻击成为致命一击
// → reason=ko。每次错误 -3 HP + 5 秒锁（约 6s 一步）；P1 满血不受威胁。
func selfDestructLoop(t *testing.T, ws *websocket.Conn, player string, ends chan<- *matchpb.MatchEnded) {
	t.Helper()
	_ = ws.SetReadDeadline(time.Now().Add(400 * time.Second))
	lastHP := int32(100) // 引擎初始 HP（spec 附注）
	wrongs := 0
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			t.Errorf("[%s] 读事件失败: %v", player, err)
			ends <- nil
			return
		}
		var msg matchpb.ServerMessage
		if err := (protojson.UnmarshalOptions{}).Unmarshal(data, &msg); err != nil {
			continue
		}
		if env := msg.GetEvent(); env != nil {
			if hp := env.GetHpChanged(); hp != nil {
				lastHP = hp.GetHp()
			}
			if (env.GetPanelReady() != nil || env.GetSubmissionJudged() != nil) &&
				lastHP > 22 && wrongs < 60 { // 上限兜底：防对局异常时的死循环
				sub, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Submit{
					Submit: &matchpb.SubmitRequest{Code: "poison", Action: "attack"},
				}})
				if err := ws.WriteMessage(websocket.TextMessage, sub); err == nil {
					wrongs++
				}
				time.Sleep(5400 * time.Millisecond) // 等锁过期（服务端 5s）
			}
			if me := env.GetMatchEnded(); me != nil {
				ends <- me
				return
			}
		}
	}
}

// completeTutorial：10 号票起 PvP 入口要求教程已完成；既有结算集成测试
// 通过真实 store 把双方进度推到通关（同时覆盖 AdvanceTutorial 的真实 SQL）。
func completeTutorial(t *testing.T, st *store.Store, users ...store.User) {
	t.Helper()
	for _, u := range users {
		if _, err := st.AdvanceTutorial(context.Background(), u.ID, 4, 4); err != nil {
			t.Fatalf("推进教程进度失败：%v", err)
		}
	}
}

// TestHubRealStoreKOSettlesEloAndRecords：KO 分胜负的完整真实链路。
// 断言四层一致：MatchEnded 事件 elo ≡ elo_history ≡ users.elo ≡ GetProfile。
func TestHubRealStoreKOSettlesEloAndRecords(t *testing.T) {
	dsn := integrationDSN(t)
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("打开 store 失败（含迁移）：%v", err)
	}
	defer st.Close()

	ua, ub, tokA, tokB := registerPair(t, st, "ko")
	completeTutorial(t, st, ua, ub)
	beforeA, beforeB := ua.Elo, ub.Elo

	h := New(contentJudge{okCode: "code"}, slog.Default(), st)
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ends := make(chan *matchpb.MatchEnded, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		ws := dialWS(t, srv.URL, "p1", tokA)
		defer ws.Close()
		playLoop(t, ws, "p1", ends)
	}()
	go func() {
		defer wg.Done()
		ws := dialWS(t, srv.URL, "p2", tokB)
		defer ws.Close()
		selfDestructLoop(t, ws, "p2", ends)
	}()
	wg.Wait()
	collected := collectEnds(t, ends, 2, 300*time.Second)
	me := collected[0]
	if me.GetWinner() != "p1" || me.GetReason() != "ko" {
		t.Fatalf("P2 自损殆尽应被 P1 先手 KO：winner=%q reason=%q", me.GetWinner(), me.GetReason())
	}
	if len(me.GetElo()) != 2 {
		t.Fatalf("终局应附双方 ELO：%v", me.GetElo())
	}
	dP1, dP2 := me.GetElo()[0], me.GetElo()[1]
	if dP1.GetEloAfter() <= dP1.GetEloBefore() || dP2.GetEloAfter() >= dP2.GetEloBefore() {
		t.Fatalf("胜者应涨分、败者应跌分：%+v / %+v", dP1, dP2)
	}
	if dP1.GetEloAfter()-dP1.GetEloBefore() != dP2.GetEloBefore()-dP2.GetEloAfter() {
		t.Fatalf("零和被破坏：%+v / %+v", dP1, dP2)
	}

	// DB 三层一致性（matchID 经事件 happens-before 可安全读取）
	matchID := h.matchID
	var (
		dbWinner *string
		dbReason string
		p1u      uuid.UUID
		p2u      uuid.UUID
	)
	if err := st.DB().QueryRowContext(ctx,
		`SELECT winner, end_reason, p1_user_id, p2_user_id FROM matches WHERE id = $1`, matchID,
	).Scan(&dbWinner, &dbReason, &p1u, &p2u); err != nil {
		t.Fatalf("matches 行缺失：%v", err)
	}
	if dbWinner == nil || *dbWinner != "p1" || dbReason != "ko" || p1u != ua.ID || p2u != ub.ID {
		t.Fatalf("matches 行不符：%v %v %v %v", dbWinner, dbReason, p1u, p2u)
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
	if int(afterA) != int(dP1.GetEloAfter()) || int(afterB) != int(dP2.GetEloAfter()) {
		t.Fatalf("elo_history 与 MatchEnded 不一致：DB(%d,%d) 事件(%d,%d)",
			afterA, afterB, dP1.GetEloAfter(), dP2.GetEloAfter())
	}
	var eloA, eloB int32
	if err := st.DB().QueryRowContext(ctx, `SELECT elo FROM users WHERE id = $1`, ua.ID).Scan(&eloA); err != nil {
		t.Fatal(err)
	}
	if err := st.DB().QueryRowContext(ctx, `SELECT elo FROM users WHERE id = $1`, ub.ID).Scan(&eloB); err != nil {
		t.Fatal(err)
	}
	if eloA != afterA || eloB != afterB {
		t.Fatalf("users.elo 未随结算更新：users(%d,%d) history(%d,%d)", eloA, eloB, afterA, afterB)
	}

	// GetProfile（个人页数据源）经真实链路可见本场与最新 ELO
	prof, err := st.GetProfile(ctx, ua.ID, 10)
	if err != nil {
		t.Fatalf("GetProfile 失败：%v", err)
	}
	if prof.Elo != afterA {
		t.Fatalf("profile ELO 应为结算后值 %d，实际 %d", afterA, prof.Elo)
	}
	if prof.Total != 1 || prof.Wins != 1 {
		t.Fatalf("profile 战绩应为 1 战 1 胜：%+v", prof)
	}
	if len(prof.Recent) == 0 {
		t.Fatal("profile 最近对局为空")
	}
	_ = beforeA
	_ = beforeB
}

// TestHubRealStoreDrawKeepsElo：平局 +0 硬规则穿透真实链路——
// 对称秒解判平后，users.elo 分毫不动，但 elo_history 仍记 delta=0 两行。
func TestHubRealStoreDrawKeepsElo(t *testing.T) {
	dsn := integrationDSN(t)
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("打开 store 失败：%v", err)
	}
	defer st.Close()

	ua, ub, tokA, tokB := registerPair(t, st, "draw")
	completeTutorial(t, st, ua, ub)
	baseA, baseB := ua.Elo, ub.Elo

	h := New(passJudge{}, slog.Default(), st)
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ends := make(chan *matchpb.MatchEnded, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		ws := dialWS(t, srv.URL, "p1", tokA)
		defer ws.Close()
		playLoop(t, ws, "p1", ends)
	}()
	go func() {
		defer wg.Done()
		ws := dialWS(t, srv.URL, "p2", tokB)
		defer ws.Close()
		playLoop(t, ws, "p2", ends)
	}()
	wg.Wait()
	collected := collectEnds(t, ends, 2, 30*time.Second)
	me := collected[0]
	if me.GetWinner() != "" {
		t.Fatalf("对称秒解应判平局，实际 winner=%q", me.GetWinner())
	}
	for _, d := range me.GetElo() {
		if d.GetEloBefore() != d.GetEloAfter() {
			t.Fatalf("平局必须 +0：%+v", d)
		}
	}

	matchID := h.matchID
	var eloA, eloB int32
	if err := st.DB().QueryRowContext(ctx, `SELECT elo FROM users WHERE id = $1`, ua.ID).Scan(&eloA); err != nil {
		t.Fatal(err)
	}
	if err := st.DB().QueryRowContext(ctx, `SELECT elo FROM users WHERE id = $1`, ub.ID).Scan(&eloB); err != nil {
		t.Fatal(err)
	}
	if int(eloA) != int(baseA) || int(eloB) != int(baseB) {
		t.Fatalf("平局后 users.elo 必须原值：(%d,%d) → (%d,%d)", baseA, baseB, eloA, eloB)
	}
	var histCount int
	if err := st.DB().QueryRowContext(ctx,
		`SELECT count(*) FROM elo_history WHERE match_id = $1`, matchID).Scan(&histCount); err != nil {
		t.Fatal(err)
	}
	if histCount != 2 {
		t.Fatalf("平局也应各记一行 delta=0 历史，实际 %d 行", histCount)
	}
	var before, after int32
	if err := st.DB().QueryRowContext(ctx,
		`SELECT elo_before, elo_after FROM elo_history WHERE match_id = $1 AND player = 'p1'`, matchID).Scan(&before, &after); err != nil {
		t.Fatal(err)
	}
	if before != after || int(before) != int(baseA) {
		t.Fatalf("平局历史行应为 delta=0：before=%d after=%d base=%d", before, after, baseA)
	}
}

// TestHubRealStoreAnonymousLeavesNoRows：B4 向后兼容对照——
// 双方匿名（无 token）接入时，即使 hub 接了真实 store，
// 也不落任何库行、MatchEnded 不带 elo（与 01 号票行为逐项等价）。
func TestHubRealStoreAnonymousLeavesNoRows(t *testing.T) {
	dsn := integrationDSN(t)
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("打开 store 失败：%v", err)
	}
	defer st.Close()

	h := New(passJudge{}, slog.Default(), st)
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ends := make(chan *matchpb.MatchEnded, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		ws := dialWS(t, srv.URL, "p1", "") // 匿名：无 token
		defer ws.Close()
		playLoop(t, ws, "p1", ends)
	}()
	go func() {
		defer wg.Done()
		ws := dialWS(t, srv.URL, "p2", "") // 匿名：无 token
		defer ws.Close()
		playLoop(t, ws, "p2", ends)
	}()
	wg.Wait()
	collected := collectEnds(t, ends, 2, 30*time.Second)
	me := collected[0]
	if len(me.GetElo()) != 0 {
		t.Fatalf("匿名对局不得携带 elo 字段：%v", me.GetElo())
	}
	matchID := h.matchID
	var matchRows, histRows int
	if err := st.DB().QueryRowContext(ctx,
		`SELECT count(*) FROM matches WHERE id = $1`, matchID).Scan(&matchRows); err != nil {
		t.Fatal(err)
	}
	if err := st.DB().QueryRowContext(ctx,
		`SELECT count(*) FROM elo_history WHERE match_id = $1`, matchID).Scan(&histRows); err != nil {
		t.Fatal(err)
	}
	if matchRows != 0 || histRows != 0 {
		t.Fatalf("匿名对局不得落库：matches=%d elo_history=%d", matchRows, histRows)
	}
}
