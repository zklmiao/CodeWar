# 25: 视觉打磨第三级（SVG 图标、动效落地、职业色血条、难度徽标）

**What to build:** UI/UX 清单第三级（c 档）。四件事：1) 启用 `design/components` 已产出未使用的 cw-icon-* SVG 替换 emoji；2) 14 号票在 styleguide 做好的动效四件套（冷却扫描/低血脉动/悬停反馈/面板入场）落地 client，reduced-motion 静止；3) HpBar 双方色从恒定红蓝改为跟随所选职业 token；4) 题面难度徽标与色彩语义系统化（对接工厂难度档位）。全部落在 24 号票的新卡结构之上。

**现状证据（2026-10-25 UI 扫描）：** design/components 下 cw-icon-*.svg 与 placeholder.css 的动效组件在 client 零引用（grep 无结果）；client 现用 emoji（GameGrid.vue:94 🛡 护盾角标、QuestionPanel 槽标题 ⚔🏃✨🛡、EndOverlay.vue:9-10 🏆💀🤝）；HpBar.vue:7-8 恒定 P1 红 P2 蓝，未用职业 token（--cw-color-class-*，tokens.css:30-32）。

**Blocked by:** 24（图标/徽标/动效的载体是 24 的题卡结构；HpBar 职业色一项独立可先行）

**Status:** ready-for-agent

- [ ] cw-icon-* SVG 替换 emoji：槽题卡、护盾角标、终局面板；占位图标风格与 13 号票规范一致（零复制 ref_resources 约束不变，全部用自产资产）
- [ ] 动效四件套落地 client：冷却扫描（技能卡灰置）、低血脉动（HP 低时）、悬停反馈、面板入场；prefers-reduced-motion 全部静止
- [ ] HpBar 双方色跟随所选职业 token（`--cw-color-class-*`），未选职业/加载中回退现红蓝 token
- [ ] 难度徽标与色彩语义：题卡/题面的难度档位可视化（L1-L5），色阶定义进 tokens.css
- [ ] 验收：`npx vitest run` 全绿 + `npx vite build` 通过；styleguide 与 client 无漂移（同一 token/图标来源）
