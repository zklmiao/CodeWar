# 优化审查报告与任务列表（2026-10-27）

> 来源：29 号票后的 main（`161d4f9`）三轴审查（功能 / UI / 工程质量）。本文件作为任务列表使用：条目状态直接在本文维护，立项后以 `.scratch/codewar-mvp/issues/` 正式票据为准并在此标注票号。未关票现状（08 / 11 / 25）已排除，以下建议均为其外增量。当前波次实态（已关票 / frontier / 挂起一览）见 [README「当前波次」](../../README.md#当前波次)（30 号票建立的互链）。

## 一、功能优化建议

| 优先级 | 建议 | 依据与体量 |
|---|---|---|
| P1 | **对局回放/复盘**：引擎事件流（Envelope+seq）天然可持久化，但 `db/migrations/000001` 重写时删掉了 `match_events` 占位表，对局结束即蒸发。落库逐回合事件 + 回放页（复用 match store 渲染），竞技游戏的复盘是留存核心；同时支撑错题复盘 | M-L |
| P1 | **错题本**：judged 失败的提交（类别+题目 id）按用户落库，PvE 模式加「错题重练」入口（QuestionFactory seed 确定性已支持复现同题）。学习型游戏最贴合的闭环 | M |
| P1 | **投降/认输**：现无任何提前结束路径，劣势方只能等 10 回合打满或超时。hub 加一条 resign 消息 + 引擎终局分支，**08 号票开工前补上成本最低**（协议定稿前进 proto） | S |
| P2 | **排行榜页**：`users.elo`/`elo_history` 数据齐全，只缺一个 `/leaderboard` 页。08 落地后价值翻倍 | S |
| P2 | **结算页统计**：终局只显示胜负+ELO，可加回合数、双方命中率、平均解题耗时、伤害总量（事件流现成可算） | S |
| P2 | **spec 开放问题 5（抽题策略 A/B）已具备落地条件**：每局已存 strategy 配置与胜负，加一张聚合统计即可验证「爬坡 vs 均匀」默认值 | S |
| P3 | 音效（命中/治疗/闪避/最后 10 秒滴答，WebAudio 零依赖）； PvE 连胜挑战关卡； 观战模式（需中性快照，L，等 08 后再说） | S/L |

## 二、UI/UX 优化建议

**先报一个真 bug**：`QuestionPanel.vue:129` 的 `setInterval(..., 100)` **没有 `onBeforeUnmount` 清理**（对照 `RoundStatusBar.vue:15` 有清）——离开对局页后心跳常驻，属于内存泄漏，一行修复。

| 优先级 | 建议 | 锚点 |
|---|---|---|
| P1 | 导航栏泄漏开发注释：标题旁「07 号票 · 账号与 ELO」是票号标注，应删或改为真正的产品标语 | App.vue:34 |
| P1 | **键盘动线**：竞速游戏时间系数 0.6~1.0，鼠标点卡→点提交全程无键盘支持。加 1-4 切卡、Ctrl+Enter 提交、方向键选向（QuestionPanel 零 keydown 处理） | QuestionPanel.vue 全文 |
| P1 | EventLog 纯文本无类型着色——伤害红/治疗绿/闪避黄/系统暗，一眼读战况；每行已有 seq 结构，加个 kind class 即可 | EventLog.vue:9-12 |
| P2 | **技能/攻击形状预览**：选定方向+距离后在棋盘上高亮覆盖格——27 号票后形状语义已下发客户端，纯前端可算，走位博弈的教与玩都靠它 | GameGrid.vue + QuestionPanel aimShape |
| P2 | 战斗反馈缺失：被击中/闪避/治疗时棋盘无任何瞬时反馈（飘字、格子闪光、开火线）；EventLog 记了但棋盘是哑的 | GameGrid.vue:70-102 |
| P2 | 入口页层级扁平：配置→职业→PvE→接入卡四段纵排无视觉分组；职业卡可复用 `design/components/cw-class-*.svg` 剪影（GameGrid 已在用）并展示技能形状示意 | MatchView.vue:224-337 |
| P2 | 结算仪式感：EndOverlay 是一张静态卡片，「刷新两个窗口再来一局」的提示在 08 后必须变成真正的「再来一局」按钮 | EndOverlay.vue:39 |
| P2 | a11y：题卡激活态无 `aria-pressed`；棋盘格不可键盘聚焦无 `aria-label`；EventLog 非 `aria-live` | QuestionPanel.vue:181-203、GameGrid.vue:73-83 |
| P3 | HP 条无浮动伤害数字（-26 飘字）、低血量警示态；棋盘 legend 文案「P1（你=是）· P2（你=否）」生硬；CodeMirror 未接霓虹主题；触屏端方向按钮热区偏小 | HpBar.vue、GameGrid.vue:97 |

## 三、工程质量（审查代理结论）

最重的三条：**CI 没有 Postgres service**——`CODEWAR_TEST_DSN` 门控的 store/hub/accounts 集成测试在 CI 从未真正跑过（ci.yml 无 `services:`），质量安全网名存实亡；**client 无类型检查门**（build 仅 `vite build`，无 vue-tsc，TS 错误永不阻塞合入）+ 全仓零 lint；**路由不懒加载**——CodeMirror 被四个页面全量背上首屏包（router.ts:3-15 全静态 import）。

其余：CI 无缓存/超时/并发去重（S）；hub 连接无 deadline/ping/连接上限，死链永久占位（hub.go:459-466，可作 08 的前置小票，M）；compose 无 healthcheck 接线（app 明明有 `/healthz`，S）；judge `requirements.txt` 全 `>=` 不 pin 版本（安全敏感，S）；`judgeclient` 包 41 行零测试；README「当前波次」还停在 Wave 1（S，半小时的事）。

## 四、推荐开票组合

1. **30 号票（S，先做）**：杂项清理——README 波次刷新、App.vue 票号标注、QuestionPanel 定时器泄漏、EventLog 着色、结算页统计。一票打包五个小件，半天量。
2. **31 号票（S，08 前置）**：投降认输 + hub 连接卫生（deadline/ping/上限），协议面趁 proto 定稿前一并处理。
3. **32 号票（M）**：CI 门控补全——Postgres service + DSN、vue-tsc/vet/lint、缓存超时。
4. **33 号票（M-L，08 后）**：对局回放 + 错题本（事件落库 schema 是共同地基，可拆两步）。
5. **34 号票（M）**：对局可玩性 UI 包——键盘动线、形状预览高亮、战斗反馈飘字。

---

## 任务勾选区（销项用）

- [x] 30 号票：杂项清理（README 波次 / App.vue 标注 / 定时器泄漏 / EventLog 着色 / 结算页统计）——已关票（merge 49204dd）
- [x] 31 号票：投降认输 + hub 连接卫生（08 前置）——已关票
- [ ] 32 号票：CI 门控补全（Postgres / 类型检查与 lint / 缓存超时）
- [ ] 33 号票：对局回放 + 错题本（08 后）
- [ ] 34 号票：对局可玩性 UI 包（键盘动线 / 形状预览 / 战斗反馈）
- [ ] 未入票的散项按需逐条回看上文三张表（P3 项与 P2 中未打包项：排行榜页、抽题策略 A/B、音效、入口页分组、a11y、HP 飘字等）
