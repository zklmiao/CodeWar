# 30: 杂项清理与结算页统计（优化审查第一批）

**What to build:** 来自 2026-10-27 优化审查（`.scratch/codewar-mvp/optimization-review-2026-10-27.md`）的第一批打包小件，五个零风险条目：1) README「当前波次」严重过时（还停在 Wave 1 / frontier=02/03/04/07），误导选票；2) App.vue 导航标题泄漏开发标注「07 号票 · 账号与 ELO」；3) QuestionPanel 的 100ms 心跳 `setInterval` 无 `onBeforeUnmount` 清理（离开对局页后定时器常驻，内存泄漏）；4) EventLog 纯文本无类型着色，战况不可一眼读；5) 结算页只有胜负+ELO，无任何对局统计。

**Blocked by:** None (can start immediately)

**Status:** resolved

**现状证据（2026-10-27，HEAD `161d4f9`）：**

- README.md「当前波次」小节：仍写 Wave 1 已交付、frontier=02/03/04/07、共 14 张票——实际已推进到 29 号票关票，frontier=08/11、25 挂起。
- App.vue:34：`<h1>CodeWar · 代码战争 <small>07 号票 · 账号与 ELO</small></h1>`——票号标注进了生产 UI。
- QuestionPanel.vue:129：`setInterval(() => (now.value = Date.now()), 100)` 无清理；对照 RoundStatusBar.vue:14-15 有 `onBeforeUnmount(() => clearInterval(timer))`。
- stores/match.ts:389-390：`log.push(...)` 后 `if (this.log.length > 50) this.log.shift()`——日志上限 50 条，**终局统计不可依赖日志文本，必须由事件流单独累计**。
- EventLog.vue:9-12：逐行渲染 `store.log`，无 kind 分类。
- EndOverlay.vue：仅 title/reason/ELO 行。

- [x] README「当前波次」刷新至实态（概览 01-29 已关、frontier=08/11、25 挂起待美术票；与 optimization-review-2026-10-27.md 互链）
- [x] App.vue 标题「07 号票 · 账号与 ELO」开发标注移除（导航链接、用户名旁 ELO 展示行为零变化）
- [x] QuestionPanel 心跳补 `onBeforeUnmount` 清理（组件卸载后无残留定时器；vitest fake timers 断言或等价证据）
- [x] LogEntry 增加 `kind` 分类（按事件类型：伤害/治疗/闪避/系统等），EventLog 按 kind 着色（伤害红/治疗绿/闪避黄/系统默认次级色）；日志文案与 seq 语义零变化
- [x] match store 由事件流累计终局统计：回合数、双方总伤害、双方闪避次数（不依赖 50 条日志上限）；EndOverlay 增统计区块（匿名局同样显示，ELO 行为不变）
- [x] 新增 vitest：store 统计累计正确性 + EventLog kind 着色渲染；沿用既有 spec 文件组织（match-store.spec / EndOverlay.spec / 新 eventlog.spec）
- [x] `pnpm -C client test` 与 `pnpm -C client build` 全绿；match.proto 与服务端零改动

> 来源：optimization-review-2026-10-27.md「推荐开票组合」第 1 项。

> **关票记录（2026-10-27 合并后验收）**：ticket/30 三个提交合入 main（merge 49204dd）。验证：分支 `pnpm -C client test` 56/56 → 合并后 main 64/64；`pnpm -C client build` 通过；match.proto 与 server/ 零改动（diff 核对）。对应测试：match-store.spec（统计累计/LogEntry.kind）、eventlog.spec（kind 着色）、EndOverlay.spec（统计区块/匿名局）、QuestionPanel.spec（卸载心跳清理）。
