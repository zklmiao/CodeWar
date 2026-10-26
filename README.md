# CodeWar（代码战争）

网页端 Python 编程对抗游戏：双方在 5×5 网格竞技场各控一个角色，每回合从同时展示的多道代码题中自选一道作答——解出攻击题向对手开火（同题镜像竞速，越快伤害越高）、移动题走位闪避、防御题格挡减伤、技能题放大招。HP 归零者败。

## 文档地图

- **Spec**: [docs/specs/001-codewar-mvp.md](docs/specs/001-codewar-mvp.md) —— MVP 全量规格（用户故事 / 实现决策 / 测试决策 / 数值初稿 / 开放问题）
- **票据**: `.scratch/codewar-mvp/issues/` —— 一票一文件，含端到端交付描述、`Blocked by` 阻塞关系与验收清单；当前波次见下文
- **优化审查**: [.scratch/codewar-mvp/optimization-review-2026-10-27.md](.scratch/codewar-mvp/optimization-review-2026-10-27.md) —— 29 号票后的三轴审查（功能 / UI / 工程质量），30+ 号票的来源任务列表
- `ref_resources/` —— 参考资料（CodeCombat 源码、The Farmer Was Replaced 拆解），仅本地存在，不入库

## 并行开发约定（git worktree）

- worktree 池在仓库内 `.worktrees/`（已 gitignore），主工作区固定在 `main` 且保持全绿
- 一票一 worktree 一分支：分支名 `ticket/<NN>-<slug>`，与票据文件名对应
- 每个新会话在对应 worktree 里实现一张票（`/implement`），提交到当前分支，验收勾完由人关票
- 端口规范：`PORT = 30NN`（NN = 票号），并行 dev server 互不抢占
- 票合入 main 后：`git worktree remove .worktrees/<NN>-<slug>` 并删除分支

## 测试（本地）

- Go 单元测试：`cd server && go test ./...`（无 DB 时 DB 集成测试自动 skip）。
- 真实 Postgres 集成测试由 `CODEWAR_TEST_DSN` 门控，源 DSN 的库名必须含 `test`。
  每个测试包使用独立库（`codewar_test_store` / `codewar_test_accounts` /
  `codewar_test_hub`），缺失时由 `server/internal/testdb` 自动创建，因此可直接
  并行跑包，不会互相 TRUNCATE。本地示例（compose 的 postgres 映射在 25432）：
  `cd server && CODEWAR_TEST_DSN='postgres://codewar:codewar-dev@localhost:25432/codewar_test?sslmode=disable' go test ./...`
- Judge：`python -m pytest judge/tests -q`。
- 客户端：`pnpm -C client test`、`pnpm -C client build`。

## 当前波次

截至 2026-10-27：`01`-`29` 号票已全部交付关票（`Status: resolved`），MVP 主线落地——热座最小对局、判题沙箱硬化、三槽行动与形状、出题工厂、对局配置、四槽面板与职业技能、账号与 ELO、PvE 练习机器人、新手教程、美术占位规范、霓虹质感、DB 测试隔离、技能形状语义等。

- **frontier（阻塞清零，可开工）**：`08 在线排行 1v1` + `11 防作弊硬化`。
- **挂起**：`25 视觉打磨清单`（`needs-info`）——待美术票启动后一并处理。
- **进行中**：来自 [optimization-review-2026-10-27.md](.scratch/codewar-mvp/optimization-review-2026-10-27.md) 第一批新票——`30 杂项清理与结算页统计`、`31 投降认输 + hub 连接卫生`（08 前置）；后续候选（回放/错题本、CI 门控、键盘动线等）见该审查文件的推荐开票组合。

选票时以票据 `Blocked by` 图为准。
