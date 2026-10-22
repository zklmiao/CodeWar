# 16: PvE 回合偶发停滞（TestPveDoesNotRecordMatchOrElo 超时）

**What to build:** 定位并修复 PvE 对局偶发卡在某一回合的问题：真人已提交、Bot 未在回合内提交，导致回合等满 75s 超时；同时把 hub 的 PvE e2e 测试从 wall-clock 赌博改成可推进时钟的事件流断言。先诊断、后修复，禁止用调大超时掩盖。

**Blocked by:** 09（已 resolved；本票是其后继缺陷修复）

**Status:** needs-triage

- [ ] 确定失败根因（产品缺陷 / 测试实时竞态），票内记录证据：最后回合、缺失行动方、事件序列
- [ ] 若是产品缺陷：修复 Bot 调度，保证每收到一个 `RoundStarted` 恰好产生一次 `Submit`（不得在 hub/引擎加特判）
- [ ] 若是测试问题：Hub 测试注入可推进的引擎时钟，改为事件流断言；wall-clock smoke 最多保留一条
- [ ] `go test -race -count=200 ./internal/hub/... ./internal/bot/...` 绿（无 DB 时注明 skip）
- [ ] 引擎 75s 回合超时自愈路径有测试覆盖（manual clock）
- [ ] 引擎包仍无 bot/tutorial 特判（既有结构测试继续绿）

## 已知证据（10 号票验收期间记录）

- `TestPveDoesNotRecordMatchOrElo` 在 10 号票分支 50 次中复现 1 次 30s 超时；在未改动的 main worktree（`E:\Agent\Proj\CodeWar`，提交 `d1e6548`）上同样复现——**非 10 号票引入**。
- 正常 PvE 局（Bot 用 2ms 快钟、人类每回合 PanelReady 即提交）应在 1s 内结束；挂起只能是某回合未结算。
- 超时分支看到的 `engine == nil` 不可作为"已正常终局"的证据：probe 客户端也有 30s 读超时，它先触发会关连接、`removeConn` 清空引擎，与测试计时器形成竞态。
- 现有 Hub 只注入了 `botClock`，引擎仍是 `match.RealClock{}`，所以 75s 回合超时无法在测试中加速——这与 spec「注入加速 ClockPort」的测试决策不一致。

## 诊断顺序（先分析、后计划、再修改）

1. 测试侧按回合记录 `RoundStarted / PanelReady / SubmissionJudged / ActionResolved`，失败时打印缺失行动方与回合号。
2. 给 Hub 增加测试用 `engineClock match.Clock` 注入口（生产仍 `RealClock`）。
3. `go test -race -count=500` 诊断版本，取得确定性复现。
4. 归类：Bot 漏提交（产品 bug，修 bot 调度）或测试实时竞态（改 Hub 测试）。
5. 修复后按"失败协议"附回归证据：复现失败 → 根因 → 修复点 → 为什么不是改另一边。
