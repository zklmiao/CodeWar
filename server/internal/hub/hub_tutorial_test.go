package hub

// 10 号票：新手教程管线——配置解析、PvP 门禁两态、教程模式路由、
// 四关端到端与进度持久化（真 DB 门控）。

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/match"
	"codewar/server/internal/store"
	"codewar/server/internal/tutorial"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
)

// ===== 配置解析 =====

func TestParseTutorialConfig(t *testing.T) {
	log, buf := captureLogger()
	mc := parseMatchConfig(joinConfig{mode: "tutorial", level: "3"}, log)
	if mc.mode != modeTutorial || mc.level != 3 {
		t.Fatalf("教程模式/关号不符：%+v", mc)
	}
	if mc.tier != match.DefaultRiskTier {
		t.Fatalf("L1-L3 应固定 L3 风险档：%+v", mc)
	}
	if mc.topicKey != "" || mc.strategy != "climb" {
		t.Fatalf("教程应使用固定题包（忽略 topic/strategy）：%+v", mc)
	}
	if strings.Contains(buf.String(), "WARN") {
		t.Fatalf("合法教程配置不应 WARN：%s", buf.String())
	}

	// L4：允许 L3/L5，缺省 L5（越高越疼）。
	for _, tc := range []struct {
		cap  string
		want int
	}{
		{"", 5}, {"L5", 5}, {"l3", 3}, {"L4", 5},
	} {
		log, _ := captureLogger()
		mc := parseMatchConfig(joinConfig{mode: "tutorial", level: "4", cap: tc.cap}, log)
		if mc.tier != tc.want {
			t.Fatalf("L4 cap=%q 应解析为 L%d，实际 L%d", tc.cap, tc.want, mc.tier)
		}
	}

	// 非法关号回退第 1 关并 WARN；level 计入后续 join 忽略判定。
	log, buf = captureLogger()
	mc = parseMatchConfig(joinConfig{mode: "tutorial", level: "9"}, log)
	if mc.level != 1 || !strings.Contains(buf.String(), "教程关号非法") {
		t.Fatalf("非法关号应回退 1 并 WARN：%+v / %s", mc, buf.String())
	}
	if !(joinConfig{level: "2"}).anySet() {
		t.Fatal("level 应计入 anySet")
	}
}

// ===== 验收 1：PvP 门禁两态（未完成拒绝 / 完成放行） =====

func TestPvpGateTwoStates(t *testing.T) {
	a, _ := testUsers()
	st := &stubStore{userByToken: map[string]store.User{"tok-a": a}}
	h := New(passJudge{}, slog.Default(), nil)
	h.st = st
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	// 未完成：拒绝（服务端关闭连接，读失败）。
	ws := dialJoin(t, srv.URL, &matchpb.JoinRequest{Player: "p1", Token: "tok-a"})
	_ = ws.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("未完成教程的登录用户应被拒绝进入 PvP")
	}
	ws.Close()
	h.mu.Lock()
	eng := h.engine
	h.mu.Unlock()
	if eng != nil {
		t.Fatal("被门禁拒绝的 join 不得创建引擎")
	}
	waitNoConns(t, h)

	// 完成：放行（收到快照；WAITING 等第二人）。
	now := time.Now()
	st.tutorial = map[uuid.UUID]store.TutorialProgress{
		a.ID: {Level: 4, Completed: true, CompletedAt: &now},
	}
	ws2 := dialJoin(t, srv.URL, &matchpb.JoinRequest{Player: "p1", Token: "tok-a"})
	defer ws2.Close()
	snap := readSnapshot(t, ws2)
	if snap.GetPhase() != matchpb.MatchPhase_MATCH_PHASE_WAITING {
		t.Fatalf("完成教程后应放行 PvP：phase=%v", snap.GetPhase())
	}
}

// 匿名热座是实验室旁路：无 token 不查进度，也不被门禁拦截（既有链路不变）。
func TestPvpGateAllowsAnonymousHotseat(t *testing.T) {
	h := New(nil, slog.Default(), nil)
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()
	ws := dialJoin(t, srv.URL, &matchpb.JoinRequest{Player: "p1"})
	defer ws.Close()
	if snap := readSnapshot(t, ws); snap.GetPhase() != matchpb.MatchPhase_MATCH_PHASE_WAITING {
		t.Fatalf("匿名热座应照常进入 WAITING：%v", snap.GetPhase())
	}
}

// ===== 教程模式：登录要求 / 关卡解锁 / 固定职业与题包 =====

func TestTutorialRequiresLogin(t *testing.T) {
	h := New(passJudge{}, slog.Default(), nil)
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()
	ws := dialJoin(t, srv.URL, &matchpb.JoinRequest{Mode: "tutorial", Level: "1"})
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("未登录用户应被教程拒连（进度无处保存）")
	}
}

func TestTutorialLevelUnlock(t *testing.T) {
	a, _ := testUsers()
	st := &stubStore{
		userByToken: map[string]store.User{"tok-a": a},
		tutorial:    map[uuid.UUID]store.TutorialProgress{a.ID: {Level: 1}},
	}
	h := New(passJudge{}, slog.Default(), nil)
	h.st = st
	h.botClock = fastClock{delay: 2 * time.Millisecond}
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	// 跳过第 2 关直连第 3 关：拒绝。
	ws := dialJoin(t, srv.URL, &matchpb.JoinRequest{Mode: "tutorial", Level: "3", Token: "tok-a"})
	_ = ws.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("未解锁的第 3 关应被拒绝")
	}
	ws.Close()
	waitNoConns(t, h)

	// 第 2 关：放行；快照带固定职业与教程模式/关号。
	ws2 := dialJoin(t, srv.URL, &matchpb.JoinRequest{Mode: "tutorial", Level: "2", Token: "tok-a"})
	defer ws2.Close()
	var snap *matchpb.Snapshot
	var tut *matchpb.TutorialState
	deadline := time.Now().Add(5 * time.Second)
	for snap == nil || tut == nil {
		_ = ws2.SetReadDeadline(deadline)
		_, data, err := ws2.ReadMessage()
		if err != nil {
			t.Fatalf("读教程接入消息失败：%v", err)
		}
		var msg matchpb.ServerMessage
		if err := (protojson.UnmarshalOptions{}).Unmarshal(data, &msg); err != nil {
			t.Fatal(err)
		}
		if s := msg.GetSnapshot(); s != nil {
			snap = s
		}
		if st := msg.GetTutorial(); st != nil {
			tut = st
		}
	}
	cfg := snap.GetMatchConfig()
	if cfg.GetMode() != "tutorial" || cfg.GetTutorialLevel() != 2 {
		t.Fatalf("快照应带教程模式/关号：%v", cfg)
	}
	if tut.GetTitle() == "" || len(tut.GetGoals()) == 0 || tut.GetCompleted() {
		t.Fatalf("教程初始状态不完整：%v", tut)
	}
	var p1, p2 *matchpb.PlayerView
	for _, pv := range snap.GetPlayers() {
		switch pv.GetId() {
		case "p1":
			p1 = pv
		case "p2":
			p2 = pv
		}
	}
	if p1 == nil || p1.GetClassId() != "warrior" {
		t.Fatalf("教程人类应为固定战士：%v", p1)
	}
	if p2 == nil || !p2.GetBot() || p2.GetClassId() != "mage" {
		t.Fatalf("L2 Bot 应为法师且标记 bot：%v", p2)
	}
}

// ===== 验收 2：未达成目标不得完成（真引擎；L2 硬扛直线攻击） =====

func TestTutorialL2HitDoesNotComplete(t *testing.T) {
	a, _ := testUsers()
	st := &stubStore{
		userByToken: map[string]store.User{"tok-a": a},
		tutorial:    map[uuid.UUID]store.TutorialProgress{a.ID: {Level: 1}},
	}
	h := New(passJudge{}, slog.Default(), nil)
	h.st = st
	h.botClock = fastClock{delay: 2 * time.Millisecond}
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ws := dialJoin(t, srv.URL, &matchpb.JoinRequest{Mode: "tutorial", Level: "2", Token: "tok-a"})
	defer ws.Close()

	hurt := false
	var lastTut *matchpb.TutorialState
	acted := map[int]bool{}
	deadline := time.Now().Add(15 * time.Second)
	// 玩家原地攻击（不移动）→ 法师的直线攻击必然命中；打两回合后断言未完成。
	for {
		_ = ws.SetReadDeadline(deadline)
		_, data, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("读消息失败：%v", err)
		}
		var msg matchpb.ServerMessage
		if err := (protojson.UnmarshalOptions{}).Unmarshal(data, &msg); err != nil {
			t.Fatal(err)
		}
		if tut := msg.GetTutorial(); tut != nil {
			lastTut = tut
			if tut.GetCompleted() {
				t.Fatal("玩家硬扛直线攻击后不得完成 L2")
			}
			continue
		}
		env := msg.GetEvent()
		if env == nil {
			continue
		}
		if hp := env.GetHpChanged(); hp != nil && hp.GetPlayer() == "p1" && hp.GetDelta() < 0 {
			hurt = true
		}
		rs := env.GetRoundStarted()
		if rs == nil {
			continue
		}
		round := int(rs.GetRound())
		if round >= 3 {
			break // 前两回合均已硬扛并结算
		}
		if acted[round] {
			continue
		}
		acted[round] = true
		sub, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Submit{
			Submit: &matchpb.SubmitRequest{Code: "code", Action: "attack"},
		}})
		if err := ws.WriteMessage(websocket.TextMessage, sub); err != nil {
			t.Fatalf("提交失败：%v", err)
		}
	}
	if !hurt {
		t.Fatal("场景无效：玩家两回合均未受伤，不能证明硬扛路径")
	}
	if lastTut == nil || lastTut.GetCompleted() {
		t.Fatalf("未达成时不得下发完成态：%v", lastTut)
	}
	h.mu.Lock()
	pending := h.tutDone
	h.mu.Unlock()
	if pending {
		t.Fatal("未达成目标不得触发进度落库")
	}
	p := st.tutorial[a.ID]
	if p.Level != 1 || p.Completed {
		t.Fatalf("进度不得推进：%+v", p)
	}
}

// ===== 验收 1/2/4：四关端到端（真引擎 + 脚本 Bot + 真 DB） =====

// tutorialAction：测试客户端每回合的行动（与关卡教学目标一致）。
// code 固定为 "code"——passJudge 全过，本测试只关心行动/事件，不重复测判题。
func tutorialAction(level, round int) (action string, moveTo *matchpb.Position) {
	switch level {
	case 2:
		// 走出法师的直线列（P2 在 (2,2) 朝下打）；每回合换一个相邻合法格。
		if round%2 == 1 {
			return "move", &matchpb.Position{Row: 3, Col: 1}
		}
		return "move", &matchpb.Position{Row: 3, Col: 0}
	case 3:
		// 第 1 回合挂盾吃普攻；第 2 回合释放技能；之后继续防御保证存活。
		if round == 2 {
			return "skill", nil
		}
		return "defend", nil
	default:
		return "attack", nil
	}
}

// runTutorialLevel：接入一关并按其教学目标操作，返回服务端完成态。
func runTutorialLevel(t *testing.T, srvURL, token string, level int, cap string) *matchpb.TutorialState {
	t.Helper()
	ws := dialJoin(t, srvURL, &matchpb.JoinRequest{
		Mode: "tutorial", Level: fmt.Sprintf("%d", level), Cap: cap, Token: token,
	})
	defer ws.Close()

	var lastTut *matchpb.TutorialState
	acted := map[int]bool{}
	deadline := time.Now().Add(20 * time.Second)
	for {
		_ = ws.SetReadDeadline(deadline)
		_, data, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("L%d 读消息失败（未完成）：%v（最后教程状态：%v）", level, err, lastTut)
		}
		var msg matchpb.ServerMessage
		if err := (protojson.UnmarshalOptions{}).Unmarshal(data, &msg); err != nil {
			t.Fatalf("L%d 解析失败：%v", level, err)
		}
		if st := msg.GetTutorial(); st != nil {
			lastTut = st
			if st.GetCompleted() {
				return st
			}
			continue
		}
		env := msg.GetEvent()
		if env == nil {
			continue
		}
		if me := env.GetMatchEnded(); me != nil {
			t.Fatalf("L%d 教程对局意外终局：%+v", level, me)
		}
		rs := env.GetRoundStarted()
		if rs == nil {
			continue
		}
		round := int(rs.GetRound())
		if acted[round] {
			continue
		}
		acted[round] = true
		action, moveTo := tutorialAction(level, round)
		sub, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Submit{
			Submit: &matchpb.SubmitRequest{Code: "code", Action: action, MoveTo: moveTo},
		}})
		if err := ws.WriteMessage(websocket.TextMessage, sub); err != nil {
			t.Fatalf("L%d 提交失败：%v", level, err)
		}
	}
}

func waitNoConns(t *testing.T, h *Hub) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		h.mu.Lock()
		n := len(h.conns)
		h.mu.Unlock()
		if n == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("连接未在 5s 内清理：%d", n)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestTutorialFullRunUnlocksPvp：完整教程跑通 → 每关进度落库 → PvP 放行。
// 真 DB（CODEWAR_TEST_DSN 门控）+ 真引擎 + 脚本 Bot + 假判题（passJudge）。
func TestTutorialFullRunUnlocksPvp(t *testing.T) {
	dsn := integrationDSN(t)
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("打开 store 失败：%v", err)
	}
	defer st.Close()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1e13)
	u, tok, err := st.Register(ctx, "tutorial-e2e-"+suffix, "password-123")
	if err != nil {
		t.Fatalf("注册失败：%v", err)
	}

	h := New(passJudge{}, slog.Default(), st)
	h.botClock = fastClock{delay: 2 * time.Millisecond}
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	for level := 1; level <= tutorial.MaxLevel; level++ {
		cap := ""
		if level == 4 {
			cap = "L5" // 第 4 关教学难度收益（默认 L5）
		}
		stState := runTutorialLevel(t, srv.URL, tok, level, cap)
		if stState.GetLevel() != uint32(level) {
			t.Fatalf("完成态关号应为 %d：%v", level, stState)
		}
		if level == 4 {
			if !stState.GetAllCompleted() || stState.GetNextLevel() != 0 {
				t.Fatalf("最后一关完成态不符：%v", stState)
			}
			if stState.GetCap() != 5 || stState.GetCapCoeff() <= 1.0 {
				t.Fatalf("L4 应携带 L5 风险系数：cap=%d coeff=%v", stState.GetCap(), stState.GetCapCoeff())
			}
		} else if stState.GetNextLevel() != uint32(level+1) {
			t.Fatalf("L%d 的下一关应为 %d：%v", level, level+1, stState)
		}
		waitNoConns(t, h)

		// 进度已落库（完成态只在 DB 写成功后下发，所以这里必然可见）。
		prog, err := st.GetTutorialProgress(ctx, u.ID)
		if err != nil {
			t.Fatalf("读取进度失败：%v", err)
		}
		if int(prog.Level) != level {
			t.Fatalf("L%d 完成后进度应为 %d，实际 %d", level, level, prog.Level)
		}
		if level < tutorial.MaxLevel && prog.Completed {
			t.Fatalf("未到最后一关不应标记通关：%+v", prog)
		}
		if level == tutorial.MaxLevel && (!prog.Completed || prog.CompletedAt == nil) {
			t.Fatalf("通关最后一关应落完成时刻：%+v", prog)
		}
	}

	// 教程不落战绩/ELO：整轮跑完 matches 不应出现该用户。
	var n int
	if err := st.DB().QueryRowContext(ctx,
		`SELECT count(*) FROM matches WHERE p1_user_id = $1 OR p2_user_id = $1`, u.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("教程对局不得落战绩，实际 %d 行", n)
	}

	// 验收 1：完整教程跑通后 1v1 匹配放行（同一门禁位置）。
	waitNoConns(t, h)
	ws := dialJoin(t, srv.URL, &matchpb.JoinRequest{Player: "p1", Token: tok})
	defer ws.Close()
	snap := readSnapshot(t, ws)
	if snap.GetPhase() != matchpb.MatchPhase_MATCH_PHASE_WAITING {
		t.Fatalf("完成教程后 PvP 应放行：phase=%v", snap.GetPhase())
	}
}
