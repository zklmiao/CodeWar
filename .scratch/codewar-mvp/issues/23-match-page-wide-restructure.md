# 23: 对局页页宽放宽 + 左栏（棋盘+日志）重排

**What to build:** UI/UX 优化第二级结构票之一（a/d 档地基）。对局 viewer 形态冲出 App 壳 960px 限宽（1366 屏两侧各空 ~200px），放宽到 ~1240；对局页重排为左栏 = GameGrid + EventLog（日志从底部整行移入左栏下方、固定 150px 改 flex 弹性），右栏整体留给题区（24 号票的题卡+主工作区落位处）。HpBar 维持顶部不动；入口/登录/注册/个人页保持 960 不受影响。

**现状证据（2026-10-25 UI 扫描）：** App.vue:45 `.page` max-width:960px；MatchView.vue:203-211 `.cols`（左 GameGrid 固定 ~320px、右 `.right-col` flex:1）+ EventLog 整行在 `.cols` 之下；EventLog.vue:17 固定高 150px；GameGrid.vue:107-109 格 56px。

**Blocked by:** None（硬依赖无；建议在 22 之后串行，减少同文件冲突）

**Status:** ready-for-agent

- [ ] `.page` 增加 `--wide` 变体：对局 viewer 形态 ~1240px（1366 屏右栏 ~590px → ~860px），其余路由保持 960
- [ ] 对局页重排：左栏 = GameGrid 上、EventLog 下（flex 弹性高度，去固定 150px）；右栏 = 题区容器；HpBar 顶部不变
- [ ] EventLog 内部结构不动（倒序/seq 列保持），既有测试选择器零迁移全绿
- [ ] 1366×768：左栏完整呈现棋盘与日志区头部（不裁切）；1366 以上宽度增量主要给右栏
- [ ] 验收：`npx vitest run` 全绿 + `npx vite build` 通过
