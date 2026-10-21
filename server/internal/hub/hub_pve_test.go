package hub

// 09 号票：单人 PvE（模拟 Bot）——配置解析、模式/配置一致性、单连接开局、
// Bot 标记、不落库、断线作废与第二名玩家拒绝。

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/bot"
	"codewar/server/internal/match"
	"codewar/server/internal/question"
	"codewar/server/internal/store"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
)

// fastClock：把 Bot 的解题延迟压缩到 2ms（对局在毫秒内推进；solveMs 语义的
// 真实断言在 bot 包的手动时钟集成测试里）。
type fastTimer struct{ t *time.Timer }

func (t fastTimer) Stop() bool { return t.t.Stop() }

type fastClock struct{ delay time.Duration }

func (fastClock) Now() time.Time { return time.Now() }
func (c fastClock) AfterFunc(_ time.Duration, f func()) bot.Timer {
	return fastTimer{t: time.AfterFunc(c.delay, f)}
}

// idleClock：Bot 永不行动（观察等待/断线清理路径时保持对局稳定）。
type idleClock struct{}

func (idleClock) Now() time.Time                            { return time.Now() }
func (idleClock) AfterFunc(time.Duration, func()) bot.Timer { return idleTimer{} }

type idleTimer struct{}

func (idleTimer) Stop() bool { return true }

// ===== 配置解析 =====

func TestParseModeAndBot(t *testing.T) {
	t.Run("pve + hard 采纳，其余配置保持默认锚点", func(t *testing.T) {
		log, _ := captureLogger()
		mc := parseMatchConfig(joinConfig{mode: "pve", bot: "hard"}, log)
		if mc.mode != modePvE || mc.botDifficulty != "hard" {
			t.Fatalf("模式/难度不符：%+v", mc)
		}
		if mc.tier != 3 || mc.strategy != question.StrategyClimb || mc.topics != nil {
			t.Fatalf("PvE 不得改变题型/cap/策略默认值：%+v", mc)
		}
	})
	t.Run("缺省 bot = normal（不 WARN）", func(t *testing.T) {
		log, buf := captureLogger()
		mc := parseMatchConfig(joinConfig{mode: " PVE "}, log)
		if mc.mode != modePvE || mc.botDifficulty != "normal" {
			t.Fatalf("默认难度应为 normal：%+v", mc)
		}
		if strings.Contains(buf.String(), "WARN") {
			t.Fatalf("缺省 bot 不应 WARN：%s", buf.String())
		}
	})
	t.Run("非法 bot 回退 normal 并 WARN", func(t *testing.T) {
		log, buf := captureLogger()
		mc := parseMatchConfig(joinConfig{mode: "pve", bot: "impossible"}, log)
		if mc.mode != modePvE || mc.botDifficulty != "normal" {
			t.Fatalf("非法 bot 应回退 normal：%+v", mc)
		}
		if !strings.Contains(buf.String(), "Bot 难度非法") {
			t.Fatalf("缺 WARN：%s", buf.String())
		}
	})
	t.Run("非法 mode 回退 PvP 并 WARN", func(t *testing.T) {
		log, buf := captureLogger()
		mc := parseMatchConfig(joinConfig{mode: "versus", bot: "hard"}, log)
		if mc.mode != "" || mc.botDifficulty != "" {
			t.Fatalf("非法 mode 应回退 PvP（无 bot）：%+v", mc)
		}
		if !strings.Contains(buf.String(), "模式非法") {
			t.Fatalf("缺 WARN：%s", buf.String())
		}
	})
	t.Run("无 pve 的 bot 参数忽略并 WARN", func(t *testing.T) {
		log, buf := captureLogger()
		mc := parseMatchConfig(joinConfig{bot: "hard"}, log)
		if mc.mode != "" || mc.botDifficulty != "" {
			t.Fatalf("非 PvE 不应携带 bot 难度：%+v", mc)
		}
		if !strings.Contains(buf.String(), "仅在 mode=pve") {
			t.Fatalf("缺 WARN：%s", buf.String())
		}
	})
	t.Run("anySet 覆盖 mode/bot", func(t *testing.T) {
		if !(joinConfig{mode: "pve"}).anySet() || !(joinConfig{bot: "easy"}).anySet() {
			t.Fatal("mode/bot 应计入后续 join 忽略判定")
		}
	})
}

// ===== 验收 4：PvE 与 PvP 使用同一套对局配置（同一构造器/默认值/seed 派生） =====

func TestPveAndPvpShareQuestionConfig(t *testing.T) {
	base := matchConfig{
		topicKey: "hash",
		topics:   []question.Topic{question.TopicHash},
		tier:     5,
		strategy: question.StrategyUniform,
		seed:     "parity-seed",
	}
	pvp := base
	pve := base
	pve.mode = modePvE
	pve.botDifficulty = "hard"
	fa, fb := newQuestions(pvp), newQuestions(pve)

	cmp := func(kind string, round int, a, b match.Question) {
		if a.ID != b.ID || a.Statement != b.Statement || a.Signature != b.Signature ||
			a.StarterCode != b.StarterCode || a.AnswerKey != b.AnswerKey ||
			!reflect.DeepEqual(a.Tests, b.Tests) {
			t.Fatalf("round %d %s 题在 PvE/PvP 下不一致：\nPvP=%+v\nPvE=%+v", round, kind, a, b)
		}
	}
	for round := 1; round <= match.MaxRounds; round++ {
		cmp("attack", round, fa.AttackQuestion(round), fb.AttackQuestion(round))
		for _, p := range []match.Player{match.P1, match.P2} {
			cmp("move", round, fa.MoveQuestion(round, p), fb.MoveQuestion(round, p))
			cmp("defend", round, fa.DefendQuestion(round, p), fb.DefendQuestion(round, p))
			cmp("skill", round, fa.SkillQuestion(round, p), fb.SkillQuestion(round, p))
		}
	}

	// 默认值一致：PvE 不得改变题型/cap/策略的缺省锚点。
	log, _ := captureLogger()
	a := parseMatchConfig(joinConfig{topic: "hash", cap: "L5", strategy: "uniform"}, log)
	b := parseMatchConfig(joinConfig{topic: "hash", cap: "L5", strategy: "uniform", mode: "pve", bot: "easy"}, log)
	if a.topicKey != b.topicKey || a.tier != b.tier || a.strategy != b.strategy ||
		!reflect.DeepEqual(a.topics, b.topics) {
		t.Fatalf("PvE/PvP 配置默认值不一致：%+v vs %+v", a, b)
	}
}

// ===== 验收 1/3：单连接即开局，快照标记 Bot，整局可终局 =====

func TestPveSingleJoinStartsAndMarksBot(t *testing.T) {
	h := New(passJudge{}, slog.Default(), nil)
	h.botClock = fastClock{delay: 2 * time.Millisecond}
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	out := make(chan *probeResult, 1)
	// player 省略：PvE 人类固定 p1（服务端权威）
	go probeClient(t, srv, &matchpb.JoinRequest{
		Mode: "pve", Bot: "hard", Class: "ranger", Skill: "pierce",
	}, out, nil)

	var r *probeResult
	select {
	case r = <-out:
	case <-time.After(30 * time.Second):
		t.Fatal("30s 内 PvE 对局未终局（单连接应直接开赛）")
	}
	if r == nil {
		t.Fatal("对局中连接断裂")
	}
	if r.snapshot == nil {
		t.Fatal("未收到接入快照")
	}
	// 单连接即开赛：快照已是回合活跃态
	if r.snapshot.GetPhase() != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE || r.snapshot.GetRound() != 1 {
		t.Fatalf("PvE 单连接应立即开赛：phase=%v round=%d", r.snapshot.GetPhase(), r.snapshot.GetRound())
	}
	var p1, p2 *matchpb.PlayerView
	for _, pv := range r.snapshot.GetPlayers() {
		switch pv.GetId() {
		case "p1":
			p1 = pv
		case "p2":
			p2 = pv
		}
	}
	if p1 == nil || p2 == nil {
		t.Fatalf("快照缺玩家行：%v", r.snapshot.GetPlayers())
	}
	if p1.GetBot() || !p2.GetBot() {
		t.Fatalf("只有 p2 应为 Bot：p1.bot=%v p2.bot=%v", p1.GetBot(), p2.GetBot())
	}
	if p1.GetClassId() != "ranger" {
		t.Fatalf("人类职业应为 ranger，实际 %q", p1.GetClassId())
	}
	switch p2.GetClassId() {
	case "warrior", "mage", "ranger":
	default:
		t.Fatalf("Bot 职业应为三职业之一（开局揭示），实际 %q", p2.GetClassId())
	}
	cfg := r.snapshot.GetMatchConfig()
	if cfg.GetMode() != "pve" || cfg.GetBot() != "hard" || cfg.GetCap() != 3 || cfg.GetStrategy() != "climb" {
		t.Fatalf("快照配置应为 pve/hard + 默认 cap3/climb：%v", cfg)
	}
	if r.ended == nil {
		t.Fatal("整局未终局")
	}
	if r.ended.GetReason() == "" {
		t.Fatalf("终局应带原因：%v", r.ended)
	}
	if len(r.ended.GetElo()) != 0 {
		t.Fatalf("PvE 不得携带 ELO：%v", r.ended.GetElo())
	}
	// 终局即清场（下一场由下一条 join 重建）
	h.mu.Lock()
	eng, ctl := h.engine, h.botCtl
	h.mu.Unlock()
	if eng != nil || ctl != nil {
		t.Fatalf("终局后引擎/Bot 应置空：engine=%v bot=%v", eng != nil, ctl != nil)
	}
}

// ===== 验收 6：PvE 整局不落库（即使人类已认证） =====

func TestPveDoesNotRecordMatchOrElo(t *testing.T) {
	a, _ := testUsers()
	st := &stubStore{
		userByToken: map[string]store.User{"tok-a": a},
		d1:          store.EloDelta{Player: "p1", UserID: a.ID, Before: 1000, After: 1016},
	}
	h := New(passJudge{}, slog.Default(), nil)
	h.st = st
	h.botClock = fastClock{delay: 2 * time.Millisecond}
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	out := make(chan *probeResult, 1)
	go probeClient(t, srv, &matchpb.JoinRequest{
		Player: "p1", Token: "tok-a", Mode: "pve", Bot: "easy",
	}, out, nil)
	var r *probeResult
	select {
	case r = <-out:
	case <-time.After(30 * time.Second):
		t.Fatal("30s 内 PvE 对局未终局")
	}
	if r == nil || r.ended == nil {
		t.Fatal("PvE 对局未正常终局")
	}
	if len(r.ended.GetElo()) != 0 {
		t.Fatalf("PvE 终局不得携带 ELO：%v", r.ended.GetElo())
	}
	if st.lastRes != nil {
		t.Fatalf("PvE 不得落库（RecordMatch 被调用）：%+v", st.lastRes)
	}
}

// ===== PvP 契约不变：player 仍必填（空值仅在 mode=pve 下合法） =====

func TestPvpStillRequiresExplicitPlayer(t *testing.T) {
	h := New(nil, slog.Default(), nil)
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ws := dialJoin(t, srv.URL, &matchpb.JoinRequest{})
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("PvP 空 player 应被拒连（与 01~07 行为一致）")
	}
	if h.engine != nil {
		t.Fatal("被拒的连接不得创建引擎")
	}
}

// ===== 单人对局：第二名玩家（含 p2 / 不带 mode）一律拒绝 =====

func TestPveRejectsSecondPlayer(t *testing.T) {
	h := New(passJudge{}, slog.Default(), nil)
	h.botClock = idleClock{}
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	first := dialJoin(t, srv.URL, &matchpb.JoinRequest{Mode: "pve"})
	defer first.Close()
	readSnapshot(t, first)

	for name, join := range map[string]*matchpb.JoinRequest{
		"p2 显式":  {Player: "p2", Mode: "pve"},
		"p2 无模式": {Player: "p2"},
		"p1 重复":  {Player: "p1", Mode: "pve"},
	} {
		t.Run(name, func(t *testing.T) {
			ws := dialJoin(t, srv.URL, join)
			defer ws.Close()
			_ = ws.SetReadDeadline(time.Now().Add(3 * time.Second))
			if _, _, err := ws.ReadMessage(); err == nil {
				t.Fatal("PvE 对局中第二名玩家应被拒连")
			}
		})
	}
	h.mu.Lock()
	n := len(h.conns)
	h.mu.Unlock()
	if n != 1 {
		t.Fatalf("首位玩家连接应不受影响，conns=%d", n)
	}
}

// ===== 断线 = 对局作废（刷新重开） =====

func TestPveDisconnectAbortsAndRefreshRestarts(t *testing.T) {
	h := New(passJudge{}, slog.Default(), nil)
	h.botClock = idleClock{}
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	ws := dialJoin(t, srv.URL, &matchpb.JoinRequest{Mode: "pve"})
	readSnapshot(t, ws)
	h.mu.Lock()
	oldEng := h.engine
	hadBot := h.botCtl != nil
	h.mu.Unlock()
	if oldEng == nil || !hadBot {
		t.Fatal("PvE 接入后应已有引擎与 Bot")
	}

	ws.Close()
	deadline := time.Now().Add(3 * time.Second)
	for {
		h.mu.Lock()
		cleared := h.engine == nil && h.botCtl == nil && h.cfg.mode == ""
		h.mu.Unlock()
		if cleared {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("断线后 3s 内未作废对局并清空 Bot")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// hub 先清引用、再（释放 h.mu 后）调 Stop：ENDED 可能稍晚于引用清空。
	for oldEng.Phase() != matchpb.MatchPhase_MATCH_PHASE_ENDED {
		if time.Now().After(deadline) {
			t.Fatalf("作废引擎应为 ENDED 态，实际 %v", oldEng.Phase())
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 刷新重开：同一入口重新接入即新开一局
	ws2 := dialJoin(t, srv.URL, &matchpb.JoinRequest{Mode: "pve"})
	defer ws2.Close()
	snap := readSnapshot(t, ws2)
	if snap.GetPhase() != matchpb.MatchPhase_MATCH_PHASE_ROUND_ACTIVE || snap.GetRound() != 1 {
		t.Fatalf("刷新后应重开第 1 回合：phase=%v round=%d", snap.GetPhase(), snap.GetRound())
	}
	var botMarked bool
	for _, pv := range snap.GetPlayers() {
		if pv.GetId() == "p2" && pv.GetBot() {
			botMarked = true
		}
	}
	if !botMarked {
		t.Fatal("刷新后的新对局仍应标记 p2 为 Bot")
	}
}

// ===== 测试小工具 =====

func dialJoin(t *testing.T, srvURL string, join *matchpb.JoinRequest) *websocket.Conn {
	t.Helper()
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srvURL, "http")+"/ws", nil)
	if err != nil {
		t.Fatalf("拨号失败：%v", err)
	}
	b, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Join{Join: join}})
	if err := ws.WriteMessage(websocket.TextMessage, b); err != nil {
		t.Fatalf("发送 join 失败：%v", err)
	}
	return ws
}

func readSnapshot(t *testing.T, ws *websocket.Conn) *matchpb.Snapshot {
	t.Helper()
	_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("读快照失败：%v", err)
		}
		var msg matchpb.ServerMessage
		if err := (protojson.UnmarshalOptions{}).Unmarshal(data, &msg); err != nil {
			t.Fatalf("解析失败：%v", err)
		}
		if snap := msg.GetSnapshot(); snap != nil {
			return snap
		}
	}
}
