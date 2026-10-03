# 09: 单人 PvE 练习模式（模拟 Bot）

**What to build:** 单人练习模式：对手是模拟 Bot——按难度配置解题耗时分布与正确率、经 SubmissionPort 提交（引擎对 Bot 无任何特判）；PvE 模式选择界面（题型 + 最高难度），与 PvP 配置一致。零压力练手与 build 验证场景。

**Blocked by:** 06 (完整题面板 + 技能系统 + 3 职业)

**Status:** ready-for-agent

- [ ] 可单人完成一整局并分出胜负
- [ ] Bot 行为随所选难度可感知变化（更高难度 = 解题更快/更准）
- [ ] Bot 经 SubmissionPort 接入，MatchEngine 无 Bot 特判代码路径
- [ ] PvE 与 PvP 使用同一套对局配置（题型/难度上限/抽题策略）
