# 08 号票开工提示词（草稿 v0.1）

> 状态：**草稿**——未经 review & check，不得直接投喂实现代理。review 结论与正式稿见同目录 `ticket-08-kickoff-review.md` / `ticket-08-kickoff-prompt.md`。
> 依据：票文 `issues/08-online-ranked-1v1.md`、ADR-0002/0003、31 号票连接卫生、验收报告 `acceptance-report-2026-10-27.md`（F-3/O-1）、仓库实态核对（2026-10-28）。

---

## 提示词正文（草稿）

你是 CodeWar 仓库（E:\Agent\Proj\CodeWar）的实现工程师，在 worktree `.worktrees/08-online-ranked-1v1`（分支 `ticket/08-online-ranked-1v1`）实现 08 号票：在线 1v1 排位对战。

### 背景

- 先读：票文、`docs/specs/001-codewar-mvp.md`（实现决策·平台与网络、测试决策）、`docs/adr/0002`（事件流同步模型——本票的协议母版）、`docs/adr/0003`（判题沙箱）、`docs/adr/0005/0006`（Bot 边界/pvpGate）、31 号票连接卫生记录、`.scratch/codewar-mvp/acceptance-report-2026-10-27.md` 第 8 节（F-3/O-1）。
- 现状锚点：`proto/codewar/match/v1/match.proto` 已有 ServerMessage/ClientMessage oneof、Snapshot、JoinRequest（含 token 字段）；`scripts/gen-proto.sh` 三端生成；hub 为单引擎单对局槽（`server/internal/hub/hub.go`，conns/seats map，首个 join 锁配置）；`Hub.pvpGate` 教程门禁在 join 校验处（ADR-0006）；客户端重连循环在 `client/src/views/MatchView.vue:132-148`（`setTimeout(connect, 500)`，无定时器清理、无退避——票文写 App.vue 已漂移，以实际代码为准）。

### 做什么

1. **proto schema 定稿并双端生成**：新增匹配消息（排队/取消/匹配成功/超时）与重连请求消息；`scripts/gen-proto.sh` 三端重生成（Go/ts-proto），生成产物入库。
2. **段位就近匹配**：单进程等待队列；按 ELO 就近配对（带宽 ±100，每 10s 放宽 50）；排队可取消；服务重启清空队列（注释记录）。
3. **排位对局会话**：登录用户经 token 鉴权 join 进入匹配；配对成功双方进入同一引擎（pvpGate 复用）；对局结束走既有 settle 全链（双方认证落库 + ELO，平局 +0）。
4. **断线重连与判负**：重连=快照 + 尾随事件（seq 单调，断号触发重拉）；断线超过 60s 判负（reason 记 disconnect-forfeit），走 MatchEnded 恰好一次；PvE/教程局维持现断线作废语义。
5. **客户端重连退避防抖**：修复 MatchView.vue 重连循环——单飞（同一时刻至多一个在途连接）+ 指数退避（建议 500ms 起、×2、上限 10s、抖动），被拒原因照旧上抛。
6. **部署加固（ADR-0003 递延）**：judge 容器改非 root 用户运行，compose 更新，判题回归全绿。

### 测试

- matchmaker 单元/集成测试：就近配对、放宽、取消、并发排队（-race）。
- 重连事件流断言：快照+尾随事件、seq 连续性、断线判负路径。
- client vitest：退避单飞与指数序列（fake timers）。
- 浏览器 E2E 自动化：两窗口匹配→整局→ELO 断言（自动化验证）。
- 全量回归：server `go vet` + `go test -race ./...` + DSN 套件；client vitest + build；pytest judge。

### 交付

- 提交到票分支（conventional commits），逐条汇报票面验收清单；完成后出一份 code-review 报告（Standards/Spec 双轴）。
