# CI 首次运行失败分析（run 40629094422）

- **仓库**：zklmiao/CodeWar
- **触发**：push `main`（commit `a203a2f`，"docs: refresh tech badge row (add gRPC and license badge)"）
- **运行时间**：2026-10-28T03:52:02Z，总耗时 3m47s
- **结果**：server ✓ 3m43s · judge ✓ 38s · **client ✗ 6s**
- **失败步骤**：client job 的 `pnpm/action-setup@v4`（setup-node、pnpm install、pnpm test、pnpm build 均未执行）

## 1. 问题描述

client job 在 `pnpm/action-setup@v4` 的自安装阶段立即失败，日志原文：

```txt
2026-10-28T03:52:08.3895431Z Error: Multiple versions of pnpm specified:
2026-10-28T03:52:08.3897675Z   - version 12 in the GitHub Action config with the key "version"
2026-10-28T03:52:08.3900104Z   - version pnpm@12.4.1 in the package.json with the key "packageManager"
2026-10-28T03:52:08.3904110Z Remove one of these versions to avoid version mismatch errors like ERR_PNPM_BAD_PM_VERSION
2026-10-28T03:52:08.3908832Z     at readTarget (/home/runner/work/_actions/pnpm/action-setup/v4/dist/index.js:1:7537)
```

即：工作流配置与 `package.json` 同时声明了 pnpm 版本，action 拒绝启动。**client 的 vitest 与 vite build 在 CI 中从未运行**，因此本次红灯不代表代码或依赖有缺陷，而是 CI 配置冲突。

## 2. 根因分析

### 2.1 直接原因：pnpm 版本双重声明

- `.github/workflows/ci.yml` 的 client job：

  ```yaml
  - uses: pnpm/action-setup@v4
    with:
      version: 12
  ```

- 仓库根 `package.json`：

  ```json
  "packageManager": "pnpm@12.4.1"
  ```

`pnpm/action-setup@v4` 对「action 的 `version`」与「package.json 的 `packageManager`」同时存在的情况做 **fail-fast** 校验（避免运行期出现 `ERR_PNPM_BAD_PM_VERSION`），因此直接报错退出。

### 2.2 为什么 action 读到的是根目录 package.json

`defaults.run.working-directory: client` **只作用于 `run:` 步骤，不作用于 `uses:` action**。`pnpm/action-setup` 默认读取仓库根目录的 `package.json`；根 `package.json` 含 `packageManager: "pnpm@12.4.1"`，而 `client/package.json` 并未声明该字段。

### 2.3 为什么现在才暴露

这是仓库首次推送到 GitHub、首次真正执行 GitHub Actions。`.github/workflows/ci.yml` 自 01 号票落地起就存在，但此前从未在 GitHub runner 上运行过。同一运行中 server（go build + go test -race）与 judge（pytest）均通过，进一步说明代码、依赖与测试本身无问题。

### 2.4 影响面

- 确定性失败：任何 push 都会复现，不是 flaky，也不受网络影响。
- client 侧的 test/build 门控在 CI 中处于**未执行**状态（此前一直如此，只是从未在 GitHub 上体现）。
- 不影响本地开发、运行时行为与已发布代码；仅 GitHub 上的 CI 状态为红。

## 3. 解决方案

### 方案 A（推荐）：以 `packageManager` 为唯一版本来源

删除 workflow 中 `pnpm/action-setup` 的 `version` 输入，让 action 自动读取根 `package.json` 的 `packageManager`：

```yaml
- uses: pnpm/action-setup@v4
```

- 优点：单一真相源；本地 Corepack 与 CI 使用同一 pnpm 版本（12.4.1）；未来升级只需改 `package.json`。
- 注意：action 读取的是根目录 `package.json`，`working-directory: client` 不影响该行为。

### 方案 B（不推荐）：删除 `packageManager` 字段

保留 workflow 的 `version: 12`，删除根 `package.json` 的 `packageManager`。

- 缺点：丢失 Corepack 的本地版本锁定；`12` 与 `12.4.1` 之间仍可能漂移，同类问题会在版本不一致时再次出现。

### 可选顺带修复（非本次失败原因）

1. **server job 缓存未命中**：日志告警
   `Restore cache failed: Dependencies file is not found in /home/runner/work/CodeWar/CodeWar. Supported file pattern: go.sum`
   ——`actions/setup-go@v5` 的缓存依赖路径默认在仓库根查找 `go.sum`，而本仓库的实际路径是 `server/go.sum`。
   修复：为该 step 增加
   ```yaml
   with:
     cache-dependency-path: server/go.sum
   ```
   （`go-version-file` 已正确指向 `server/go.mod`。）只影响缓存命中率，不影响正确性。

2. **Node.js 20 弃用告警**：`actions/checkout@v4`、`actions/setup-go@v5`、`pnpm/action-setup@v4` 被强制运行在 Node 24 上，当前仅为 warning；后续可择机升级 action 版本。

3. **runner 迁移提示**：`ubuntu-latest` 将于 2026-10-19 起迁移 Ubuntu 26（info 级，无需动作）。

## 4. 验证方案（修复后）

1. 本地干净验证：`pnpm -C client install --frozen-lockfile && pnpm -C client test && pnpm -C client build` 全绿。
2. 远端验证：push 修复提交后 `gh run watch`；期望 client job 日志出现 `Installed pnpm@12.4.1`，随后 setup-node → install → test → build 全部通过。
3. 回归确认：server 与 judge 两个 job 保持绿（本次已通过，修复不涉及其路径）。

## 5. 结论

- **失败性质**：CI 工作流配置冲突（pnpm 版本双重声明），非代码、依赖或测试缺陷。
- **建议处置**：采用方案 A（workflow 删除 `version` 一行）；如需一并消除缓存告警，加 `cache-dependency-path: server/go.sum`。
- **风险**：极低，改动仅限 `.github/workflows/ci.yml`，不触碰运行时逻辑。

> 本文只做分析，未修改任何 CI 配置；处置方式待人工 review 后决定。

## 6. 处置记录（2026-10-30）

经人工 review 确认采用 **方案 A + 可选修复 1**，已实施：

- `.github/workflows/ci.yml`：client job 删除 `pnpm/action-setup` 的 `version: 12`（根 `package.json` 的 `packageManager: pnpm@12.4.1` 成为唯一版本源）；server job 的 `setup-go` 增加 `cache-dependency-path: server/go.sum`。
- 修复提交 `0dd3dd2`（"ci: resolve pnpm version conflict and fix go cache path"）推送后，run `40974981793` **三 job 全绿**：client 22s（此前 6s 失败）、server 4m5s（"Restore cache failed" 告警消失）、judge 36s。
- 剩余注解仅 Node.js 20 弃用告警与 ubuntu-latest 迁移提示，均为非阻塞项，见「可选顺带修复」第 2/3 条，择机处理。
