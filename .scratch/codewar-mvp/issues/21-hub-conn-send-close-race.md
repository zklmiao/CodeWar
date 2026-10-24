# 21: hub conn send/close 竞态（断连与广播并发时 send-on-closed-channel panic）

**What to build:** 修复 hub 连接生命周期的预存数据竞争：`conn.close()` 曾同时关闭信号通道 `closed` 与数据通道 `outbox`，而 `conn.send()` 的 select 发送分支与 `closechan` 之间无 happens-before——`-race` 报 DATA RACE，交错不利时 `send on closed channel` panic 直接崩掉测试进程。触发结构：readLoop 断连收尾的 close（hub 锁外）× Bot 定时器 goroutine 的广播 send（hub 锁内）。修复采用"数据通道永不关闭"模式：close 只关信号；writePump 改 select 退出（退出前排干缓冲，对齐旧 close+range 收尾语义）；send 零改动。发现于 20 号票合并门控（全包 -race 首次照出）；代码出自初始提交 b7962b4，与 19/20 号票 diff 零因果（main 隔离压测已确定性复现）。20 号票合并在本票落 main 后恢复。

**根因证据（2026-10-24，20 worktree 全包首跑红 + 3×5 轮中 1 轮红；main 隔离压测必红）：**

```txt
WARNING: DATA RACE
Read : runtime.chansend ← conn.send ← sendTutorialStateLocked ← broadcast
       ← Engine.emit ← Engine.Submit ← bot.Controller.act（定时器 goroutine）
Write: runtime.closechan ← conn.close ← readLoop defer（断连收尾，hub 锁外）
panic: send on closed channel → 整个测试进程崩溃
```

**Blocked by:** None（可立即开工）

**Status:** ready-for-agent

- [ ] 红→绿：并发 send/close 压测（500 轮 × 4 发送者对 1 close）修复前 `-race` 必红（DATA RACE / panic），修复后绿
- [ ] close() 只关 `closed` 信号（保留 once 幂等），outbox 永不关闭；send() 零改动
- [ ] writePump 改 select(outbox, closed) 退出，退出前排干已缓冲消息（对齐旧 close+range 收尾语义）
- [ ] `go test -race -count=1 ./...`（server 全包）绿、零 DATA RACE
- [ ] 断连语义不退：hub 既有断连/教程测试全绿（TestPveDisconnectAbortsAndRefreshRestarts、TestJoinBindsUserAndCleansUpOnDisconnect、TestTutorialL2HitDoesNotComplete 等）
- [ ] 高强度回归不退：`go test -race ./internal/hub/... -run 'TestPve' -count=200`
