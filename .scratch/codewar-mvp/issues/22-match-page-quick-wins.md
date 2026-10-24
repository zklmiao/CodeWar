# 22: 对局页 quick wins（token 合规、字号提档、覆盖层 z-index、移动端防溢出）

**What to build:** UI/UX 优化清单第一级（a/d 档 quick wins 打包，零新依赖，除字号提档外零行为变化）。四件事：1) 裸色值还债归位 `design/tokens.css`（tokens.css:5-8 规定"任何色值/字号写在别处即为违规"，现状十余处违规）；2) px 字号 → `--cw-font-size-*`，正文/等宽接 `--cw-font-sans`/`--cw-font-mono`（两者在 client 目前零引用），题面 12px → 13px 档；3) EndOverlay 补 z-index（对齐 TutorialOverlay 的 z-10，教程局终局层可能被遮盖）；4) 全站唯一一条 `@media` 防移动端横向溢出。设计总纲：对局页为主、1366×768 下界、深度限定现有结构内调整。

**现状证据（2026-10-25 UI 扫描）：** 裸色值：App.vue:44-51（body 背景 `#10141f` 与 `--cw-color-bg-base #12141A` 不一致）、HpBar.vue:7-8、GameGrid.vue:110/115/123-124/144、EventLog.vue:17-22、EndOverlay.vue:46-58、LoginView/RegisterView/ProfileView、CodeEditor.vue:51、QuestionPanel.vue:259（`#333c52`）；QuestionPanel.vue:245 引用不存在的 `--cw-color-accent-warrior`（靠 fallback 撑着）。px 字号：QuestionPanel:213/218/228/233/235/250、GameGrid:134/144、EventLog:18、HpBar:44、Login/Register。z-index：EndOverlay.vue:45-48 无 vs TutorialOverlay.vue:45 z-10。client/src 内 0 个 `@media`。

**Blocked by:** None（可立即开工；与 23/24 有同文件面，先行可减少后续冲突）

**Status:** ready-for-agent

- [ ] Token 合规：上述文件裸色值全部替换为 `--cw-*` 变量（缺的 token 补进 tokens.css，含 z-index 阶梯）；App.vue body 背景与 `--cw-color-bg-base` 二选一对齐；修正 `--cw-color-accent-warrior` 引用
- [ ] 字号/字体 token 化：px 字号 → `--cw-font-size-*`；正文 font-family 接 `--cw-font-sans`，签名/日志/编辑器语境接 `--cw-font-mono`；题面字号提档 12px → 13px 档
- [ ] EndOverlay z-index 补齐：教程局终局时结算层位于 TutorialOverlay 之上不被遮盖（补断言或人工目检记录）
- [ ] 移动端防横向溢出：`@media (max-width: ~860px)` 下 `.cols` 纵向堆叠、棋盘格 `min()` 缩放、日志跟随；窄窗口无横向滚动条
- [ ] 验收：`npx vitest run` 全绿 + `npx vite build` 通过；diff 审查可证除字号提档外零逻辑/行为改动
