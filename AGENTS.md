# AGENTS.md

CodeWar（代码战争）：网页端 Python 编程对抗游戏。

## 真相源

- **Spec**：`docs/specs/001-codewar-mvp.md` —— 设计与数值的唯一权威。做任何设计决策、写任何测试前，先核对其对应小节；领域术语表在文末「附注」。
- **票据**：`.scratch/codewar-mvp/issues/` —— 每张票含端到端交付描述、Blocked by、验收清单。frontier（当前可开工）= 阻塞全部清零的票。
- Spec 与票据冲突时：先向人报告冲突，再动手。

## 工作流

1. 选一张 frontier 票，进入它的 worktree（`.worktrees/<NN>-<slug>`，分支 `ticket/<NN>-<slug>`；worktree 布局见 README）。
2. 读票 → 读 spec 相关小节 → 实现。
3. 完成标准 = 票据验收清单逐条可验证。测试在三个 seam 边界上断言外部行为（MatchEngine 对局事件流 / Judge / QuestionFactory），错误路径优先，细则见 spec「测试决策」。
4. 提交到当前票分支，逐条汇报验收结果；勾选与关票由人完成。
5. main 保持 green：功能提交只出现在票分支。

架构决策（技术栈、对局同步方案等）写成 ADR 放 `docs/adr/`，随对应票据落地。

## 环境

- 并行开发时 dev server 端口 `PORT=30NN`（NN=票号），互不抢占。
- `ref_resources/` 是外部项目参考（CodeCombat 源码、The Farmer Was Replaced 拆解）：用途限于设计思路；其代码与美术资产许可不同，保持零复制、零 import。
- 文档与代码注释用中文；提交信息用 conventional commits（英文）。
