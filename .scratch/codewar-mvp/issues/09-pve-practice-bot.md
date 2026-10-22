# 09: 单人 PvE 练习模式（模拟 Bot）

**What to build:** 单人练习模式：对手是模拟 Bot——按难度配置解题耗时分布与正确率、经 SubmissionPort 提交（引擎对 Bot 无任何特判）；PvE 模式选择界面（题型 + 最高难度），与 PvP 配置一致。零压力练手与 build 验证场景。

**Blocked by:** 06 (完整题面板 + 技能系统 + 3 职业)

**Status:** resolved

- [x] 可单人完成一整局并分出胜负
- [x] Bot 行为随所选难度可感知变化（更高难度 = 解题更快/更准）
- [x] Bot 经 SubmissionPort 接入，MatchEngine 无 Bot 特判代码路径
- [x] PvE 与 PvP 使用同一套对局配置（题型/难度上限/抽题策略）

> **验收记录（人工验收 + 全量回归）**：
> - 验收 1：`TestFullPveMatchReachesDecision`（引擎 seam 整局终局）+ `TestPveSingleJoinStartsAndMarksBot`（hub 单连接 e2e）。
> - 验收 2：30 局/档同 seed 配对——easy 平均 58098ms 胜 0/30、normal 35182ms 胜 3/30、hard 18714ms 胜 13/30（easy/hard = 3.10×）。
> - 验收 3：`TestMatchPackageHasNoBotReferences`（词边界 + AST 标识符扫描）；Bot 每次提交均有正常判题事件（ok/wrong-answer），无 locked/judge-error 旁路。
> - 验收 4：`TestPveAndPvpShareQuestionConfig` 用引擎 seam 比较两种配置下 10 回合 `PanelReady` 与含技能槽的首回合快照面板；同一 `newQuestions` 构造器/默认值/seed 派生。
> - 手工验收：困难档单窗口打满，`P2 · 机器人（困难）`、Bot 经真实判题出手、终局无 ELO；简单+游侠观察到 Bot 45–70s 出手、错误自损 3。
> - 评审期修复：技能冷却登记顺序导致回合空转（9f5e52a）、入口页 TDZ 渲染失败（ef4d254）、废弃 WAITING 大厅阻塞新配置（aba61d6）。
> - 已知既有问题（非本票）：`cmd/smoke` 的 P1/P2 同毫秒 tie-break 间歇 `rounds-cap`（main 同样复现），不作为本票门禁。
