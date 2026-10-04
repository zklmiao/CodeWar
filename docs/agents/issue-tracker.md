# Issue tracker: Local Markdown

本仓库的 issue 与规格以 markdown 文件形式存放（纯本地仓库，无远端 tracker）。

## Conventions

- Spec 存放于 `docs/specs/<NNN>-<slug>.md`（三位编号递增；设计与数值的真相源）
- 实现票据一票一文件：`.scratch/<feature-slug>/issues/<NN>-<slug>.md`，从 `01` 起按依赖序编号，禁止单一合并大文件
- 当前活跃 feature：`codewar-mvp`
- Triage 状态记录在每个票据文件头部的 `Status:` 行（角色字符串见 `triage-labels.md`；被阻塞的票同样标 `ready-for-agent`，是否可开工以 `Blocked by` 图为准）
- 评审意见与会话结论直接记入票据正文相关小节（如 01 票尾部"已知偏差记录"），不设 `## Comments` 固定栏

## When a skill says "publish to the issue tracker"

- Spec → 在 `docs/specs/` 创建新文件（编号顺延）
- Ticket → 在 `.scratch/<feature-slug>/issues/` 创建新文件（必要时创建目录）

## When a skill says "fetch the relevant ticket"

读取被引用路径的文件。用户通常直接给出路径或票号。

## Frontier / Claim / Resolve（本仓库实际用法）

上游 `/wayfinder` 的 map 流程（`map.md`、`Type:` 行、`claimed` 中间态、`## Answer`）在本仓库未采用；skill 提到时以本节为准。

- **Blocking**：票据头部 `Blocked by:` 行列出前置票号；当前置票均为 `resolved` 时阻塞清零。
- **Frontier**：`Status` 非 `resolved`/`wontfix` 且阻塞清零的票。注意 `Status` 不区分阻塞与否，选票必须解 `Blocked by` 图。
- **Claim**：进入对应 worktree（`.worktrees/<NN>-<slug>`，分支 `ticket/<NN>-<slug>`）开工；本仓库不使用 `claimed` 中间态。
- **Resolve**：验收清单逐条可验证后，由人确认并将 `Status:` 置为 `resolved`（合入 main 后；终态定义见 `triage-labels.md`）。
