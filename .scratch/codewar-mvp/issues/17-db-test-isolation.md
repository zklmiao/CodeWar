# 17: DB 门控测试互相清库（codewar_test 共用）

**What to build:** 让每个使用真实 Postgres 的测试包拥有独立测试库，消除 `go test ./...` 并行包之间的 `TRUNCATE` 互相干扰；把 hub 已采用的专属库模式抽成共享测试辅助包。

**Blocked by:** 无

**Status:** resolved

- [x] `store` / `accounts` / `hub` 各用独立测试库（如 `codewar_test_store` / `codewar_test_accounts` / `codewar_test_hub`）
- [x] 共享 `server/internal/testdb` 辅助：解析 `CODEWAR_TEST_DSN` → 校验源库名含 `test` → 目标库不存在则 `CREATE DATABASE` → 返回目标 DSN；由调用方 `store.Open` 自动迁移
- [x] 带 `CODEWAR_TEST_DSN` 直接 `go test ./...` 连续 5 次并行全绿，无需 `-p 1`
- [x] 无 DSN 时仍按既有约定 `t.Skip`（CI 无 DB 保持绿）
- [x] 库名必须含 `test` 的破坏性护栏保留（源 DSN 不满足即 skip）
- [x] 无生产代码行为变化（辅助包只被测试 import）
- [x] README 或 `docs/agents/issue-tracker.md` 记录测试库命名约定

## 根因（10 号票验收期间记录）

- `server/internal/store/store_test.go` 与 `server/internal/accounts/accounts_test.go` 都连 `codewar_test`，各自 `TRUNCATE ... CASCADE`。
- `go test ./...` 默认并行跑不同包的测试二进制：store 的 TRUNCATE 会删掉 accounts 刚注册的会话，表现为 `/api/tutorial`/`/api/me` 偶发 401。
- `server/internal/hub/hub_store_integration_test.go` 已用专属库 `codewar_test_hub` 规避，说明范式已在仓库内存在，只是未推广。

## 方案取舍

- **采用**：每包独立数据库。迁移/sqlc/测试代码零改动，隔离最强。
- **不采用**：每包独立 schema / `search_path`（迁移与 sqlc 需额外处理，收益低）。
- **不采用**：长期 `-p 1`（掩盖包间耦合、拖慢 CI；只作为本票落地前的临时 workaround）。

## 实现记录（待人工验收）

- 新增 `server/internal/testdb`：读取 `CODEWAR_TEST_DSN`、校验源库名含 `test`、
  目标库 `codewar_test_<suffix>` 不存在则 `CREATE DATABASE`，返回目标 DSN；
  连接/迁移仍由调用方 `store.Open` 完成。只被测试 import。
- `store` → `codewar_test_store`；`accounts` → `codewar_test_accounts`；
  `hub` → `codewar_test_hub`（复用同一 helper）。
- 验证：
  - 带 DSN 直接 `go test ./internal/store/... ./internal/accounts/... ./internal/hub/...`
    并行通过（不再需要 `-p 1`）。
  - `go test ./internal/store/... ./internal/accounts/... -count=5` 并行通过。
  - 带 DSN `go test ./...` 并行全绿。
  - 无 DSN 时 DB 集成测试仍 `SKIP`（CI 无 DB 保持绿）。
- 注意：迁移版本变更后，若旧测试库已迁移到更高版本，需 `DROP DATABASE`
  对应 `codewar_test_*` 后重跑（跨分支共享本地测试库的固有现象，不在本票范围）。

