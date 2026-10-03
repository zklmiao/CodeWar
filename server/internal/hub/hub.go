// Package hub：热座对局的 WebSocket 接入层（ADR-0002 事件流同步）。
// 本票拓扑：同一台机器两个浏览器窗口分别以 ?player=1/2 接入同一场本地对局；
// 在线对战（08 号票）复用同一引擎与事件流，仅连接拓扑不同。
// 注意：proto 消息含 oneof，必须走 protojson + 原始字节收发，
// gorilla 的 ReadJSON/WriteJSON（encoding/json）无法处理 oneof。
package hub

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/match"

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

type Hub struct {
	mu     sync.Mutex
	engine *match.Engine
	judge  match.JudgePort
	conns  map[match.Player]*conn
	logger *slog.Logger
}

func New(judge match.JudgePort, logger *slog.Logger) *Hub {
	return &Hub{
		judge:  judge,
		conns:  make(map[match.Player]*conn),
		logger: logger,
	}
}

// ServeWS：GET /ws 处理器。首条消息必须是 join。
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	player, err := h.readJoin(ws)
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
	if h.engine == nil {
		h.engine = match.NewEngine(match.RealClock{}, match.StubQuestions{}, h.judge, h.broadcast)
	}
	c := &conn{player: player, ws: ws, outbox: make(chan []byte, 64), closed: make(chan struct{})}
	h.conns[player] = c
	eng := h.engine
	started := false
	if len(h.conns) == 2 && eng.Phase() == matchpb.MatchPhase_MATCH_PHASE_WAITING {
		eng.StartMatch()
		go eng.Run() // 计时泵：服务端权威倒计时（ADR-0002）
		started = true
	}
	h.mu.Unlock()

	h.logger.Info("玩家接入", "player", player, "match_started", started)
	go h.writePump(c)
	h.sendSnapshot(c, player)
	h.readLoop(c)
}

func (h *Hub) readJoin(ws *websocket.Conn) (match.Player, error) {
	_, data, err := ws.ReadMessage()
	if err != nil {
		return "", err
	}
	var msg matchpb.ClientMessage
	if err := protojson.Unmarshal(data, &msg); err != nil {
		return "", err
	}
	join := msg.GetJoin()
	if join == nil {
		return "", fmt.Errorf("首条消息必须是 join")
	}
	p := match.Player(join.GetPlayer())
	if p != match.P1 && p != match.P2 {
		return "", fmt.Errorf("player 必须是 p1 或 p2")
	}
	return p, nil
}

func (h *Hub) reject(ws *websocket.Conn, msg string) {
	h.logger.Warn("连接被拒", "reason", msg)
	_ = ws.Close()
}

func (h *Hub) readLoop(c *conn) {
	defer func() {
		c.close()
		h.mu.Lock()
		if h.conns[c.player] == c {
			delete(h.conns, c.player)
		}
		h.mu.Unlock()
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
			h.engine.Submit(c.player, sub.GetCode())
		}
	}
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
	b, err := protojson.Marshal(&matchpb.ServerMessage{Payload: &matchpb.ServerMessage_Event{Event: env}})
	if err != nil {
		h.logger.Error("事件序列化失败", "err", err)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, c := range h.conns {
		c.send(b)
	}
}
