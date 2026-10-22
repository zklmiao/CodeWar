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
	"codewar/server/internal/bot"
	"codewar/server/internal/classes"
	"codewar/server/internal/match"
	"codewar/server/internal/question"
	"codewar/server/internal/question/templates"
	"codewar/server/internal/store"
	"codewar/server/internal/tutorial"

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
	// 05 号票：本场对局配置（首个 join 创建引擎时锁定，快照展示与 ELO 结算
	// 共用；终局随引擎置空一并复位，下一场由下一条 join 重新锁定）。
	cfg matchConfig
	// 09 号票：PvE Bot 控制器（仅 pve 局非空）。Bot 不占用 conns 席位——
	// 对局接入层只有一个人；终局/断线时 Stop 并置空。
	botCtl *bot.Controller
	// botClock：Bot 调度时钟的测试注入口（nil = 真实时钟）。
	botClock bot.Clock
	// 10 号票：教程目标判定器（仅 tutorial 局非空）。与 Bot 同一 observer 通道；
	// tutDone 防止同一关达成后重复落库（DB 写失败会复位以便重试）。
	tutCtl  *tutorial.Checker
	tutDone bool
	// started：当前引擎是否已开赛。废弃大厅回收用它判断，避免持 h.mu 时
	// 调 Engine.Phase() 取引擎锁（broadcast 的锁序是 engine→h.mu，反向会死锁）。
	started bool
}

// accountStore：hub 需要的持久化子集（*store.Store 满足；测试注入桩）。
type accountStore interface {
	Authenticate(ctx context.Context, token string) (store.User, error)
	RecordMatch(ctx context.Context, res store.MatchResult) (store.EloDelta, store.EloDelta, error)
	// 10 号票：教程进度（服务端唯一写入方）。
	GetTutorialProgress(ctx context.Context, userID uuid.UUID) (store.TutorialProgress, error)
	AdvanceTutorial(ctx context.Context, userID uuid.UUID, level, totalLevels int32) (store.TutorialProgress, error)
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

	player, token, classID, skillID, jc, err := h.readJoin(ws)
	if err != nil {
		ws.Close()
		return
	}

	h.mu.Lock()
	// 废弃大厅回收：首个 join 锁定的引擎还没开赛、且当前一个连接都没有时，
	// 说明上一场没有第二个人来（或首个玩家在开赛前断开）。此时保留旧配置锁
	// 会让新 URL 的 mode/topic/cap/strategy 全部被忽略（09 手工验收复现：
	// 残留的 WAITING PvP 局把 PvE 链接吃成了等第二个玩家）。按新 join 重建。
	if h.engine != nil && len(h.conns) == 0 && !h.started {
		if h.botCtl != nil {
			h.botCtl.Stop()
			h.botCtl = nil
		}
		h.tutCtl = nil
		h.tutDone = false
		h.engine.Stop()
		h.engine = nil
		h.seats = make(map[match.Player]store.User)
		h.cfg = matchConfig{}
		h.matchID = uuid.Nil
		h.matchStart = time.Time{}
		h.logger.Info("回收废弃大厅（WAITING 且无连接），按新 join 重新锁定配置")
	}
	// 09 号票：模式归属本场配置（首个 join 锁定）。引擎已存在时以锁定值为准，
	// 本窗口携带的 mode/bot 与其余配置参数一样被忽略（日志见下方 anySet 分支）。
	firstJoin := h.engine == nil
	mc := h.cfg
	if firstJoin {
		mc = parseMatchConfig(jc, h.logger)
	}
	pve := mc.mode == modePvE
	tutorialMode := mc.mode == modeTutorial
	solo := pve || tutorialMode
	if solo {
		// 单人模式人类固定 p1（player 可省略）；p2 是 Bot 席位，不接受人类接入。
		if player == "" {
			player = match.P1
		}
		if player != match.P1 {
			h.mu.Unlock()
			h.reject(ws, "单人模式人类固定为 p1（p2 由机器人占位）")
			return
		}
		if !firstJoin {
			h.mu.Unlock()
			h.reject(ws, "对局进行中，不接受第二名玩家")
			return
		}
	}
	if _, taken := h.conns[player]; taken {
		h.mu.Unlock()
		h.reject(ws, player+" 已有玩家接入（本票不支持重连，08 号票处理）")
		return
	}
	// 06 号票：职业装备组合校验（盲选值 fail-closed；缺省/空 = legacy 锚点）。
	// 10 号票：教程使用固定职业，join 携带的 class/skill 忽略，不做校验。
	if !tutorialMode && (classID != "" || skillID != "") {
		if err := classes.ValidateLoadout(classID, skillID); err != nil {
			h.mu.Unlock()
			h.reject(ws, "职业装备组合非法："+err.Error())
			return
		}
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
	// 10 号票：教程需要登录（进度只落服务端账号）；且只能进入已解锁的关。
	if tutorialMode {
		u, ok := h.seats[player]
		if !ok {
			h.mu.Unlock()
			h.reject(ws, "新手教程需要登录（进度保存在账号上）")
			return
		}
		prog, err := h.tutorialProgress(u.ID)
		if err != nil {
			h.mu.Unlock()
			h.reject(ws, "教程进度读取失败，请稍后重试")
			return
		}
		if !prog.Completed && int32(mc.level) > prog.Level+1 {
			h.mu.Unlock()
			h.reject(ws, fmt.Sprintf("教程第 %d 关尚未解锁（当前进度：已完成第 %d 关）", mc.level, prog.Level))
			return
		}
	}
	// 10 号票：已登录且未完成教程 → 拒绝 PvP 入口（spec 用户故事 6）。
	// 08 号票的在线匹配入口复用本函数位置；匿名热座 URL 直连是实验室旁路
	// （无账号即无进度可查），显式放行——不在此拦截，也不伪造教程状态。
	if mc.mode == "" {
		if u, ok := h.seats[player]; ok {
			if err := h.pvpGate(u.ID); err != nil {
				h.mu.Unlock()
				h.reject(ws, err.Error())
				return
			}
		}
	}
	if firstJoin {
		// 05 号票：首个 join 锁定整场配置——题型过滤喂给 04 工厂（question.
		// NewFactory(Config) 承接面零改动），难度上限档同时决定风险系数
		// （引擎普攻伤害）与出题区间上限（同源同档）。每局新 MatchSeed。
		// 09 号票：PvE 与 PvP 走完全相同的 Config 构造与 seed 派生（验收 4）。
		// 10 号票：教程改走固定题包（同一条引擎构造路径，QuestionPort 换实现）。
		mc.seed = question.NewMatchSeed()
		var questions match.QuestionPort
		setup := match.Setup{RiskTier: mc.tier}
		var level tutorial.Level
		if tutorialMode {
			var ok bool
			level, ok = tutorial.LevelByN(mc.level)
			if !ok {
				h.mu.Unlock()
				h.reject(ws, "未知教程关号")
				return
			}
			bank, err := tutorial.NewBank(mc.level)
			if err != nil {
				h.mu.Unlock()
				h.reject(ws, "教程题包初始化失败："+err.Error())
				return
			}
			questions = bank
			// 教程固定职业组合（不做盲选，数值可复现才好教学）。
			setup = match.Setup{
				P1Class: level.PlayerClass, P1Skill: level.PlayerSkill,
				P2Class: level.BotClass, P2Skill: level.BotSkill,
				RiskTier: mc.tier,
			}
		} else {
			questions = newQuestions(mc)
		}
		h.engine = match.NewEngineWithSetup(match.RealClock{}, questions, h.judge, h.broadcast, setup)
		h.started = false
		if pve || tutorialMode {
			// 09 号票：PvE Bot 职业按 matchSeed 确定性盲选；教程用固定职业。
			botClassID, botSkillID := bot.Loadout(mc.seed)
			if tutorialMode {
				botClassID, botSkillID = level.BotClass, level.BotSkill
			}
			if err := h.engine.BindClass(match.P2, botClassID, botSkillID); err != nil {
				h.engine.Stop()
				h.engine = nil
				h.started = false
				h.mu.Unlock()
				h.reject(ws, "Bot 职业绑定失败："+err.Error())
				return
			}
			botCfg := bot.Config{
				Seed:       mc.seed,
				Difficulty: bot.Difficulty(mc.botDifficulty),
				ClassID:    botClassID,
				SkillID:    botSkillID,
				Player:     match.P2,
				Questions:  questions,
				Engine:     h.engine,
				Clock:      h.botClock,
				Logger:     h.logger,
			}
			if tutorialMode {
				botCfg.Script = &level.Script
			}
			ctl, err := bot.New(botCfg)
			if err != nil {
				h.engine.Stop()
				h.engine = nil
				h.started = false
				h.mu.Unlock()
				h.reject(ws, "Bot 初始化失败："+err.Error())
				return
			}
			h.botCtl = ctl
			if tutorialMode {
				chk, err := tutorial.NewChecker(mc.level, mc.tier)
				if err != nil {
					h.botCtl.Stop()
					h.botCtl = nil
					h.engine.Stop()
					h.engine = nil
					h.started = false
					h.mu.Unlock()
					h.reject(ws, "教程判定器初始化失败："+err.Error())
					return
				}
				h.tutCtl = chk
				h.tutDone = false
			}
		}
		h.cfg = mc
		h.logger.Info("对局配置已锁定（首个 join）",
			"topic", mc.topicDisplay(), "cap", mc.tier, "strategy", mc.strategy,
			"mode", mc.mode, "bot", mc.botDifficulty, "tutorial_level", mc.level)
	} else if jc.anySet() {
		// 05 裁决 1：后续 join 的配置参数忽略（本场已按首个 join 锁定）。
		// 不拒连——第二窗口复制粘贴同一链接是常态路径，参数一致时无感。
		h.logger.Warn("后续 join 的对局配置参数已忽略（本场按首个 join 锁定）",
			"ignored_topic", jc.topic, "ignored_cap", jc.cap, "ignored_strategy", jc.strategy,
			"ignored_mode", jc.mode, "ignored_bot", jc.bot, "ignored_level", jc.level)
	}
	// 06 号票：职业绑定（仅 WAITING 可绑；join 请求已 fail-closed 校验过组合）。
	// 第二名玩家 join 时引擎已存在，同样可绑——双方就位开赛前数值与初始 HP 冻结。
	// 10 号票：教程使用固定职业，忽略 join 的 class/skill（不静默覆盖教学配置）。
	if tutorialMode && (classID != "" || skillID != "") {
		h.logger.Info("教程使用固定职业，join 的 class/skill 已忽略",
			"class", classID, "skill", skillID, "tutorial_level", mc.level)
	}
	if !tutorialMode && classID != "" {
		if err := h.engine.BindClass(player, classID, skillID); err != nil {
			// 纵深防御：组合已在 join 时校验过，此分支理论上不可达。单人首 join
			// 若走到这里必须整体回滚，否则会在没有人类席位的情况下留下 Bot 局。
			if solo && firstJoin {
				if h.botCtl != nil {
					h.botCtl.Stop()
					h.botCtl = nil
				}
				h.tutCtl = nil
				h.tutDone = false
				h.engine.Stop()
				h.engine = nil
				h.seats = make(map[match.Player]store.User)
				h.cfg = matchConfig{}
				h.started = false
			}
			h.mu.Unlock()
			h.reject(ws, "职业绑定失败："+err.Error())
			return
		}
	}
	c := &conn{player: player, ws: ws, outbox: make(chan []byte, 64), closed: make(chan struct{})}
	h.conns[player] = c
	eng := h.engine
	// 开赛必须在释放 h.mu 之后：StartMatch 会同步 emit 事件 → broadcast 再次请求 h.mu，
	// 持锁调用会自死锁（sync.Mutex 不可重入）。
	// 09 号票：单人模式没有第二个人类窗口——Bot 已就位即开赛（人类接入仅一人）。
	shouldStart := eng.Phase() == matchpb.MatchPhase_MATCH_PHASE_WAITING && (solo || len(h.conns) == 2)
	if shouldStart {
		// 07 号票：本场 ID 与开赛时刻（结果落库与时长用）。
		h.started = true
		h.matchID = uuid.New()
		h.matchStart = time.Now()
	}
	h.mu.Unlock()

	if shouldStart {
		eng.StartMatch()
		go eng.Run() // 计时泵：服务端权威倒计时（ADR-0002）
	}

	h.logger.Info("玩家接入", "player", player, "match_started", shouldStart, "pve", pve,
		"tutorial", tutorialMode, "tutorial_level", mc.level)
	go h.writePump(c)
	h.sendSnapshot(c, player)
	// 10 号票：教程状态先于 readLoop 下发（目标清单/关卡元数据/槽位）。
	// 未登录/未知关已在上面拒绝，这里只负责渲染数据。
	if tutorialMode {
		h.sendTutorialState(c)
	}
	h.readLoop(c)
}

// newQuestions：PvE/PvP 共用的出题工厂构造（09 验收 4 的单一构造路径）。
// 模式与 Bot 参数不参与配置——两种模式从同一 question.Config、同一默认值、
// 同一 seed 派生出题；PvE 只是把 p2 的行动提交者换成 Bot。
func newQuestions(mc matchConfig) *question.Factory {
	f := question.NewFactory(question.Config{
		Topics:        mc.topics,
		MaxDifficulty: question.Difficulty(mc.tier),
		Strategy:      mc.strategy,
		MatchSeed:     mc.seed,
	})
	f.RegisterAll(templates.All()...)
	return f
}

func (h *Hub) readJoin(ws *websocket.Conn) (match.Player, string, string, string, joinConfig, error) {
	_, data, err := ws.ReadMessage()
	if err != nil {
		return "", "", "", "", joinConfig{}, err
	}
	var msg matchpb.ClientMessage
	if err := protojson.Unmarshal(data, &msg); err != nil {
		return "", "", "", "", joinConfig{}, err
	}
	join := msg.GetJoin()
	if join == nil {
		return "", "", "", "", joinConfig{}, fmt.Errorf("首条消息必须是 join")
	}
	p := match.Player(join.GetPlayer())
	// 09/10 号票：单人模式人类固定 p1，player 参数可省略（Bot 占 p2）。
	// PvP 契约不变：空 player 仍在 readJoin 阶段拒绝（与 01~07 的拒连路径一致）。
	if p == "" {
		if m := normalize(join.GetMode()); m == modePvE || m == modeTutorial {
			p = match.P1
		}
	}
	if p != match.P1 && p != match.P2 {
		return "", "", "", "", joinConfig{}, fmt.Errorf("player 必须是 p1 或 p2")
	}
	// 05 号票：对局配置参数随 join 注入（页面 URL → 客户端 → join 契约）。
	// 09/10 号票：mode/bot/level 走同一锁定路径。
	jc := joinConfig{
		topic: join.GetTopic(), cap: join.GetCap(), strategy: join.GetStrategy(),
		mode: join.GetMode(), bot: join.GetBot(), level: join.GetLevel(),
	}
	return p, join.GetToken(), join.GetClass(), join.GetSkill(), jc, nil
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
				// 06 号票：提交自带 targeting 时先落引擎（结算读最新值；等效
				// SetTargeting，省一次往返）。
				if t := sub.GetTargeting(); t != nil {
					eng.SetTargeting(c.player, match.Targeting{
						Direction: t.GetDirection(), Distance: int(t.GetDistance()),
					})
				}
				eng.Submit(c.player, sub.GetAction(), sub.GetCode(), moveTo)
			}
		}
		// 06 号票：SetTargeting——解出攻击/技能题后、结算前随时修改目标（免费操作）。
		if st := msg.GetSetTargeting(); st != nil {
			h.mu.Lock()
			eng := h.engine
			h.mu.Unlock()
			if eng != nil && st.GetTargeting() != nil {
				eng.SetTargeting(c.player, match.Targeting{
					Direction: st.GetTargeting().GetDirection(),
					Distance:  int(st.GetTargeting().GetDistance()),
				})
			}
		}
	}
}

func (h *Hub) removeConn(c *conn) {
	h.mu.Lock()
	if h.conns[c.player] == c {
		delete(h.conns, c.player)
	}
	// 席位绑定一并清除：同席位换身份重连不得沿用旧身份结算（fail-closed）。
	delete(h.seats, c.player)

	// 09/10 号票：单人模式单连接断开 = 对局作废（刷新重开）。Bot 与人一起清场，
	// 不保留"看似还在打、实际 Bot 已停"的僵尸局；通用重连归 08 号票。
	var eng *match.Engine
	var ctl *bot.Controller
	if h.cfg.mode == modePvE || h.cfg.mode == modeTutorial {
		eng, ctl = h.engine, h.botCtl
		h.engine, h.botCtl = nil, nil
		h.tutCtl, h.tutDone = nil, false
		h.seats = make(map[match.Player]store.User)
		h.cfg = matchConfig{}
		h.started = false
	}
	h.mu.Unlock()

	// Stop 取引擎锁：必须在释放 h.mu 之后（broadcast 的锁序是 engine→h.mu）。
	if ctl != nil {
		ctl.Stop()
	}
	if eng != nil {
		eng.Stop()
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
	// 05 号票：快照携带本场锁定的配置（对局内只读展示的权威源——展示服务端
	// 锁定值而非本窗口 URL 参数，两窗口参数冲突时不撒谎）。
	// h.mu 只圈配置/引擎指针读取：engine.Snapshot 取引擎锁，持 h.mu 进入会与
	// broadcast（引擎锁内取 h.mu）构成锁序死锁。
	h.mu.Lock()
	cfg, eng := h.cfg, h.engine
	h.mu.Unlock()
	if eng == nil {
		return
	}
	snap := eng.Snapshot(player)
	// 09/10 号票：PvE/教程的 p2 是 Bot——在 hub 层标注（引擎对 Bot 零感知）。
	// 开局后快照才会揭示其职业（引擎的盲选规则），此处只标记席位属性。
	if cfg.mode == modePvE || cfg.mode == modeTutorial {
		for _, pv := range snap.GetPlayers() {
			if pv.GetId() == string(match.P2) {
				isBot := true
				pv.Bot = &isBot
			}
		}
	}
	snap.MatchConfig = &matchpb.MatchConfig{
		Topic:         cfg.topicKey,
		Cap:           uint32(cfg.tier),
		Strategy:      cfg.strategy,
		Mode:          cfg.mode,
		Bot:           cfg.botDifficulty,
		TutorialLevel: uint32(cfg.level),
	}
	b, err := protojson.Marshal(&matchpb.ServerMessage{Payload: &matchpb.ServerMessage_Snapshot{
		Snapshot: snap,
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
	if r := env.GetSubmissionRejected(); r != nil {
		// 诊断用（Debug）：PvE Bot 提交非法目标/冷却中技能时，回合可能等到超时，
		// 这条日志是定位"卡住"原因的最短路径。
		h.logger.Debug("提交被拒", "player", r.GetPlayer(), "reason", r.GetReason())
	}
	if me := env.GetMatchEnded(); me != nil {
		// 07 号票：引擎终局后、事件广播前完成结算与落库，结果附在同一条
		// MatchEnded 上发出（终局面板即时可见）。任一方匿名/落库失败时
		// elo 字段为空，行为与 01 号票完全一致（降级不断连）。
		h.settleEnded(me)
		// 09/10 号票：Bot 随终局停止（不残留下一个对局的定时提交）。
		if h.botCtl != nil {
			h.botCtl.Stop()
			h.botCtl = nil
		}
		h.tutCtl, h.tutDone = nil, false
		// 对局结束即作废引擎：下一场对局由下一条 join 重新创建。
		// 否则 ENDED 相位的引擎被永久复用，后续所有接入永远无法开赛——
		// UI 承诺的"刷新两个窗口可再来一局"将落空（浏览器验收发现的 P0）。
		h.engine = nil
		h.seats = make(map[match.Player]store.User)
		h.cfg = matchConfig{} // 05 号票：本场配置一并复位（下一场重新锁定）
		h.started = false
	} else if h.botCtl != nil {
		// 09 号票：Bot 仅经广播 observer 获知 RoundStarted（不侵入引擎事件路径）。
		// OnEvent 只采样与调度，不同步回调引擎（此处仍在引擎锁内）。
		h.botCtl.OnEvent(env)
	}
	b, err := protojson.Marshal(&matchpb.ServerMessage{Payload: &matchpb.ServerMessage_Event{Event: env}})
	if err != nil {
		h.logger.Error("事件序列化失败", "err", err)
		return
	}
	for _, c := range h.conns {
		c.send(b)
	}
	// 10 号票：教程目标判定（与 Bot 同一 observer 通道；事件先下发，
	// 状态消息后下发）。完成态要等进度真正落库后才发——见 finishTutorial。
	if env.GetMatchEnded() == nil && h.tutCtl != nil {
		st := h.tutCtl.Observe(env)
		if st.Completed {
			if !h.tutDone {
				h.tutDone = true
				user, ok := h.seats[match.P1]
				level := h.cfg.level
				go h.finishTutorial(user, ok, level)
			}
			// 落库完成前：目标全部点亮，但 completed 仍为 false。
			st.Completed = false
		}
		h.sendTutorialStateLocked(st)
	}
}

// sendTutorialState：接入时只发给当前连接（快照之后的第一条教程状态）。
func (h *Hub) sendTutorialState(c *conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.tutCtl == nil {
		return
	}
	b, err := marshalTutorial(h.tutCtl.State())
	if err != nil {
		h.logger.Error("教程状态序列化失败", "err", err)
		return
	}
	c.send(b)
}

// sendTutorialStateLocked：广播路径（调用方持 h.mu）——状态变化扇出到所有连接。
func (h *Hub) sendTutorialStateLocked(st tutorial.State) {
	b, err := marshalTutorial(st)
	if err != nil {
		h.logger.Error("教程状态序列化失败", "err", err)
		return
	}
	for _, c := range h.conns {
		c.send(b)
	}
}

func marshalTutorial(st tutorial.State) ([]byte, error) {
	goals := make([]*matchpb.TutorialGoal, len(st.Goals))
	for i, g := range st.Goals {
		goals[i] = &matchpb.TutorialGoal{Id: g.ID, Label: g.Label, Done: g.Done}
	}
	return protojson.Marshal(&matchpb.ServerMessage{
		Payload: &matchpb.ServerMessage_Tutorial{Tutorial: &matchpb.TutorialState{
			Level:        uint32(st.Level),
			Title:        st.Title,
			Mechanic:     st.Mechanic,
			Goals:        goals,
			Slots:        st.Slots,
			Completed:    st.Completed,
			NextLevel:    uint32(st.NextLevel),
			AllCompleted: st.AllCompleted,
			Cap:          uint32(st.Cap),
			CapCoeff:     st.CapCoeff,
		}},
	})
}

// finishTutorial：目标达成后的唯一写入方——先落库，成功后才下发完成态并
// 作废本场（客户端没有任何 advance 入口，防自报造假）。
// 调用方已在广播路径把 tutDone 置位；落库失败会复位以便后续事件重试。
func (h *Hub) finishTutorial(user store.User, hasUser bool, level int) {
	if h.st == nil || !hasUser || user.ID == uuid.Nil {
		h.logger.Error("教程达成但缺少登录席位/存储，无法落库", "level", level)
		h.mu.Lock()
		if h.tutCtl != nil {
			h.tutDone = false
		}
		h.mu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), settleTimeout)
	defer cancel()
	if _, err := h.st.AdvanceTutorial(ctx, user.ID, int32(level), int32(tutorial.MaxLevel)); err != nil {
		h.logger.Error("教程进度落库失败（不标记完成，等待重试）",
			"user", user.ID, "level", level, "err", err)
		h.mu.Lock()
		if h.tutCtl != nil {
			h.tutDone = false
		}
		h.mu.Unlock()
		return
	}

	// 落库成功：下发完成态（触发客户端完成页），并像断线一样清场——
	// 教程目标已达成，不再产生新的对局事件，避免终局/超时覆盖完成页。
	h.mu.Lock()
	if h.tutCtl != nil {
		h.sendTutorialStateLocked(h.tutCtl.State())
	}
	eng, ctl := h.engine, h.botCtl
	h.engine, h.botCtl, h.tutCtl = nil, nil, nil
	h.tutDone = false
	h.seats = make(map[match.Player]store.User)
	h.cfg = matchConfig{}
	h.started = false
	h.mu.Unlock()

	// Stop 取引擎锁：必须在释放 h.mu 之后（broadcast 的锁序是 engine→h.mu）。
	if ctl != nil {
		ctl.Stop()
	}
	if eng != nil {
		eng.Stop()
	}
	h.logger.Info("教程关卡完成并已落库", "user", user.ID, "level", level)
}

// tutorialProgress：带超时的进度读取（DB 挂起不得永久卡住持锁的 join 路径）。
func (h *Hub) tutorialProgress(userID uuid.UUID) (store.TutorialProgress, error) {
	ctx, cancel := context.WithTimeout(context.Background(), settleTimeout)
	defer cancel()
	return h.st.GetTutorialProgress(ctx, userID)
}

// pvpGate：PvP 入口门禁（spec 用户故事 6：完成教程才解锁 1v1 匹配）。
// 08 号票的在线匹配入口复用同一判定位置；匿名热座不经过此处
// （无账号即无进度可查——实验室旁路，调用点已显式注释）。
func (h *Hub) pvpGate(userID uuid.UUID) error {
	if h.st == nil {
		return nil
	}
	prog, err := h.tutorialProgress(userID)
	if err != nil {
		return fmt.Errorf("教程进度读取失败，请稍后重试")
	}
	if !prog.Completed {
		return fmt.Errorf("请先完成新手教程（入口页「新手教程」）")
	}
	return nil
}

// settleEnded：终局结算与落库（调用方持 h.mu；引擎已 ENDED，无后续事件，
// 同步 DB I/O 不会阻塞对局推进）。仅双方均为已认证用户时写入；
// 同一用户分饰双方时跳过（防自刷分）。
func (h *Hub) settleEnded(me *matchpb.MatchEnded) {
	if h.st == nil {
		return
	}
	// 09/10 号票：PvE/教程不落战绩/ELO（Bot 无账号——"双方认证才落库"规则的
	// 显式化；显式守卫同时钉住未来拓扑演进时不误把 Bot 席位算进结算）。
	if h.cfg.mode == modePvE || h.cfg.mode == modeTutorial {
		h.logger.Info("单人模式对局不落战绩/ELO（Bot 无账号）", "mode", h.cfg.mode)
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
	// 05 号票：结算奖励乘难度风险系数（首 join 锁定的 cap 档；h.cfg.tier=0
	// 的 paranoid 路径由 RiskCoeff(0)=1.0 兜住，07 号票匿名/缺省行为不变）。
	d1, d2, err := h.st.RecordMatch(ctx, store.MatchResult{
		MatchID:    h.matchID,
		Winner:     me.GetWinner(),
		Reason:     me.GetReason(),
		DurationMs: durationMs,
		P1UserID:   u1.ID,
		P2UserID:   u2.ID,
		RiskCoeff:  match.RiskCoeff(h.cfg.tier),
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
