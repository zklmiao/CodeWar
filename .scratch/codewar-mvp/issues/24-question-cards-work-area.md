# 24: 题卡 + 主工作区范式重构 + sticky 回合状态条

**What to build:** UI/UX 优化核心票（a+b 档）。QuestionPanel 从"四槽纵排、各嵌一个 CodeMirror"（右栏总高 ~1400px+、整页长滚动）重构为"题卡 + 主工作区"：四张紧凑题卡（横排一行）+ 激活制共享大编辑器；回合状态上移为 sticky 条，滚动时倒计时始终在场。设计依据：游戏机制"一回合仅一题生效"——选卡→聚焦的心智模型；`codes` 已按 action 分槽（QuestionPanel.vue:21），切换不丢码；同时仅挂载一个 CodeMirror（现挂四个）。验收锚点：**1366×768 下主工作流无页面级滚动**。

**现状证据（2026-10-25 UI 扫描）：** QuestionPanel.vue:138-202 四槽纵排各嵌 CodeEditor；:227-230 题面 12px、max-height 96px 内滚；:129-135 回合/倒计时/状态在面板头部（滚下去即不可见）；skill-cd.spec.ts 依赖 `.slot.skill`/`.submit.skl`/`.cooldown` 选择器；教程模式 slots props（:15）只渲染部分槽。

**Blocked by:** 23（右栏空间与布局来自 23 的重排）

**Status:** resolved

- [x] 四张紧凑题卡横排一行：槽名 + 题面 clamp 2-3 行（点卡激活后在主工作区读全文）+ 难度徽标位（占位，样式归 25）；技能卡冷却角标"还剩 N 回合"（沿用 `store.skillCdRounds`，20 号票事件推导语义零改动）
- [x] 激活制主工作区：点卡激活 → 完整题面（去除 96px 内滚，超长在工作区内滚动）+ signature + 共享 CodeEditor + targeting + 提交按钮；`codes` 按 action 分槽保留，切换不丢码；同时仅挂载一个 CodeMirror
- [x] 默认激活：回合开始未点选时默认激活**第一个可见卡**（PvP/PvE 恒为攻击卡；教程模式 slots 可能不含攻击——L2 仅 move），工作区即刻可用、无空态
- [x] 棋盘点选联动：回合活跃且未行动时点棋盘合法目标格 → 自动激活移动卡（目标格回显迁至工作区），点选动线一步到位
- [x] 已行动语义与现状一致：提交后仍可切换/查看/编辑其他题卡，仅禁用各卡提交按钮（不加"禁切换"）
- [x] 已行动状态：当前卡高亮"已提交 ✓"、其余卡降权
- [x] 教程模式 `slots` props 语义保留（只渲染对应题卡）
- [x] sticky 回合状态条（HpBar 下方）：回合号/倒计时/75s 细进度条/已行动/锁定/本场配置；tabular-nums；≤10s 变红；动效遵守 prefers-reduced-motion
- [x] 测试迁移：主迁移面 = skill-cd.spec.ts（`.slot`/`.submit`/`.cooldown` 结构选择器，grep -o 口径 5 处）；match-config.spec 为纯 store 状态机（配置行迁入 sticky 条零迁移）、match-view.spec 走 data-testid 且测入口屏、class-panel/match-store 为纯状态机——沿用 data-testid 策略则零迁移；行为断言不降级（灰置/递减/恢复全保留）；`npx vitest run` 全绿 + `npx vite build` 通过
- [x] 验收锚点：1366×768 下棋盘/状态条/题卡/主工作区一屏内（日志与超长题面允许内部滚动），无页面级滚动（浏览器 1366×768 窗口人工目检＋尺寸记录；jsdom 无布局，不做自动化断言）
