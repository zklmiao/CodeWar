# CodeWar（代码战争）

网页端 Python 编程对抗游戏：双方在 5×5 网格竞技场各控一个角色，每回合从同时展示的多道代码题中自选一道作答——解出攻击题向对手开火（同题镜像竞速，越快伤害越高）、移动题走位闪避、防御题格挡减伤、技能题放大招。HP 归零者败。

## 文档地图

- **Spec**: [docs/specs/001-codewar-mvp.md](docs/specs/001-codewar-mvp.md) —— MVP 全量规格（用户故事 / 实现决策 / 测试决策 / 数值初稿 / 开放问题）
- **票据**: `.scratch/codewar-mvp/issues/` —— 13 张 tracer-bullet 票，每张含阻塞关系与验收清单
- `ref_resources/` —— 参考资料（CodeCombat 源码、The Farmer Was Replaced 拆解），仅本地存在，不入库

## 并行开发约定（git worktree）

- worktree 池在仓库内 `.worktrees/`（已 gitignore），主工作区固定在 `main` 且保持全绿
- 一票一 worktree 一分支：分支名 `ticket/<NN>-<slug>`，与票据文件名对应
- 每个新会话在对应 worktree 里实现一张票（`/implement`），提交到当前分支，验收勾完由人关票
- 端口规范：`PORT = 30NN`（NN = 票号），并行 dev server 互不抢占
- 票合入 main 后：`git worktree remove .worktrees/<NN>-<slug>` 并删除分支

## 当前波次

Wave 1 进行中：`01 Tracer（热座最小对局）` + `13 美术占位规范`（两者均无阻塞，可并行）。
