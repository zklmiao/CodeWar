# 26: Mimosa L3 十二个 high 存量 finding 的安全 triage（executor 沙箱威胁模型 + 测试噪音定性）

**What to build:** Mimosa L3 完整扫描于 23 号票关票提交时首次跑满并硬拦截 commit，报出 12 个 high——全部为**存量**（早于 19–25 号工作），与被拦的 23 号 client diff 零交集。本票对 12 条逐条 triage 并处置，使安全债务显性化、commit 门禁恢复畅通。分组定调（人已认可）：

**A 组 · 认真对待（6 条，`judge/judgesvc/executor.py`）**：
- `:304/:306/:308` [high] 路径穿越 —— 容器内 /tmp tmpfs 注入写入（解法/worker/helpers 三个文件）
- `:436/:442/:450` [high] 代码注入 —— `exec(setup_src)` / `eval(call)` / `exec(compare_src)`（题作者构造/结算面，ADR-0004 结构契约机制）
- 背景：该服务职责就是在容器内执行不可信代码（ADR-0003 docker-outside-of-docker 拓扑），scanner 报点大概率"按设计如此"，但**威胁模型从未文档化、边界从未复核**。处置 = ADR-0008 威胁模型（容器隔离边界、/tmp 注入面数据流、exec/eval 输入约束）+ 逐条定性：true-positive 修复（附测试，不得削弱 ADR-0003 隔离语义）；by-design 以 ADR-0008 论证记录。
- 注：**Mimosa 无 suppression/忽略配置机制**（scan-contract 只有 finding 状态机与 compare 对照）——处置只能走代码侧消除报点或人侧调整扫描范围，本票不做也不承诺 suppression。

**B 组 · 疑似噪音（6 条，测试文件）**：
- `judge/tests/test_executor.py:76/:89`、`judge/tests/test_struct_contract.py:28` [high] 代码注入 —— 测试字符串内的 `eval(`/`exec(`（对抗性窃取者测试与 helpers 同源执行）
- `server/internal/bot/realjudge_test.go:44/:68`、`server/internal/question/templates/registry_test.go:248` [high] 命令注入 —— 测试里 `exec.Command(py, "-c", script, …)` 调 python
- 处置 = 逐条定性噪音依据；代码侧能低成本消除报点的（如测试字符串构造方式重构）优先改代码；不能的给出扫描范围调整**建议**（是否调门禁范围由人决定，本票不擅改插件配置）。

**行号复核（2026-10-25 已对 main 完成一轮）：** 12 条行号与类别均与 main 当前文件对上，报点内容如上述；执行侧只需复核定性、无需重新定位。

**门禁背景：** Mimosa 钩子当前策略 = 扫描不完整时兼容放行、扫描完整且见 high 时强制拦截。本票落地前，main 侧任何 commit 都可能被这 12 条拦下。

**Blocked by:** None（可立即开工；23 号票的关票/合并等本票解封或人另行授权）

**Status:** ready-for-agent

- [ ] 12 条逐条有定性记录（true-positive / by-design / false-positive），落 ADR-0008 或票面附录
- [ ] A 组：executor 沙箱威胁模型 ADR-0008（隔离边界、/tmp 注入数据流、exec/eval 输入约束）；true-positive 修复附回归测试；by-design 以 ADR-0008 记录论证；全程不削弱 ADR-0003 隔离语义
- [ ] B 组：逐条说明测试语义噪音的定性依据；可低成本消除报点的改代码，其余给出扫描范围调整建议（执行与否由人批准）
- [ ] 处置验证（硬标准）：改动后跑一次完整深度扫描（Mimosa security_scan，产物带 seal）+ compare 对照改动前基线扫描，确认 12 条 finding 身份为 resolved 或不再以 high 出现——**不得以"某次 commit 碰巧放行"作为门禁解封证据**
- [ ] 行为回归不退（若动 executor/测试）：`sandbox-tests/run-all.sh` 8/8、server 全包 `go test -race -count=1 ./...` 绿、`python -m pytest judge/tests -q` 绿（README:27 口径；docker 门控用例按 skipif 语义处理）
- [ ] spec/ADR 冲突检查：triage 结论与 001「测试决策」或 ADR-0003/0004 冲突时先上报再动
