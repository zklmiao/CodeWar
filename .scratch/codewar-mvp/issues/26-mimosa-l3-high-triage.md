# 26: Mimosa L3 十二个 high 存量 finding 的安全 triage（executor 沙箱威胁模型 + 测试噪音定性）

**What to build:** Mimosa L3 完整扫描于 23 号票关票提交时首次跑满并硬拦截 commit，报出 12 个 high——全部为**存量**（早于 19–25 号工作），与被拦的 23 号 client diff 零交集。本票对 12 条逐条 triage 并处置，使安全债务显性化、commit 门禁恢复畅通。分组定调（人已认可）：

**A 组 · 认真对待（6 条，`judge/judgesvc/executor.py`）**：
- `:304/:306/:308` [high] 路径穿越 —— 沙箱执行器的路径构造点
- `:436/:442/:450` [high] 代码注入 —— 执行器的代码执行点
- 背景：该服务职责就是在容器内执行不可信代码（ADR-0003 docker-outside-of-docker 拓扑），scanner 报点大概率是"按设计如此"，但**威胁模型从未文档化、边界从未复核**。处置 = 威胁模型 ADR 化（容器隔离边界、路径输入面与数据流、代码注入面的输入约束）+ 逐条定性：true-positive 修复（附测试）；by-design 以 ADR 论证并 suppression 记录。

**B 组 · 疑似噪音（6 条，测试文件）**：
- `judge/tests/test_executor.py:76/:89`、`judge/tests/test_struct_contract.py:28` [high] 代码注入
- `server/internal/bot/realjudge_test.go:44/:68`、`server/internal/question/templates/registry_test.go:248` [high] 命令注入
- 背景：测试本就在构造/执行 python/go 代码，属 SAST 对判题类项目的典型报点。处置 = 逐条定性 false-positive/acceptable，产出 suppression 或扫描范围调整**建议**——是否调门禁范围由人决定，本票不擅改插件配置。

**门禁背景：** Mimosa 钩子当前策略 = 扫描不完整时兼容放行、扫描完整且见 high 时强制拦截。本票落地前，main 侧任何 commit 都可能被这 12 条拦下（本票 docs 提交本身也是）。

**Blocked by:** None（可立即开工；23 号票的关票/合并等本票解封或人另行授权）

**Status:** ready-for-agent

- [ ] 12 条逐条有定性记录（true-positive / by-design / false-positive），落 ADR 或票面附录（报点行号以 main 当前文件为准复核，原始扫描出自 23 worktree 的同内容副本）
- [ ] A 组：executor 沙箱威胁模型 ADR（隔离边界、路径数据流、注入面输入约束）；true-positive 修复附回归测试，by-design 以 ADR + suppression 记录
- [ ] B 组：逐条说明测试语义噪音的定性依据；给出 suppression / 扫描范围调整建议（执行与否由人批准）
- [ ] 处置落地后：main 的 commit 门禁不再被这 12 条拦截（修复生效或经人认可的 suppression/范围调整生效）
- [ ] 行为回归不退（若动 executor/测试）：`sandbox-tests/run-all.sh` 8/8、server 全包 `go test -race -count=1 ./...` 绿、judge 侧测试绿
- [ ] spec/ADR 冲突检查：triage 结论与 001「测试决策」或 ADR-0003 冲突时先上报再动
