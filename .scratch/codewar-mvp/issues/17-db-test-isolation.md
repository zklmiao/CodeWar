# 17: DB 门控测试互相清库（codewar_test 共用）

**What to build:** 让每个使用真实 Postgres 的测试包拥有独立测试库，消除 `go test ./...` 并行包之间的 `TRUNCATE` 互相干扰；把 hub 已采用的专属库模式抽成共享测试辅助包。

**Blocked by:** 无

**Status:** ready-for-agent

- [ ] `store` / `accounts` / `hub` 各用独立测试库（如 `codewar_test_store` / `codewar_test_accounts` / `codewar_test_hub`）
- [ ] 共享 `server/internal/testdb` 辅助：解析 `CODEWAR_TEST_DSN` → 校验源库名含 `test` → 目标库不存在则 `CREATE DATABASE` → 返回目标 DSN；由调用方 `store.Open` 自动迁移
- [ ] 带 `CODEWAR_TEST_DSN` 直接 `go test ./...` 连续 5 次并行全绿，无需 `-p 1`
- [ ] 无 DSN 时仍按既有约定 `t.Skip`（CI 无 DB 保持绿）
- [ ] 库名必须含 `test` 的破坏性护栏保留（源 DSN 不满足即 skip）
- [ ] 无生产代码行为变化（辅助包只被测试 import）
- [ ] README 或 `docs/agents/issue-tracker.md` 记录测试库命名约定

## 根因（10 号票验收期间记录）

- `server/internal/store/store_test.go` 与 `server/internal/accounts/accounts_test.go` 都连 `codewar_test`，各自 `TRUNCATE ... CASCADE`。
- `go test ./...` 默认并行跑不同包的测试二进制：store 的 TRUNCATE 会删掉 accounts 刚注册的会话，表现为 `/api/tutorial`/`/api/me` 偶发 401。
- `server/internal/hub/hub_store_integration_test.go` 已用专属库 `codewar_test_hub` 规避，说明范式已在仓库内存在，只是未推广。

## 方案取舍

- **采用**：每包独立数据库。迁移/sqlc/测试代码零改动，隔离最强。
- **不采用**：每包独立 schema / `search_path`（迁移与 sqlc 需额外处理，收益低）。
- **不采用**：长期 `-p 1`（掩盖包间耦合、拖慢 CI；只作为本票落地前的临时 workaround）。
