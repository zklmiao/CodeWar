// Package hub：热座对局的 WebSocket 接入层（ADR-0002 事件流同步）。
// 本票拓扑：同一台机器两个浏览器窗口分别以 ?player=1/2 接入同一场本地对局；
// 在线对战（08 号票）复用同一引擎与事件流，仅连接拓扑不同。
// 注意：proto 消息含 oneof，必须走 protojson + 原始字节收发，
// gorilla 的 ReadJSON/WriteJSON（encoding/json）无法处理 oneof。
package hub

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // 热座同机；在线模式的来源校验归 08 号票
}

// conn：一条玩家连接。outbox 由 writePump 消费，避免引擎广播被慢客户端阻塞。
type conn struct {
	player match.Player
	ws     *websocket.Conn
	outbox chan []byte
	closed chan struct{}
	once   sync.Once
}

func (c *conn) send(b []byte) {
	select {
	case c.outbox <- b:
	case <-c.closed:
	}
}

func (c *conn) close() {
	c.once.Do(func() {
		close(c.closed)
		close(c.outbox)
	})
}

// settleTimeout：终局落库的 DB 预算（H3）。结算实测毫秒级，5s 只拦挂死；
// 超时走"落库失败降级"分支（无 ELO 广播，对局照常结束），不引入新失败模式。
const settleTimeout = 5 * time.Second

type Hub struct {
	mu     sync.Mutex
	engine *match.Engine
	judge  match.JudgePort
	conns  map[match.Player]*conn
	logger *slog.Logger

	// 07 号票：结果落库与 ELO 结算。
	// st 为 nil = 无库模式（纯匿名热座，与 01 号票行为一致）。
	st accountStore
	// 本场身份与计时：join 时按 token 解析用户，开赛时生成 matchID 并计时，
	// 终局时结算。随引擎置空一起清空（下一场重建）。
	matchID    uuid.UUID
	matchStart time.Time
	seats      map[match.Player]store.User
}

// accountStore：hub 需要的持久化子集（*store.Store 满足；测试注入桩）。
type accountStore interface {
	Authenticate(ctx context.Context, token string) (store.User, error)
	RecordMatch(ctx context.Context, res store.MatchResult) (store.EloDelta, store.EloDelta, error)
}

func New(judge match.JudgePort, logger *slog.Logger, st *store.Store) *Hub {
	h := &Hub{
		judge:  judge,
		conns:  make(map[match.Player]*conn),
		logger: logger,
		seats:  make(map[match.Player]store.User),
	}
	if st != nil {
		h.st = st
	}
	return h
}

// ServeWS：GET /ws 处理器。首条消息必须是 join。
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	player, token, err := h.readJoin(ws)
	if err != nil {
		ws.Close()
		return
	}

	h.mu.Lock()
	if _, taken := h.conns[player]; taken {
		h.mu.Unlock()
		h.reject(ws, player+" 已有玩家接入（本票不支持重连，08 号票处理）")
		return
	}
	// 07 号票：有 token 则鉴权建绑，无 token = 匿名热座（向后兼容）。
	// token 非空但无效/过期 → 直接拒连（客户端应重登后重试，不静默降级，
	// 否则"我以为已登录但 ELO 没结算"更难排查）。
	if token != "" {
		if h.st == nil {
			h.mu.Unlock()
			h.reject(ws, "服务端未接数据库，会话无法验证（请以匿名接入）")
			return
		}
		u, err := h.st.Authenticate(context.Background(), token)
		if err != nil {
			h.mu.Unlock()
			h.reject(ws, "会话无效或已过期，请重新登录")
			return
		}
		h.seats[player] = u
	}
	if h.engine == nil {
		// 04 号票：出题工厂替换硬编码桩（默认配置：全题型、上限 L3、爬坡；
		// 05 号票从 UI 喂真实 Config）。每局新 MatchSeed，连打两局题目不同。
		questions := question.NewFactory(question.Config{
			Strategy:  question.StrategyClimb,
			MatchSeed: question.NewMatchSeed(),
		})
		questions.RegisterAll(templates.All()...)
		h.engine = match.NewEngine(match.RealClock{}, questions, h.judge, h.broadcast)
	}
	c := &conn{player: player, ws: ws, outbox: make(chan []byte, 64), closed: make(chan struct{})}
	h.conns[player] = c
	eng := h.engine
	// 开赛必须在释放 h.mu 之后：StartMatch 会同步 emit 事件 → broadcast 再次请求 h.mu，
	// 持锁调用会自死锁（sync.Mutex 不可重入）。
	shouldStart := len(h.conns) == 2 && eng.Phase() == matchpb.MatchPhase_MATCH_PHASE_WAITING
	if shouldStart {
		// 07 号票：本场 ID 与开赛时刻（结果落库与时长用）。
		h.matchID = uuid.New()
		h.matchStart = time.Now()
	}
	h.mu.Unlock()

	if shouldStart {
		eng.StartMatch()
		go eng.Run() // 计时泵：服务端权威倒计时（ADR-0002）
	}

	h.logger.Info("玩家接入", "player", player, "match_started", shouldStart)
	go h.writePump(c)
	h.sendSnapshot(c, player)
	h.readLoop(c)
}

func (h *Hub) readJoin(ws *websocket.Conn) (match.Player, string, error) {
	_, data, err := ws.ReadMessage()
	if err != nil {
		return "", "", err
	}
	var msg matchpb.ClientMessage
	if err := protojson.Unmarshal(data, &msg); err != nil {
		return "", "", err
	}
	join := msg.GetJoin()
	if join == nil {
		return "", "", fmt.Errorf("首条消息必须是 join")
	}
	p := match.Player(join.GetPlayer())
	if p != match.P1 && p != match.P2 {
		return "", "", fmt.Errorf("player 必须是 p1 或 p2")
	}
	return p, join.GetToken(), nil
}

func (h *Hub) reject(ws *websocket.Conn, msg string) {
	h.logger.Warn("连接被拒", "reason", msg)
	_ = ws.Close()
}

func (h *Hub) readLoop(c *conn) {
	defer func() {
		c.close()
		h.removeConn(c)
		c.ws.Close()
		h.logger.Info("玩家断开", "player", c.player)
	}()
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		var msg matchpb.ClientMessage
		if err := protojson.Unmarshal(data, &msg); err != nil {
			h.logger.Warn("非法消息", "err", err)
			continue
		}
		if sub := msg.GetSubmit(); sub != nil {
			// 引擎可能在终局广播后被置空（下一场对局重建），nil 保护避免空指针。
			h.mu.Lock()
			eng := h.engine
			h.mu.Unlock()
			if eng != nil {
				// 03 号票：透传行动选择与移动目标格（参数校验在引擎侧）。
				moveTo := match.Pos{}
				if mt := sub.GetMoveTo(); mt != nil {
					moveTo = match.Pos{Row: int(mt.GetRow()), Col: int(mt.GetCol())}
				}
				eng.Submit(c.player, sub.GetAction(), sub.GetCode(), moveTo)
			}
		}
	}
}

func (h *Hub) removeConn(c *conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[c.player] == c {
		delete(h.conns, c.player)
	}
	// 席位绑定一并清除：同席位换身份重连不得沿用旧身份结算（fail-closed）。
	delete(h.seats, c.player)
}

func (h *Hub) writePump(c *conn) {
	for b := range c.outbox {
		if err := c.ws.WriteMessage(websocket.TextMessage, b); err != nil {
			return
		}
	}
}

func (h *Hub) sendSnapshot(c *conn, player match.Player) {
	b, err := protojson.Marshal(&matchpb.ServerMessage{Payload: &matchpb.ServerMessage_Snapshot{
		Snapshot: h.engine.Snapshot(player),
	}})
	if err != nil {
		h.logger.Error("快照序列化失败", "err", err)
		return
	}
	c.send(b)
}

// broadcast：引擎事件扇出（在引擎锁内被调用——只做序列化与投递，不回调引擎）。
func (h *Hub) broadcast(env *matchpb.Envelope) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if me := env.GetMatchEnded(); me != nil {
		// 07 号票：引擎终局后、事件广播前完成结算与落库，结果附在同一条
		// MatchEnded 上发出（终局面板即时可见）。任一方匿名/落库失败时
		// elo 字段为空，行为与 01 号票完全一致（降级不断连）。
		h.settleEnded(me)
		// 对局结束即作废引擎：下一场对局由下一条 join 重新创建。
		// 否则 ENDED 相位的引擎被永久复用，后续所有接入永远无法开赛——
		// UI 承诺的"刷新两个窗口可再来一局"将落空（浏览器验收发现的 P0）。
		h.engine = nil
		h.seats = make(map[match.Player]store.User)
	}
	b, err := protojson.Marshal(&matchpb.ServerMessage{Payload: &matchpb.ServerMessage_Event{Event: env}})
	if err != nil {
		h.logger.Error("事件序列化失败", "err", err)
		return
	}
	for _, c := range h.conns {
		c.send(b)
	}
}

// settleEnded：终局结算与落库（调用方持 h.mu；引擎已 ENDED，无后续事件，
// 同步 DB I/O 不会阻塞对局推进）。仅双方均为已认证用户时写入；
// 同一用户分饰双方时跳过（防自刷分）。
func (h *Hub) settleEnded(me *matchpb.MatchEnded) {
	if h.st == nil {
		return
	}
	u1, ok1 := h.seats[match.P1]
	u2, ok2 := h.seats[match.P2]
	if !ok1 || !ok2 {
		return
	}
	if u1.ID == u2.ID {
		h.logger.Info("同用户分饰双方，跳过结算落库（防自刷分）")
		return
	}
	if h.matchID == uuid.Nil {
		// 防御：终局先于开赛记账的路径（仅测试/未来拓扑）也应有唯一 ID，不复用零值。
		h.matchID = uuid.New()
	}
	if h.matchStart.IsZero() {
		// H2：与上同源——零值计时会导致 duration 回绕成垃圾毫秒数入库。
		h.matchStart = time.Now()
	}
	durationMs := int32(time.Since(h.matchStart).Milliseconds())
	if durationMs < 0 {
		durationMs = 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), settleTimeout)
	defer cancel()
	d1, d2, err := h.st.RecordMatch(ctx, store.MatchResult{
		MatchID:    h.matchID,
		Winner:     me.GetWinner(),
		Reason:     me.GetReason(),
		DurationMs: durationMs,
		P1UserID:   u1.ID,
		P2UserID:   u2.ID,
	})
	if err != nil {
		// 降级：落库失败不断连、不重发，终局照常展示（无 ELO 区块）。
		h.logger.Error("终局落库失败（已降级为无 ELO 广播）", "match", h.matchID, "err", err)
		return
	}
	me.Elo = []*matchpb.PlayerElo{
		{Player: string(match.P1), UserId: d1.UserID.String(), EloBefore: d1.Before, EloAfter: d1.After},
		{Player: string(match.P2), UserId: d2.UserID.String(), EloBefore: d2.Before, EloAfter: d2.After},
	}
}
