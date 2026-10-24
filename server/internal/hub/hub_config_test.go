package hub

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/match"
	"codewar/server/internal/question"
	"codewar/server/internal/question/templates"
	"codewar/server/internal/store"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
)

// ===== 对局配置：解析 / 首个 join 锁定 / 非法回退 / 快照展示 / 结算系数（05 号票）=====

// syncBuffer：并发安全的日志 sink。hub 协程的日志写入（readLoop defer 的
// 「玩家断开」等）与测试断言 buf.String() 并发访问同一缓冲，无锁的
// bytes.Buffer 在 -race 下报 DATA RACE；Write 与 String 均持锁，断言内容不变。
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func captureLogger() (*slog.Logger, *syncBuffer) {
	var buf syncBuffer
	return slog.New(slog.NewTextHandler(&buf, nil)), &buf
}

func TestParseMatchConfig(t *testing.T) {
	t.Run("合法值全量采纳", func(t *testing.T) {
		log, _ := captureLogger()
		mc := parseMatchConfig(joinConfig{topic: "hash", cap: "L5", strategy: "uniform"}, log)
		if mc.topicKey != "hash" || len(mc.topics) != 1 || mc.topics[0] != question.TopicHash {
			t.Fatalf("题型过滤不符：%+v", mc)
		}
		if mc.tier != 5 || mc.strategy != question.StrategyUniform {
			t.Fatalf("档位/策略不符：%+v", mc)
		}
	})
	t.Run("大小写与空白宽松清洗", func(t *testing.T) {
		log, _ := captureLogger()
		mc := parseMatchConfig(joinConfig{topic: " Hash ", cap: "l3", strategy: "CLIMB"}, log)
		if mc.topicKey != "hash" || mc.tier != 3 || mc.strategy != question.StrategyClimb {
			t.Fatalf("清洗后不符：%+v", mc)
		}
	})
	t.Run("空参数 = 默认锚点（全部题型 / L3 / climb）", func(t *testing.T) {
		log, buf := captureLogger()
		mc := parseMatchConfig(joinConfig{}, log)
		if mc.topicKey != "" || mc.topics != nil || mc.tier != 3 || mc.strategy != question.StrategyClimb {
			t.Fatalf("默认锚点不符：%+v", mc)
		}
		if strings.Contains(buf.String(), "WARN") {
			t.Fatalf("缺省链路不应产生警告日志：%s", buf.String())
		}
	})
	t.Run("非法值回退默认并 WARN（钉住回退语义）", func(t *testing.T) {
		log, buf := captureLogger()
		mc := parseMatchConfig(joinConfig{topic: "bogus", cap: "L9", strategy: "random"}, log)
		if mc.topicKey != "" || mc.topics != nil {
			t.Fatalf("非法 topic 应回退全部题型：%+v", mc)
		}
		if mc.tier != 3 || mc.strategy != question.StrategyClimb {
			t.Fatalf("非法 cap/strategy 应回退 L3/climb：%+v", mc)
		}
		for _, want := range []string{"topic 非法", "cap 非法", "strategy 非法"} {
			if !strings.Contains(buf.String(), want) {
				t.Fatalf("缺 WARN 日志 %q：%s", want, buf.String())
			}
		}
	})
	t.Run("anySet：后续 join 忽略日志的判定面", func(t *testing.T) {
		if (joinConfig{}).anySet() {
			t.Fatal("空参数 anySet 应为 false")
		}
		if !(joinConfig{cap: "L1"}).anySet() {
			t.Fatal("任一参数非空 anySet 应为 true")
		}
	})
}

// ===== WS 端到端 =====

var diffRe = regexp.MustCompile(`-(L[1-5])-`)

// idTemplate / idDifficulty：从 Question ID 提取模板段与难度档。
func idTemplate(qid string) string {
	if i := strings.IndexByte(qid, '@'); i >= 0 {
		return qid[:i]
	}
	return qid
}

func idDifficulty(t *testing.T, qid string) int {
	t.Helper()
	m := diffRe.FindStringSubmatch(idTemplate(qid))
	if m == nil {
		t.Fatalf("题 ID 无难度段：%q", qid)
	}
	return int(m[1][1] - '0')
}

// probeResult：单客户端观察到的事件流切片。
type probeResult struct {
	snapshot  *matchpb.Snapshot
	attackIDs []string
	hpDeltas  []int32 // HpChanged.cause=attack 的 delta（负值）；URL→伤害事件的端到端观测面
	ended     *matchpb.MatchEnded
}

// probeClient：接入 → 见面板即秒解攻击（passJudge 全过）→ 收集至终局。
// onSnapshot 非空时在收到快照（即已完成接入）后投递一帧——调用方用它把
// 两个 join 排序（首个 join 锁定配置，谁先到谁生效，必须确定性）。
// 出错 Errorf + 投递 nil，调用方统一判失败（快速失败不断言细节）。
func probeClient(t *testing.T, srv *httptest.Server, join *matchpb.JoinRequest, out chan<- *probeResult, onSnapshot chan<- *matchpb.Snapshot) {
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Errorf("拨号失败：%v", err)
		out <- nil
		return
	}
	defer ws.Close()
	data, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Join{Join: join}})
	if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Errorf("发送 join 失败：%v", err)
		out <- nil
		return
	}
	res := &probeResult{}
	_ = ws.SetReadDeadline(time.Now().Add(30 * time.Second))
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			t.Errorf("读事件失败: %v", err)
			out <- nil
			return
		}
		var msg matchpb.ServerMessage
		if err := (protojson.UnmarshalOptions{}).Unmarshal(data, &msg); err != nil {
			t.Errorf("解析失败: %v", err)
			out <- nil
			return
		}
		if snap := msg.GetSnapshot(); snap != nil {
			res.snapshot = snap
			if onSnapshot != nil {
				onSnapshot <- snap
			}
			continue
		}
		env := msg.GetEvent()
		if env == nil {
			continue
		}
		if pr := env.GetPanelReady(); pr != nil {
			res.attackIDs = append(res.attackIDs, pr.GetAttack().GetId())
			sub, _ := protojson.Marshal(&matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Submit{
				Submit: &matchpb.SubmitRequest{Code: "code", Action: "attack"}}})
			if err := ws.WriteMessage(websocket.TextMessage, sub); err != nil {
				t.Errorf("提交失败：%v", err)
				out <- nil
				return
			}
		}
		if hp := env.GetHpChanged(); hp != nil && hp.GetCause() == "attack" {
			res.hpDeltas = append(res.hpDeltas, hp.GetDelta())
		}
		if me := env.GetMatchEnded(); me != nil {
			res.ended = me
			out <- res
			return
		}
	}
}

// runConfiguredMatch：双方接入打满一局（对称秒解 → 数回合内平局）。
// 排序保证：j1 收到快照后 j2 才拨号——「首个 join 锁定」语义下 j1 必然生效。
// 返回两个客户端的观察结果（共享事件流，attackIDs 一致；快照各带同源配置）。
func runConfiguredMatch(t *testing.T, h *Hub, j1, j2 *matchpb.JoinRequest) (*probeResult, *probeResult) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()
	out := make(chan *probeResult, 2)
	snapCh := make(chan *matchpb.Snapshot, 1)
	go probeClient(t, srv, j1, out, snapCh)
	var snap1 *matchpb.Snapshot
	select {
	case snap1 = <-snapCh:
	case <-time.After(10 * time.Second):
		t.Fatal("首个接入未收到快照")
	}
	go probeClient(t, srv, j2, out, nil)
	var r1, r2 *probeResult
	timeout := time.After(30 * time.Second)
	for r1 == nil || r2 == nil {
		select {
		case r := <-out:
			if r == nil {
				t.Fatal("对局中连接断裂（probeClient 已报明细）")
			}
			if r1 == nil {
				r1 = r
			} else {
				r2 = r
			}
		case <-timeout:
			t.Fatal("30s 内对局未终局")
		}
	}
	if r1.snapshot == nil {
		r1.snapshot = snap1
	}
	return r1, r2
}

// 验收 1/2 端到端：首个 join 的 topic=hash 锁定整场；第二窗口携带冲突参数
// （topic=sort&cap=L1&strategy=uniform）被忽略并记日志；全部攻击题命中哈希
// 题型；两窗口快照收到同一服务端锁定配置。
func TestJoinConfigLocksOnFirstJoin(t *testing.T) {
	log, buf := captureLogger()
	h := New(passJudge{}, log, nil)
	r1, r2 := runConfiguredMatch(t, h,
		&matchpb.JoinRequest{Player: "p1", Topic: "hash", Cap: "L3", Strategy: "climb"},
		&matchpb.JoinRequest{Player: "p2", Topic: "sort", Cap: "L1", Strategy: "uniform"},
	)

	hashAttacks := map[string]bool{}
	for _, tpl := range templates.All() {
		m := tpl.Meta()
		if m.Topic == question.TopicHash && m.Action == question.ActionAttack {
			hashAttacks[m.ID] = true
		}
	}
	if len(r1.attackIDs) == 0 {
		t.Fatal("未收集到攻击题")
	}
	for i, qid := range r1.attackIDs {
		if !hashAttacks[idTemplate(qid)] {
			t.Fatalf("round %d 攻击题 %q 越出哈希题型（第二窗口冲突参数必须被忽略）", i+1, qid)
		}
	}
	if r1.ended.GetWinner() != "" {
		t.Fatalf("对称秒解应平局，实际 %q", r1.ended.GetWinner())
	}
	// 默认锚点的伤害事件流：cap=L3 → 系数 1.0，每回合 22（05 之前行为逐字节一致）
	for i, d := range r1.hpDeltas {
		if d != -22 {
			t.Fatalf("默认配置第 %d 次攻击伤害应为 22，实际 %d", i+1, -d)
		}
	}
	// 两窗口快照都携带服务端锁定的配置（对局内展示不撒谎）
	for name, snap := range map[string]*matchpb.Snapshot{"p1": r1.snapshot, "p2": r2.snapshot} {
		cfg := snap.GetMatchConfig()
		if cfg == nil {
			t.Fatalf("%s 快照缺 match_config", name)
		}
		if cfg.GetTopic() != "hash" || cfg.GetCap() != 3 || cfg.GetStrategy() != "climb" {
			t.Fatalf("%s 快照配置应为 hash/3/climb（首个 join 锁定）：%v", name, cfg)
		}
	}
	if !strings.Contains(buf.String(), "已忽略") || !strings.Contains(buf.String(), "sort") {
		t.Fatalf("后续 join 冲突参数应记忽略日志：%s", buf.String())
	}
}

// 错误路径：非法 topic/cap/strategy 全部回退默认（全部题型 / L3 / climb）
// 并逐项 WARN；对局照常进行。难度只断言 ≤3（全部题型下 L1 格仅语法有模板，
// 首回合就近平移到 L2 属预期，realized 首档不具确定性——确定性序列断言在
// 固定题型的 TestJoinCapL5RaisesDifficultyCeiling）。
func TestJoinInvalidConfigFallsBackToDefaults(t *testing.T) {
	log, buf := captureLogger()
	h := New(passJudge{}, log, nil)
	r1, _ := runConfiguredMatch(t, h,
		&matchpb.JoinRequest{Player: "p1", Topic: "bogus", Cap: "L9", Strategy: "zzz"},
		&matchpb.JoinRequest{Player: "p2"},
	)
	for i, qid := range r1.attackIDs {
		if d := idDifficulty(t, qid); d > 3 {
			t.Fatalf("round %d 难度 %d 超出默认 cap L3（L9 未被钳制）", i+1, d)
		}
	}
	cfg := r1.snapshot.GetMatchConfig()
	if cfg.GetTopic() != "" || cfg.GetCap() != 3 || cfg.GetStrategy() != "climb" {
		t.Fatalf("回退后快照配置应为全部/L3/climb：%v", cfg)
	}
	for _, want := range []string{"topic 非法", "cap 非法", "strategy 非法"} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("缺 WARN 日志 %q：%s", want, buf.String())
		}
	}
}

// 验收 1（cap 端到端）：topic=hash 固定题型下 cap=L5 + 爬坡——哈希攻击模板
// L1-L5 齐备（12 号票矩阵收官），realized 爬坡序列 [1,2,2,3]：round1 合法落
// L1 正是爬坡"开局低档"的设计本意（05 时期期望 L2 是哈希缺 L1 时的就近平移
// workaround，矩阵补齐后该假设过时）；第 4 回合出现 L3，与默认 cap=L3 的
// 序列确定性分化，证明 cap 真实放宽了出题区间上限。
func TestJoinCapL5RaisesDifficultyCeiling(t *testing.T) {
	h := New(passJudge{}, slog.Default(), nil)
	r1, r2 := runConfiguredMatch(t, h,
		&matchpb.JoinRequest{Player: "p1", Topic: "hash", Cap: "L5", Strategy: "climb"},
		&matchpb.JoinRequest{Player: "p2"},
	)
	want := []int{1, 2, 2, 3} // 26 伤害/回合（22×1.2，伤害与抽取档位无关）→ 第 4 回合双双归零
	if len(r1.attackIDs) != len(want) {
		t.Fatalf("应打满 %d 回合，实际 %d 回合", len(want), len(r1.attackIDs))
	}
	for i, qid := range r1.attackIDs {
		tpl := idTemplate(qid)
		if !strings.HasPrefix(tpl, "hash-") {
			t.Fatalf("round %d 攻击题 %q 越出哈希题型", i+1, tpl)
		}
		if d := idDifficulty(t, qid); d != want[i] {
			t.Fatalf("round %d 难度应为 L%d（爬坡 cap=L5），实际 L%d（%q）", i+1, want[i], d, qid)
		}
	}
	// URL cap=L5 → 伤害事件流：riskCoeff 1.2 下普攻 26（round(22×1.2)）——
	// 验收 3/4 的 hub 端到端伤害断言（也解释了 4 回合终局：26×4 ≥ 100）
	for i, d := range r1.hpDeltas {
		if d != -26 {
			t.Fatalf("cap=L5 第 %d 次攻击伤害应为 26，实际 %d", i+1, -d)
		}
	}
	if r2.snapshot.GetMatchConfig().GetCap() != 5 {
		t.Fatalf("快照 cap 应为 5：%v", r2.snapshot.GetMatchConfig())
	}
}

// 验收 3（结算奖励面）：hub 把首个 join 锁定的档位换算成风险系数传给落库
// （乘算与四舍五入在 store/elo 层，另有纯函数与 DB 门控测试钉住）。
func TestSettleCarriesRiskCoeffByCap(t *testing.T) {
	cases := []struct {
		name string
		join *matchpb.JoinRequest
		want float64
	}{
		{"cap=L5", &matchpb.JoinRequest{Player: "p1", Token: "tok-a", Cap: "L5"}, match.RiskCoeffMax},
		{"cap=L1", &matchpb.JoinRequest{Player: "p1", Token: "tok-a", Cap: "L1"}, match.RiskCoeffMin},
		{"无配置=1.0", &matchpb.JoinRequest{Player: "p1", Token: "tok-a"}, 1.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, b := testUsers()
			now := time.Now()
			st := &stubStore{
				userByToken: map[string]store.User{"tok-a": a, "tok-b": b},
				d1:          store.EloDelta{Player: "p1", UserID: a.ID, Before: 1000, After: 1000},
				d2:          store.EloDelta{Player: "p2", UserID: b.ID, Before: 1000, After: 1000},
				// 10 号票起 PvP 入口要求教程已完成（本测试只关心风险系数落库）。
				tutorial: map[uuid.UUID]store.TutorialProgress{
					a.ID: {Level: 4, Completed: true, CompletedAt: &now},
					b.ID: {Level: 4, Completed: true, CompletedAt: &now},
				},
			}
			log, _ := captureLogger()
			h := New(passJudge{}, log, nil)
			h.st = st
			srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
			defer srv.Close()
			out := make(chan *probeResult, 2)
			snapCh := make(chan *matchpb.Snapshot, 1)
			// 排序：p1（带配置）完成接入后 p2 才拨号——首个 join 锁定语义
			go probeClient(t, srv, tc.join, out, snapCh)
			select {
			case <-snapCh:
			case <-time.After(10 * time.Second):
				t.Fatal("p1 接入未收到快照")
			}
			// P2 无配置（避免后续 join 忽略日志噪声；系数只由首个 join 决定）
			go probeClient(t, srv, &matchpb.JoinRequest{Player: "p2", Token: "tok-b"}, out, nil)
			timeout := time.After(30 * time.Second)
			for n := 0; n < 2; n++ {
				select {
				case r := <-out:
					if r == nil {
						t.Fatal("对局中连接断裂")
					}
				case <-timeout:
					t.Fatal("30s 内对局未终局")
				}
			}
			if st.lastRes == nil {
				t.Fatal("终局未落库")
			}
			diff := st.lastRes.RiskCoeff - tc.want
			if diff < -1e-9 || diff > 1e-9 {
				t.Fatalf("落库 RiskCoeff = %v, want %v", st.lastRes.RiskCoeff, tc.want)
			}
		})
	}
}
