# 31: 投降认输 + hub 连接卫生（08 前置）

**What to build:** 两件事，均为 08 号票（在线 1v1）开工前的协议与鲁棒性铺垫：

**A. 投降认输**：当前对局无任何提前结束路径，劣势方只能等 10 回合打满或超时。加 resign 链路：proto `ClientMessage` 扩展 resign 消息（命名与 oneof 归属沿用 match.proto 现有风格）→ hub 接线 → 引擎新入口 `Resign(player)`（活跃对局生效：winner=对方、reason=`resign`、MatchEnded 恰好一次）。错误路径：非活跃对局 / 已终局 / 教程局拒绝；重复投降幂等（第二次无新 MatchEnded）。结算语义：PvP 走既有 settle 路径（self-play guard 沿用），`matches.end_reason` 新值 `'resign'`（列是 TEXT+注释、无 CHECK 约束，**无需 schema migration**，同步注释并先核实写入侧无枚举校验）；PvE resign → bot 胜且不落 matches/elo_history。

**B. hub 连接卫生**：`readLoop`（hub.go:459）/`writePump`（hub.go:545）无读写 deadline、无 ping/pong 保活、无并发连接上限——死链（断网/休眠）永久占位。补：读超时 + 服务端 ping ticker + pong 重置 deadline、写超时、连接数上限（超限走既有 close-frame 拒绝路径 hub.go:450 一带）；**上限按活跃连接计、关闭即时释放名额**（否则断线重连被自己锁死）。ping/pong 是 WebSocket 控制帧，零 proto 变化。

**Blocked by:** None（与 08 无硬依赖；作为其前置建议先行，proto 定稿前把 resign 一起带进去）

**Status:** resolved

**现状证据（2026-10-27，HEAD `161d4f9`）：**

- proto/codewar/match/v1/match.proto:245 `message ClientMessage`（oneof），:294 SubmitRequest 等——resign 新增消息照此风格。
- server/internal/match/engine.go:651 `endMatch(winner, reason)`——reason 现值 ko / self-ko / rounds-cap。
- db/migrations/000001_init.up.sql:40 `end_reason TEXT NOT NULL, -- ko | self-ko | rounds-cap`。
- hub.go:459 readLoop / :545 writePump：无 SetReadDeadline / SetPongHandler / 写超时 / maxConn（grep 零命中）。
- hub.go:450 一带：既有「连接被拒」close-frame 拒绝路径可复用。
- scripts/gen-proto.sh：三端重生成入口。
- client：ws.ts 发送面 + match-store 终局处理；RoundStatusBar/MatchView 为认输按钮落位候选。

- [x] proto：ClientMessage 扩展 resign 消息，`scripts/gen-proto.sh` 三端重生成通过（Go/ts 编译绿，judge 端如受影响一并绿）
- [x] engine：`Resign(player)` 事件流断言——终局 winner/reason 正确、MatchEnded 恰好一次；重复投降、已终局后、教程局拒绝路径全覆盖（`go test -race`）
- [x] hub：resign 接线（真人 + PvE 皆可达引擎）；PvE resign → bot 胜且零 DB 落库；PvP resign → 既有 settle 全链（matches 一行 end_reason='resign' + elo_history 两行 + self-play guard 沿用）
- [x] hub 连接卫生：读超时 + ping/pong 保活 + 写超时 + 并发上限，参数常量化；死连接（不回 pong）被超时回收、超限拒绝、关闭释放后再入——hub 单测全覆盖且 `-race` 绿
- [x] client：认输按钮（confirm 防误触、终局后隐藏、教程局不渲染）+ ws.ts 发送 + store 终局处理与既有路径一致（vitest）
- [x] 行为零变化面：除 resign 新路径外，提交/结算/重连/教程既有测试全绿——`go test -race ./...` + `CODEWAR_TEST_DSN` 并行 DB 包 + `pnpm -C client test` + `pnpm -C client build`
- [x] 对齐记录：resign 属 ADR-0002 事件流框架内的应用层消息，票内记录消息命名与拒绝语义即可，无新 ADR

> 来源：optimization-review-2026-10-27.md「推荐开票组合」第 2 项。

## 对齐记录（31 号票实现裁决，无新 ADR）

- **消息命名**：`ClientMessage` oneof 新增 `ResignRequest resign = 4`（沿 JoinRequest/SubmitRequest 命名风格；请求体无字段——玩家身份由 join 绑定的连接席位权威决定，请求体不可指定对局或玩家，防代投/错席）。ADR-0002 事件流框架内的应用层消息，`MatchEnded.reason` 增值 `resign`，无 schema migration（end_reason 为 TEXT+注释、无 CHECK；已核实 `store.RecordMatch` 无枚举校验、直通入库；`db/migrations/000001_init.up.sql` 行内注释与 store 侧 `MatchResult.Reason` 注释同步）。
- **拒绝语义与归属**：`Engine.Resign(player) error`——仅 ROUND_ACTIVE 生效（winner=对方 / reason=resign / MatchEnded 恰好一次，走与超时/自损同一条 `endMatch` 收束路径，hub 终局广播零改动复用）；拒绝（返回错误、零事件）：禁投对局 / 非活跃（WAITING）/ 已终局（含上一次投降——重复投降幂等即由此保证）。**教程局拒绝的实现取"禁投标记"而非"教程标记"**：引擎内不得出现 tutorial 字样（10 号票 `TestMatchPackageHasNoTutorialReferences` 零侵入边界），故 Setup 增加通用 `ResignLocked bool`，hub 在教程局设置——引擎不知道禁投动机，教学局拒绝是接入层的裁决；同理引擎注释不出现 bot 字样（`TestMatchPackageHasNoBotReferences`）。
- **hub 连接卫生参数**（hub.go 常量块，注释含依据）：pongTimeout=30s（读窗口）/ pingPeriod=27s（=90%，gorilla chat 惯例）/ writeTimeout=10s / joinTimeout=10s（首条 join 读取窗）/ maxConns=8（当前拓扑单场 ≤2 连接，08 号票在线化前的防滥用兜底，08 重估）。附带修复：`conn.send` 满缓冲从阻塞改丢弃——broadcast 在引擎锁内扇出，慢消费者/死链缓冲填满会卡死全场事件流（丢弃后断号，客户端按 ADR-0002 重连自愈）。
- **测试时长说明**：死链回收/保活（`TestDeadConnRecycledAndPongKeepsAlive`）与握手超时（`TestJoinHandshakeTimeout`）走真实墙钟与真实 socket（SetReadDeadline 无法注入假钟），t.Parallel 重叠执行，hub 包 -race 总时长 ≈35s。

> **关票记录（2026-10-27 合并后验收）**：ticket/31 合入 main。验证：`go vet ./...` + `go test -race ./...` 全包绿（hub 36s）；`CODEWAR_TEST_DSN` 落库套件全包绿（hub 185s，含 resign 结算与 PvE 零落库断言）；分支 `pnpm -C client test` 55/55 → 合并后 main 64/64；`pnpm -C client build` 通过；`python -m pytest judge/tests -q` 44 passed。新测试：engine_resign_test（终局/幂等/WAITING/ResignLocked 零事件）、hub_resign_test、hub_conn_hygiene_test、resign.spec（按钮三态/confirm/发送）。
