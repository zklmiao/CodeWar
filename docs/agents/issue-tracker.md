# Issue tracker: Local Markdown

本仓库的 issue 与规格以 markdown 文件形式存放（纯本地仓库，无远端 tracker）。

## Conventions

- Spec 存放于 `docs/specs/<NNN>-<slug>.md`（三位编号递增；设计与数值的真相源）
- 实现票据一票一文件：`.scratch/<feature-slug>/issues/<NN>-<slug>.md`，从 `01` 起按依赖序编号，禁止单一合并大文件
- 当前活跃 feature：`codewar-mvp`
- Triage 状态记录在每个票据文件头部的 `Status:` 行（角色字符串见 `triage-labels.md`）
- 评审意见与会话结论追加到文件底部 `## Comments` 标题下

## When a skill says "publish to the issue tracker"

- Spec → 在 `docs/specs/` 创建新文件（编号顺延）
- Ticket → 在 `.scratch/<feature-slug>/issues/` 创建新文件（必要时创建目录）

## When a skill says "fetch the relevant ticket"

读取被引用路径的文件。用户通常直接给出路径或票号。

## Wayfinding operations

Used by `/wayfinder`. The **map** is a file with one **child** file per ticket.

- **Map**: `.scratch/<effort>/map.md` (the Notes / Decisions-so-far / Fog body).
- **Child ticket**: `.scratch/<effort>/issues/NN-<slug>.md`, numbered from `01`, with the question in the body. A `Type:` line records the ticket type (`research`/`prototype`/`grilling`/`task`); a `Status:` line records `claimed`/`resolved`.
- **Blocking**: a `Blocked by: NN, NN` line near the top. A ticket is unblocked when every file it lists is `resolved`.
- **Frontier**: scan `.scratch/<effort>/issues/` for files that are open, unblocked, and unclaimed; first by number wins.
- **Claim**: set `Status: claimed` and save before any work.
- **Resolve**: append the answer under an `## Answer` heading, set `Status: resolved`, then append a context pointer (gist + link) to the map's Decisions-so-far in `map.md`.
