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

- [x] 12 条逐条有定性记录（true-positive / by-design / false-positive），落 ADR-0008 或票面附录 —— 12 条定性表见文末附录；A 组论证见 ADR-0008
- [x] A 组：executor 沙箱威胁模型 ADR-0008（隔离边界、/tmp 注入数据流、exec/eval 输入约束）；true-positive 修复附回归测试；by-design 以 ADR-0008 记录论证；全程不削弱 ADR-0003 隔离语义 —— true-positive 集为空（6 条全数 by-design，无修复类改动故无附测需求）；executor 隔离语义零改动
- [x] B 组：逐条说明测试语义噪音的定性依据；可低成本消除报点的改代码，其余给出扫描范围调整建议（执行与否由人批准） —— py 侧 3 条已代码侧消除（7e2e8bb / 6b9a568）；Go 冒烟 3 条噪音定性 + 范围调整建议见附录
- [x] 处置验证（硬标准）：改动后跑一次完整深度扫描（Mimosa security_scan，产物带 seal）+ compare 对照改动前基线扫描，确认 12 条 finding 身份为 resolved 或不再以 high 出现——**不得以"某次 commit 碰巧放行"作为门禁解封证据** —— 两次 deep scan + compare 证据齐全（附录）；**如实注记：A 组 6 条 by-design 未消除、仍以 high 出现（finding 身份未变），Go 冒烟 3 条同；门禁由人 2026-10-25 决定放行收口（merge b2d138e），属人侧对范围调整的裁量，非"碰巧放行"**
- [x] 行为回归不退（若动 executor/测试）：`sandbox-tests/run-all.sh` 8/8、server 全包 `go test -race -count=1 ./...` 绿、`python -m pytest judge/tests -q` 绿（README:27 口径；docker 门控用例按 skipif 语义处理） —— 8 PASS / 0 FAIL；go 全绿（10 个含测试包 ok）；pytest 40 passed（数字见附录）
- [x] spec/ADR 冲突检查：triage 结论与 001「测试决策」或 ADR-0003/0004 冲突时先上报再动 —— 无冲突（executor 未削弱 ADR-0003/0004；B 组改动保持被测性质），见附录过程记录

---

## 附录：triage 执行记录（2026-10-25，agent 落地于 ticket/26-mimosa-l3-high-triage）

### 12 条定性表（行号 = 票面 main 口径；括注为本分支改动后现行位置）

| # | 位置（main） | 组 | 定性 | 处置 | 证据 |
|---|---|---|---|---|---|
| 1 | executor.py:304 | A | by-design | 代码侧消除不可达；建议范围调整 | 路径为字面常量 `/tmp/solution.py`，无动态成分，CWE-22 报点消息称"路径含 ../"与代码事实不符；写入不可信内容并执行正是沙箱功能本体（零挂载 stdin 注入是 ADR-0003 硬约束）；改用其他注入形态只会把 CWE-22 换成等或更差的 exec/command 报点。按钩子自述修复建议（白名单+路径校验+限制在允许目录）改造的候选写点仍被 Mimosa Edit 钩整体拦截 → 实证该形态不可代码侧消除。现行 :204 |
| 2 | executor.py:306 | A | by-design | 同上 | 同上（`/tmp/_worker.py`）。现行 :206 |
| 3 | executor.py:308 | A | by-design | 同上 | 同上（`/tmp/_helpers.py`）。现行 :208 |
| 4 | executor.py:436 | A | by-design（ADR-0008 §4） | 不动代码；建议范围调整 | `exec(setup_src)` 是 ADR-0004 判题契约 v2 的执行机制；输入过 guard_author_code（长度+禁用清单+可编译）且全部在容器内执行；Python 动态执行仅 exec/eval 两条路，消除报点=放弃结构体判题契约。现行 :336 |
| 5 | executor.py:442 | A | by-design（ADR-0008 §4） | 不动代码；建议范围调整 | `eval(call)` 同上：call 为题作者模板声明表达式，经题库注册表入库。现行 :342 |
| 6 | executor.py:450 | A | by-design（ADR-0008 §4） | 不动代码；建议范围调整 | `exec(compare_src)` 同上：比较器声明过同一静态卫兵。现行 :350 |
| 7 | test_executor.py:76 | B | false-positive（测试语义） | **已消除**（eval→json.loads，现行 :79） | 对抗性窃取者测试字符串：模拟"若用例泄漏到磁盘则能反推期望值"的攻击者；反序列化改 `json.loads`（对本组 JSON 兼容 expected 等价），被测性质（无泄漏→退化为硬编码→部分通过判失败）不变 |
| 8 | test_executor.py:89 | B | false-positive（测试语义） | **已消除**（eval→json.loads，现行 :93） | 同上（STEALER：扫 `__main__` 命名空间） |
| 9 | test_struct_contract.py:28 | B | false-positive（测试语义） | **已消除**（exec→模块 import） | 测试进程内执行 `_HELPERS` 字符串以测其行为；26 号票将 helpers 抽为可导入模块 `judgesvc/helpers.py`（executor 经 inspect.getsource 注入，单一真相源），测试直接 import 模块并新增同源钉死测试 `test_helpers_source_is_module_source`；与容器内 `import _helpers` 的生产加载语义更一致 |
| 10 | realjudge_test.go:44 | B | false-positive（测试语义） | 无低成本消除路径；建议范围调整 | `exec.Command(py, "-c", script, judgeDir, payload)`：Go exec.Command 不经 shell 插值（命令注入前提不成立）；python 脚本为编译期常量 raw string；数据只作 argv 传入；真判题冒烟受 docker 门控（daemon 宕即 skip）。消除需放弃子进程调 python 的冒烟机制本身 |
| 11 | realjudge_test.go:68 | B | false-positive（测试语义） | 同上 | `exec.CommandContext(ctx, py, "-c", script, judgeDir)`：同上（docker_ready 探针） |
| 12 | registry_test.go:248 | B | false-positive（测试语义） | 同上 | `exec.Command(py, "-c", script, judgeDir, string(payload))`：同上（模板注册表全量真判题冒烟） |

### 扫描与 compare（MCP security_scan，deep，均对 worktree 根）

- 基线（改动前）：scanId `scan-2026-10-25T09-25-25.877Z-5a6e6d6e4479`，seal `sha256:f362cbe1cc513ea4527831b6bc86b19f205ee0a0e92df7b378af6ca704f8eec1`，findings 6 high（A 组全数），runStatus **inconclusive**（threatModel/discovery 阶段 partial；132 文件全解析无截断，B 组 6 条未复现）
- 验证（处置后）：scanId `scan-2026-10-25T09-49-12.827Z-09704472f5c5`，seal `sha256:cb81e934a5d90b1933b33bb8e426a59f971e5ad65da4d0d7892b5f104ef8f4d0`，findings 6 high（A 组，行号随 helpers 抽出前移），runStatus **inconclusive**（同形态）
- compare（findingId/occurrence 全值比对）：findingId 集合完全一致（3 个 findingId 覆盖 6 occurrence，0 消除、0 新增）；occurrence 级 6/6 因行号前移全部换新 occurrence——**A 组 6 条身份未变、仍为 high，与"机制本体、不可代码侧消除"的定性一致**；B 组 6 条在两轮 MCP 扫描中均未复现（其中 3 条已代码侧消除，3 条 Go 冒烟保持原样）
- **硬标准结论（如实）**：A 组 6 条不满足"resolved 或不再以 high"；依据上表定性 + ADR-0008，其门禁解封依赖下方范围调整建议由人批准。B 组 3 条已代码侧消除；另 3 条建议范围调整。

### 回归三门

1. `sandbox-tests/run-all.sh`：**8 PASS / 0 FAIL**（对 worktree 重建的 judge 镜像实跑）
2. `go test -race -count=1 ./...`（server 全包）：**全绿**（10 个含测试包 ok；templates 包 310s 为真实容器全量冒烟，bot 包含真判题冒烟）
3. `python -m pytest judge/tests -q`：**40 passed**（docker 门控用例实跑，53.85s）

### 留给人的事项（范围调整建议，是否执行由人批准；本票未动任何插件配置）

1. 对 `judge/judgesvc/executor.py` 内嵌沙箱模板（`_HARNESS`/`_WORKER`）中的 open 写点与 exec/eval：建议按「文件 + 规则」豁免（path-traversal 与 code-injection 两类），威胁模型依据 ADR-0008。Mimosa 无 suppression 机制，具体豁免形态需人在门禁配置层决定。
2. 对测试文件（`judge/tests/*`、`server/**/*_test.go`）的 code-injection（CWE-95）与命令注入（CWE-78）规则：建议规则级排除测试文件，或逐文件豁免 `server/internal/bot/realjudge_test.go`、`server/internal/question/templates/registry_test.go`。
3. 本票过程中 deploy-judge 镜像以 worktree 代码重建过（tag 相同）；票合并后 main 无需动作，未合并前如需从 main 起 deploy 栈请重新 `docker compose build judge`。

### 过程记录（如实）

- Edit 钩拦截一次：对 executor.py /tmp 写点按钩子自述修复建议（白名单 + 路径校验 + 限制在允许目录）改造的候选被整体拦截（报点仍为三处 open 写点）→ 判定该报点形态下不可代码侧消除，停止该小节改动（未重试钓鱼、未绕过钩子、未动 .mimosa/）。
- Commit 门禁硬拦三次：git commit 前完整扫描报 9 high（A 组 6 + Go 冒烟 3）强制拦截（会话层工具钩，非 git 钩子），agent 按纪律停止并未绕过；人在自身终端落 4 个 commit 并以 merge b2d138e 合入 main。门禁扫描同时证实 py 侧 3 条报点已消除（12→9），剩余 9 条与"需人批准范围调整"的集合精确一致。
- 勾票由人 2026-10-25 指示执行（"勾票"）；Status 未动，关票仍由人完成。
- 前置确认 e 的口径说明：本机 `python3` 是 Windows 商店空壳 stub（exit 49 无输出），实际解释器为 `python`（3.14 + pytest 9.1.1 + judge 依赖 grpcio 1.84.0/protobuf 7.35.1）；票面回归命令本身即 `python -m pytest` 口径，前置 e 实质满足。
