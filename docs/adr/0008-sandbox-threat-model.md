# ADR-0008：判题沙箱威胁模型——隔离边界、/tmp 注入面与 exec/eval 输入约束

**状态**：已接受（26 号票）
**关联**：ADR-0003（一次性容器拓扑）、ADR-0004（结构契约 setup/compare）、设计 002 §5；本 ADR 是对 Mimosa L3 扫描在 `judge/judgesvc/executor.py` 报出的 6 条 high（3×CWE-95 代码注入 `:436/:442/:450`、3×CWE-22 路径穿越 `:304/:306/:308`）的定性依据与威胁模型记录。

## 背景

26 号票对存量 12 条 Mimosa high 做 triage 时发现：沙箱判题的**威胁模型从未被文档化**。ADR-0003 记录了"怎么做隔离"，ADR-0004 记录了"题作者代码进容器的信任边界变化"，但两者都没有站在攻击者视角回答：报点位置的输入是谁控制的、逃逸要突破什么、为什么这些形态是判题机制的本体而非缺陷。本 ADR 补齐该记录；定性结论（true-positive / by-design / false-positive）以它为引用锚点。

## 决策

### 1. 主体与信任分级

| 主体 | 信任级 | 说明 |
|---|---|---|
| 宿主 judge 服务（judgesvc） | 可信 | 自有代码；harness/worker/helpers 源码经 stdin 注入容器 |
| 题作者 setup/compare | 半可信 | 题库入库面；注入前过 `guard_author_code`（长度 4000、禁用清单 `__import__/open/os.system/subprocess/shutil/importlib/eval/exec`、必须可编译），运行期崩溃归 infra（judge-error，不罚玩家）——ADR-0004 |
| 玩家提交代码 | 不可信 | 判题对象本身；它进沙箱就是为了被执行，隔离由容器承担 |
| 期望值 expected | 秘密 | 一分钟都不进容器；比对在宿主（`_assemble_verdict`），worker 命名空间里不存在 |

### 2. 隔离边界（攻击者必须突破的层，全部来自 ADR-0003）

容器级：`--network=none`（断网）+ `--memory=256m` + `--pids-limit=64` + `--cpus=0.5`（配额）+ `--cap-drop=ALL` + `--read-only`（只读根）+ `--rm`（一次性，即抛）；唯一可写区是 `--tmpfs /tmp:size=16m`。进程级：worker 另设 `RLIMIT_AS=256m`；每用例 `subprocess` 超时（首败即停）；容器整体墙钟熔断（宿主施加，第二道保险）。服务级：docker 不可用一律 fail-closed 返回 judge-error，绝不回退无隔离执行。

**攻击面结论**：容器内代码（无论玩家还是题作者声明）的任何"恶意成功"都必须以突破上述容器隔离为前提。逃逸容器 = ADR-0003 红线被击穿，是宿主 docker 内核隔离层的失效，而非 executor.py 注入面的问题；该风险（含容器内 root + docker.sock 挂载）已在 ADR-0003「已知递延」明示，归 08 号票部署加固。

### 3. /tmp 注入面数据流（:304/:306/:308 的定性依据）

数据流：宿主 `build_stdin_payload`（solution/worker/helpers + calls，**不含 expected**）→ stdin 进容器 → harness 把三个文件写到 `/tmp` 固定文件名 → worker 以子进程方式执行 worker 脚本、以 `import solution`/`import _helpers` 加载另两份。

逐点澄清：

- **路径无动态成分**：三个写点路径均为字面常量（`/tmp/solution.py`、`/tmp/_worker.py`、`/tmp/_helpers.py`），不存在用户可影响的路径拼接，CWE-22（路径穿越）的前提不成立——基线扫描报点消息称"未校验的路径含 `../`"，与代码事实不符（字面路径不含 `..`）。
- **"写入不可信内容并执行"是沙箱的功能本体**：把玩家代码落盘供 import、把 worker 落盘供 spawn，正是"不可信代码在隔离环境内被执行"这一判题职责的实现；不可信内容逃不出第 2 节的容器边界。
- **零挂载是硬约束**：docker-out-of-docker 下任何 `-v` 挂载都解析不了宿主路径（ADR-0003 浏览器验收发现的 P1），stdin 注入 + 容器内 /tmp 落盘是唯一可行注入通道，不是可选实现细节。
- **收紧（26 号票落地）**：harness 写点收敛到 `inject_target(name)`——文件名白名单常量 + `os.path.join("/tmp", name)` 解析结果必须恰为 `/tmp/<name>`，白名单外或解析越界即抛错 → 容器无输出 → 宿主判 judge-error（fail-closed）。该约束对现有调用是恒真的纵深防御，防未来重构意外引入动态文件名；同时构成对 CWE-22 规则的可审计回答（写点被白名单+解析双重校验）。
- **tmpfs exec 开启是解释器需要**（ADR-0003 原文"小而可执行，供解释器展开用"），16m 上限 + 只读根把它限制在容器内一次性区域。

### 4. exec/eval 输入约束（:436/:442/:450 的定性依据）

worker 内 `exec(setup_src, NS)` / `eval(call, dict(NS))` / `exec(compare_src, cns)` 是 ADR-0004 判题契约 v2（"题侧声明，沙箱执行，宿主比对"）的执行机制：setup 为 call 绑定结构体入参，call 在玩家命名空间求值，compare 把 got 规范化为 repr。三条输入的约束链：

1. **setup/compare**：过 `guard_author_code`（见第 1 节半可信级）；违规 judge-error、不注入容器（`run()` 入口先验）。
2. **call**：题作者模板声明的 Python 表达式，经题库注册表入库（设计 002 §1），与 setup/compare 同属题作者信任面。
3. **执行位置**：三者全部在沙箱容器内执行；即使输入被攻破（题库被污染），后果被限制在第 2 节容器边界内——这正是"真实边界是沙箱本身"（ADR-0004 原文）的含义。

**为什么不可代码侧消除**：判题契约要求动态执行题作者声明与玩家命名空间表达式求值，Python 动态执行仅有 `exec`/`eval` 两条路；`importlib`/`compile` 等替代形态要么最终仍落到 exec 语义、要么表达力不足（ADR-0004 否决项 2/4 同理）。消除报点的唯一途径是放弃结构体判题契约本身——不成立。定性：**by-design**，scanner 报点在"沙箱执行器"语境下为预期误报，门禁层面的消除需人侧扫描范围调整（见票面附录建议）。

### 5. 残余风险清单（明示接受，不构成"项目安全"结论）

- 容器内 root + 挂载 docker.sock ≈ 宿主 root：ADR-0003 已明示接受，08 号票收紧。
- `guard_author_code` 是黑名单，理论上存在绕过面：靠容器运行时约束兜底（ADR-0004 (−)）。
- 玩家代码在容器内可见 call 表达式与用例名（测试输入天然对执行进程可见，ADR-0003 已接受）；expected 不可见有泄漏断言测试钉死。
- 本 ADR 只记录威胁模型与定性依据，不宣称"项目安全"；scanner 是否继续报点属门禁策略问题，由人决定（Mimosa 无 suppression 机制）。

## 后果

- **(+)** 12 条存量报点中的 A 组 6 条有了可引用的定性锚点；后续扫描报同位置时按本 ADR 处置，不再逐次重新论证。
- **(+)** `/tmp` 写点获得白名单 + 路径解析双校验（fail-closed），威胁模型的"注入面"清单化。
- **(−)** scanner 对 executor.py 的 exec/eval 报点将持续存在（机制不可消除且无 suppression 机制）——门禁解封依赖人侧范围调整，agent 侧不得以重构为名做检测规避（如动态拼接 `exec` 标识符）。
- **(−)** harness 模板新增 `inject_target`：注入路径逻辑多一层间接（对现有行为恒真，测试全量覆盖）。
