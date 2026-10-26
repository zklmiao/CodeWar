# 28: 引擎在判题期间持锁——后到玩家的排队时间污染 solveMs（镜像竞速不公平）

**What to build:** `server/internal/match/engine.go` 的 `Submit` 全程持有 `e.mu`，包括同步调用 `e.judge.Judge`（Docker 判题，0.2–3 秒，RPC 上限 16 秒）。`now := e.clock.Now()` 在拿到锁之后才取，所以当对手（或 PvE Bot，见 `server/internal/bot/controller.go` 中的 `Submit` 调用）的判题正在进行时，后到玩家的排队时间会被算进 `solveMs`：时间系数变低、先手顺序被打乱，违反 spec 用户故事 12、15、16。判题期间 `SetTargeting`、`Snapshot`（重连）、计时泵 `HandleRoundTimeout` 也全部被阻塞。

修复要求：`Submit` 拆三段——
1. **锁外**：`received := e.clock.Now()`（进入 Submit 的第一件事）。
2. **持锁预检**：保留现有全部校验，顺序与语义不变（phase、已行动、非法 action、技能未装备/冷却、移动参数、锁定期——锁定判断改用 `received`）。新增 `inFlight map[Player]bool`：同一玩家已有判题在途时，再次提交静默丢弃（与"已行动再交"一致）。记下 `round := e.round` 和本次要判的题，把 `inFlight[player]` 置为 true，然后**释放锁**。
3. **锁外判题**，然后**重新加锁落地**（用 defer 清除 `inFlight[player]`）：`phase != ROUND_ACTIVE` 或 `e.round != round`（判题期间回合已结算或对局已结束）→ 丢弃结果，不发事件，不罚；否则按现有逻辑处理，`solveMs = received - e.roundStart`。

约束：锁序约定（先引擎锁、后 `h.mu`；broadcast 在引擎锁内调用）不变；不改 proto、不改客户端；`Submit` 对调用方仍是同步返回（调用它的 goroutine 阻塞到判题结束），既有测试应当无需修改。

测试（`server/internal/match/`，可阻塞的 fake judge + 现有假钟；断言风格遵守 spec「测试决策」：只断言事件流等外部行为）：
1. 公平性回归（在旧代码上必须失败）：P1 提交，fake judge 阻塞；把假钟拨到 +10s；P2 在另一个 goroutine 提交。断言 P2 的 judge 在 P1 判题仍阻塞时就被调用（加 2s 超时，防止测试挂死）。再拨到 +20s，放行两个判题，断言 P2 的 `SubmissionJudged.solve_ms == 10000`。
2. 判题阻塞期间，`SetTargeting` 和 `Snapshot` 能在 100ms 内返回。
3. 判题阻塞期间回合超时结算 → 放行后该结果被丢弃，事件流中没有它的 `SubmissionJudged`。
4. 同一玩家并发两次提交 → fake judge 只被调用一次。

**Blocked by:** None

**Status:** ready-for-agent

- [ ] 四条新测试落地；在修复前代码上的失败/通过情况实测并如实记录（测试 1 必须失败）
- [ ] `Submit` 三段拆分落地；既有测试零修改
- [ ] `go vet ./...` 绿；`go test ./... -count=1` 绿；`go test -race ./internal/match/... ./internal/hub/... ./internal/bot/... -count=1` 绿（Windows 缺 cgo/gcc 时如实注明，不跳过其余检查）
- [ ] 锁序约定与同步返回语义不变（代码检查 + 既有事件流断言佐证）
