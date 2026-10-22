# CodeWar（代码战争）

网页端 Python 编程对抗游戏：双方在 5×5 网格竞技场各控一个角色，每回合从同时展示的多道代码题中自选一道作答——解出攻击题向对手开火（同题镜像竞速，越快伤害越高）、移动题走位闪避、防御题格挡减伤、技能题放大招。HP 归零者败。

## 文档地图

- **Spec**: [docs/specs/001-codewar-mvp.md](docs/specs/001-codewar-mvp.md) —— MVP 全量规格（用户故事 / 实现决策 / 测试决策 / 数值初稿 / 开放问题）
- **票据**: `.scratch/codewar-mvp/issues/` —— 14 张 tracer-bullet 票，每张含阻塞关系与验收清单
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

Wave 1 已交付：`01 Tracer（热座最小对局）` + `13 美术占位规范` + `14 霓虹质感精修`（均已合入 main，`Status: resolved`）。

当前 frontier（阻塞已清零，可开工）：`02 判题沙箱硬化` + `03 移动防御攻击形状` + `04 出题工厂管线` + `07 账号战绩与 ELO`（四者均只被已关闭的 01 阻塞）。其余未开工票均有阻塞：`05←04`、`06←03/04`、`08←06/07`、`09←06`、`10←09`、`11←08/02`、`12←04`；选票时以票据 `Blocked by` 图为准。
