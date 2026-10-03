# 02: 判题沙箱硬化

**What to build:** 用生产级沙箱判题（Judge：judge(code, testCases) → {passed, category, durationMs}）替换 Tracer 中的简易判题。100% 用例通过才算通过；死循环在时限内被终止、资源超限与联网尝试被安全拒绝。这是安全边界票：判题器必须能在公网环境承载不可信代码。判题管线流程、verdict 契约与遥测点见 docs/design/002-question-system.md §5-§6；judge 为独立 Python 微服务（ADR-0001），沙箱机制 ADR 在该服务边界内选定（容器运行时访问权或用户命名空间），Compose 部署形态已为此预留。

**Blocked by:** 01 (Tracer——热座最小对局)

**Status:** ready-for-agent

- [ ] 正确代码全部通过时返回 passed；durationMs（代码执行耗时）正确上报，并与玩家解题耗时（ClockPort 计时、驱动时间系数）严格区分
- [ ] 失败反馈只含类别（wrong-answer / runtime-error / timeout / limit-exceeded / syntax），不回传隐藏用例通过计数
- [ ] 死循环在时限内被终止并判失败，服务不垮
- [ ] 联网 / 资源超限尝试被安全拒绝
- [ ] 部分用例通过 = 判失败（不允许部分伤害）
- [ ] 换用沙箱判题后，真实对局中上述行为可复现（事件流断言）
