# 08 号票开工提示词（正式稿 v1.0）

> 状态：**正式**——已按 `ticket-08-kickoff-review.md` 的 R-1…R-12 修订，可直接投喂实现代理。
> 使用方式：将「提示词正文」整段发给实现代理；三项「开工呈报门」数值由人批复后放行实施。

---

## 提示词正文

你是 CodeWar 仓库（E:\Agent\Proj\CodeWar）的实现工程师，实现 08 号票：**在线 1v1 排位对战**。在 `.worktrees/08-online-ranked-1v1`（分支 `ticket/08-online-ranked-1v1`）工作；dev server 端口 `PORT=3008`。

### 0. 开工前必读（按序）

1. 票文 `.scratch/codewar-mvp/issues/08-online-ranked-1v1.md`（验收清单是唯一完成标准，6 条逐条对应本提示词阶段）
2. `docs/specs/001-codewar-mvp.md`：实现决策「平台与网络」、测试决策（三 Seam、事件流断言、错误路径优先）
3. `docs/adr/0002`（事件流同步——本票协议母版）、`docs/adr/0003`（已知递延：judge 非 root 加固归本票）、`docs/adr/0005/0006`（Bot 边界、pvpGate 匹配入口复用点）
4. 31 号票连接卫生记录（pongTimeout/pingPeriod/writeTimeout/joinTimeout/maxConns 语义，`server/internal/hub/hub.go` 常量块）
5. `.scratch/codewar-mvp/acceptance-report-2026-10-27.md` 第 6-8 节（走查证据、F-3/O-1）
6. `docs/agents/issue-tracker.md` 与 AGENTS.md 工作流

### 1. 开工前已核对基线（不必重查，直接采用）

- **已交付产物无漏做**：01-24、26-31 经 2026-10-27 独立验收（37/37 故事矩阵）；验收发现 F-1（ADR-0008 inject_target 记录失实）、F-2（EndOverlay 认输文案）、F-4（风格页部署缺失）均为**独立问题，本票禁止顺手修**。
- **票文漂移注记**：票文清单第 6 条写的「App.vue connect()」已漂移——重连循环实际在 `client/src/views/MatchView.vue:132-148`（`setTimeout(connect, 500)`，无定时器清理、无退避）。以实际代码为准；票文修改留待勾票时由人确认。
- **存量测试审计结论**：hub 9 测试文件（4041 行）/client 12 spec（64 用例）/match/elo/store——-race + DSN + 真容器全绿，数值断言抽查为具体值断言，无 mock 互证，**无既有虚假 pass 或过度测试需要修正**。

### 2. 开工呈报门（未获人批复不得实施对应阶段）

以下三项 spec 未定值，属裁决面。开工第一步：呈报建议值与理由，获批后作为**可调常量**实现（各集中于唯一定义处）：

1. **断线判负等待时长**（建议：45-90s 区间取值，须 > pongTimeout 30s；常量 + 注释含依据）。
2. **匹配带宽初值与放宽节奏**（建议：初始 ±200，每 5s 线性放宽一档；只定语义「就近代价最小配对 + 随等待单调放宽」，不定复杂算法）。
3. **在线对局与教程/PvE/热座的互斥规则**（建议：登录用户同一时刻至多在一个在线会话/队列；排队中开教程/PvE 先自动取消排队）。

### 3. 范围

**做（6 项，对应票面 6 条验收）**：

1. **proto schema 定稿并双端生成**（P0）：扩展 `proto/codewar/match/v1/match.proto`——匹配消息（入队/取消/匹配成功/排队状态）与重连协商消息（重连请求带 token 与最后 seq）；沿用现有 oneof 风格与注释规范（每条消息注明引入票号与语义）；`scripts/gen-proto.sh` 三端重生成，生成产物入库；为事件/对局落库（33 号票：回放+错题本）留**注释级扩展点，不实现**。
2. **段位就近匹配后端**（P0）：hub 侧单进程等待队列（本进程内，不引入 Redis/独立匹配服务；服务重启清空队列并在注释如实记录）；登录用户 token 鉴权入队；就近代价最小配对 + 随等待单调放宽（呈报值）；取消/登出/断开即出队；配对成功→双方进同一 `match.Engine`（复用 `Hub.pvpGate` 门禁与首个 join 锁配置语义，重估 `maxConns=8` 并在注释记录新值依据）。
3. **重连与判负**（P1）：重连=服务端推送快照 + 尾随事件（沿用 seq 单调 + 断号自愈语义，`TestSnapshotMidRound` 为行为锚点）；断线超判负时长（呈报值）→ 判负结算（reason 记录、MatchEnded 恰好一次、走既有 settle 全链：双方认证落库 + ELO，平局 +0）；PvE/教程/热座单连接断线作废语义**零变化**。
4. **客户端**（P2）：重连退避防抖——单飞（同一时刻至多一个在途连接，清理旧定时器）+ 指数退避（500ms 起步 ×2、上限 10s、带抖动），被拒原因照旧上抛；匹配 UI（排队中/取消/匹配成功进入对局）；重连期间 UI 状态以服务端快照为权威，双源不漂移（沿 20 号票口径）。
5. **部署加固**（P3，ADR-0003 递延，独立 phase 不阻塞主线）：judge 容器非 root 运行，`deploy/docker-compose.yml` 与 `Dockerfile.judge` 相应更新，ADR-0003 同步修订；判题回归全绿后才算完成。
6. **服务端权威不回归**（贯穿）：客户端无法自行改变 HP/位置，提交仅经 SubmissionPort——既有引擎/Hub 契约零放松，新增消息全部走 hub 校验后进入引擎既有入口。

**不做（禁令，防过度设计）**：排行榜页、观战/回放/错题本（33 号票）、匹配池持久化与服务恢复、独立匹配服务/消息中间件/多实例拓扑、rating 衰减等复杂匹配算法、反作弊增强（11 号票）、除上述外任何既有测试修改。遇 spec/票据/ADR 冲突：**记录并呈报人，不当场裁决**。

### 4. 阶段与完成标准

| 阶段 | 内容 | 完成标准 |
|---|---|---|
| P0 | 呈报门 + proto 定稿 + 匹配后端 | 呈报获批；proto 三端生成产物入库且 `go build ./... && go vet ./...` 干净；matchmaker 事件流断言全绿（含 -race） |
| P1 | 重连 + 判负 + 在线会话 | 重连/判负事件流断言绿（可注入时钟，不用真实墙钟等 60s）；既有 PvE/教程/热座语义回归零变化 |
| P2 | 客户端退避 + 匹配 UI | vitest 退避单飞/指数序列（fake timers）红→绿；`pnpm -C client test` + `build` 绿 |
| P3 | judge 非 root | compose 起 judge 非 root；`python -m pytest judge/tests -q` 44 passed + templates 冒烟绿；ADR-0003 修订 |
| P4 | 全量回归 + 交付 | 第 6 节回归清单全绿；票面 6 条逐条汇报；code-review 报告 |

### 5. 测试硬约束（防虚假 pass 与过度测试）

1. 只测外部行为：引擎/Hub 断言走**事件流 seam**（spec 测试决策）；client 断言走 data-testid/store 状态机，不做布局像素断言（jsdom 无布局，沿 22-24 号票口径）。
2. 断言**具体值**（伤害数值、ELO 变化、seq 序列、退避时延序列），禁止"不报错即过"、禁止 mock 断言 mock 的配置式互证。
3. 错误路径优先：配对取消/断线判负/重连竞态/被拒重连必须有事件流断言；每条行为修复先写**红测试**再修（提交信息或票尾记录附红→绿证据）。
4. 判负等待、退避时延一律可注入时钟/fake timers；**禁止新增分钟级真实墙钟测试**（既有 hub ~36s 真实时钟用例为既有基线，不效仿不扩）。
5. 测试场景边界：匹配语义/重连连续性/判负路径/ELO 落库/退避序列必测；网络抖动重放、恶意流量、多实例、持久化恢复**不测**（超纲）。
6. 发现既有测试疑似虚假 pass/过度测试：记录现象与锚点 → **呈报人批准** → 修正提交必须附全量回归证据；未批准不得动任何既有测试（沿 31 号票"断言、期望值、用例集不变"惯例）。
7. 「两台设备两个浏览器完成排位赛」为**人工验收项**：自动化不替代；随票交付一份人工操作清单（含预期 ELO 变化核对点）供人执行。

### 6. 回归基线（P4 必须全绿，作为"修正不影响原有功能"的证据）

- `cd server && go build ./... && go vet ./... && go test -race ./... -count=1`
- `CODEWAR_TEST_DSN='postgres://codewar:codewar-dev@localhost:25432/codewar_test?sslmode=disable' go test ./... -count=1`（store/accounts/hub 真跑不得 skip；**不得与其他 go test 进程并行**，会互相污染同名测试库）
- `python -m pytest judge/tests -q`（44 passed 基线，P3 后重跑验证非 root 无回归）
- `pnpm -C client test`（64 用例基线）与 `pnpm -C client build`
- 专项回归：hub 既有测试族零修改全绿（`hub_pve_test`/`hub_tutorial_test`/`hub_conn_hygiene_test`/`hub_resign_test`/`hub_elo_test`）；`TestSnapshotMidRound`、`TestPveDisconnectAbortsAndRefreshRestarts` 语义不变

### 7. 交付协议

1. 分支提交用 conventional commits（英文），代码注释与文档中文；每阶段一议，不攒巨提交。
2. 最终汇报：票面 6 条验收**逐条**给出结论与证据（测试名/命令输出/截图）；不确定或部分完成的如实标注，禁止无证据 PASS。
3. 附 **code-review 报告**：双轴（Standards 轴：仓库惯例/token/锁序/测试边界；Spec 轴：票面 6 条 + ADR-0002/0003 语义逐条对照），给出硬违规/建议项与自查结论。
4. **不修改票据 Status、不勾验收框**（勾票归人）；票文漂移（App.vue→MatchView.vue）留待勾票时由人更新。
5. 完成标准之外的一切新想法记入票尾「后续」或另开票建议，不在本票实施。

---

*正式稿完。草稿（v0.1）与审查记录（R-1…R-12）见同目录。*
