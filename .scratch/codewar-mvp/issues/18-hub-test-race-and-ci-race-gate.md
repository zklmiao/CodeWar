# 18: hub 测试日志竞争修复 + CI race 门控

**What to build:** 两件事，均为测试基建（产品代码零改动）：
1. **修 hub 测试的数据竞争**：`hub_config_test.go` 的 `captureLogger()` 用无保护的 `*bytes.Buffer` 作 slog 输出接收器——测试断言 `buf.String()`（:262）与 hub 协程的日志写入（readLoop 的 defer"玩家断开"等）并发，`go test -race` 下 `TestJoinConfigLocksOnFirstJoin` / `TestJoinInvalidConfigFallsBackToDefaults` 必挂（已实测复现）。改为 mutex 保护的 sink（Write/String 均持锁），断言内容与测试语义不变。
2. **CI 加 race 门控**：`.github/workflows/ci.yml` 的 go test 改为 `-race`——让数据竞争类问题（如已修复的 Engine.resetC 竞争、本次的测试 sink）在 CI 即暴露，不再潜伏。

背景：Engine.resetC 的产品侧竞争已由 fb7a258 修复（currentResetC 锁内访问 + wakePump 持锁写点），本票不涉及引擎代码。

**Blocked by:** None (can start immediately)

**Status:** resolved

- [x] `go test -race -count=1 ./internal/hub/` 全绿（captureLogger 换线程安全 sink，测试断言语义零变化）
- [x] `go test -race -count=1 ./...`（server）全绿——race 门控在本地全量可复现
- [x] `.github/workflows/ci.yml` 的 go test 步骤带 `-race`
- [x] 既有测试除 sink 载体外零改动（断言、期望值、用例集不变；diff 审查可证）
