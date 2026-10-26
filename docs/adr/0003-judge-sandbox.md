# ADR-0003：判题沙箱机制（一次性容器，docker-out-of-docker）

**状态**：已接受（02 号票；27 号票修订——worker 协议结论行规则收紧为"恰好一行"，沙箱命令描述同步至现行零挂载形态）
**关联**：ADR-0001（judge 为独立 Python 微服务，gRPC 同步 + 每用例 deadline）；设计 002 §5（判题管线流程与契约）、§6（遥测点）；票据 02

## 背景

01 号票的简易执行器与玩家代码共享文件系统与网络，仅靠整体超时保护——不可信代码可读宿主文件、对外联网、耗尽内存拖垮判题服务。本票要在 judge 服务边界内换成 OS 级隔离，且判题仍须在回合内同步返回（75 秒回合，单次提交预算 ≤ 5 秒）。

约束：开发机 Windows + Docker Desktop（WSL2 后端）；CI ubuntu runner 自带 docker；判题失败不得依赖运行时拉镜像（离线可判）。

## 决策

**每次提交启动一个一次性容器（docker-out-of-docker），容器内 harness 逐用例执行并施加每用例超时。**

judge 容器挂载宿主 `/var/run/docker.sock`，判题时执行（要点逐条对应）：

```txt
docker run --rm -i
  --network=none                 # 容器级断网
  --memory=256m                  # 内存上限
  --pids-limit=64                # 进程数上限（防 fork 炸弹）
  --cpus=0.5                     # CPU 配额（防占满宿主）
  --cap-drop=ALL                 # 丢弃全部 capability
  --read-only                    # 根文件系统只读
  --tmpfs /tmp:size=16m,exec     # 唯一可写区（小而可执行，供解释器展开用）
  -e PYTHONDONTWRITEBYTECODE=1   # 配合只读根文件系统（禁 .pyc 落盘）
  python:3.12-slim python -c <harness> <per-test-ms>
```

**零挂载、stdin 注入**（现行形态，以 `build_sandbox_cmd` 为准）：命令中不存在任何
`-v`/`--mount`——解法源码、harness/worker/构造辅助库脚本与用例清单全部经 stdin
注入（harness 经 `python -c` 加载，stdin 完整留给载荷），运行期文件只写容器内
/tmp tmpfs。原因：docker-out-of-docker 下 `-v` 的宿主路径由外层 daemon 解析，
judge 容器内的临时目录在宿主上不存在，挂载方案在嵌套形态下必然失败（浏览器
验收发现的 P1，后文"备选/后果"中的挂载路径讨论随之作废）。

- **期望值不进容器（比对在宿主）**：容器 stdin 只接收载荷（解法源码 + harness/worker/构造辅助库脚本 + call 表达式清单 `[{name, call, setup, compare}]`，零挂载不经网络；测试输入天然对执行进程可见，可接受）；worker 逐用例执行并逐行输出 `{i, status, repr/err}`（`repr` 仅正常返回时有）；**repr 与 expected 的字符串相等判断、category 汇总全部在宿主 `_assemble_verdict` 完成**。玩家代码即使读遍 `__main__` 命名空间也见不到任何期望值（泄漏断言见 judge 测试）。
- **容器粒度 = 每次提交一个容器**（不是每用例一个）：容器启动约 300–800ms，每用例一个容器会把 5–8 个隐藏用例的开销放大数倍；容器内 harness 用子进程逐用例执行，首败即停（任何非 ok 状态立即收尾，调用期异常也停——抛异常即没算出值，不可能正确）。worker 另设 `RLIMIT_AS=256m`（大分配转 MemoryError）纵深于容器级 `--memory` 之内；fork 炸弹由容器级 `--pids-limit` 转 EAGAIN（errno 11，harness 按 errno 归因——root 用户不受 NPROC 约束，NPROC 设置仅作将来 `--user` 运行的纵深；27 号票注记：fork 的子进程继承管道写端并各自 emit 结论行，故 fork 炸弹实际先触发下述"恰好一行"规则归 runtime-error，同为惩罚路径，errno 归因保留给无 fork 污染的配额路径）。worker 协议行以换行为界，**每个用例的 worker stdout 中带 `outcome` 键的结论行必须恰好一行**（27 号票修订）：worker 在 emit 结论后立即 `os._exit(0)` 硬退，跳过 atexit 与解释器收尾——玩家代码无法在结论之后经收尾钩子补写协议行（已复现的 atexit 伪造向量：错误答案被洗成免罚的 judge-error）；harness 对 0 行或多于 1 行一律判 runtime-error 走惩罚路径（fail-closed：协议流被污染即不可信，绝不采信其中任何一行，也绝不落入免罚的 judge-error），防多进程写污染。
- **每用例超时 = `PER_TEST_TIMEOUT_MS`（默认 3000ms，judge 侧常量）**：请求携带的 `timeout_ms` 若为正且更小则采用（兼容旧调用与测试的小超时）；harness 对 worker 施加 `subprocess` 超时并判 `timeout` 首败即停，恶意循环最多消耗一个 per-test 配额。
- **容器整体墙钟上限 = 用例数 × per-test + 30s**：宿主 `Popen.communicate` 超时施加，为第二道保险。熔断时先 kill 再兜底回收已 flush 的 stdout 行（`_DRAIN_TIMEOUT_S`），**已完成的用例按其结论计入汇总，未完成的判 timeout**——harness 每行 flush 且正常结束必带 summary 行，此承诺由实现保证（首败即停下外层熔断几乎不可达）。
- **延迟预算**：单次提交 ≤ 5 秒（容器启动 + 首败即停的用例执行），75 秒回合内可接受；全过路径每个用例毫秒级，总耗时主要由容器启动构成。
- **`duration_ms` = harness summary 行实测的用例总墙钟**（容器内测量）；仅当容器被外层熔断、无 summary 行时退化为宿主实测墙钟。与玩家解题耗时（引擎 ClockPort 计时、驱动时间系数）是两个量——沙箱侧不做任何换算，只如实上报（遥测见设计 002 §6）。

- **失败反馈契约（设计 002 §3 收紧）**：verdict 只含 `{passed, category, duration_ms}` 三字段；
  `category ∈ ok / wrong-answer / runtime-error / timeout / limit-exceeded / syntax / judge-error`；
  `detail` 只允许语法错误行号这类无害信息（`line N: <msg>`），其余类别一律空字符串——禁止携带任何可推断隐藏用例通过情况的内容（含通过计数、stderr、actual/expected）。
  新增 `limit-exceeded`（内存/pids 超限）与 `timeout` 区分：前者是配额杀（OOM 137、MemoryError、无法 fork），后者是时钟杀。
- **镜像固定 `python:3.12-slim`**：judge 的 `Dockerfile.judge` 加装 docker-cli；构建期无 daemon 可用，故"预拉取"落在 judge 服务启动时——启动即 `docker image inspect`，缺失则 best-effort `docker pull`，离线且已缓存则直接判题（离线可判）；拉取失败且无缓存才退化为 `judge-error`（fail-closed，绝不回退到无隔离执行）。
- **fail-closed**：docker 不可用（无 CLI、无 daemon、无镜像）一律返回 `judge-error`，引擎侧已对该类别不罚不锁（engine.go Submit），对局不受牵连。

## 备选方案（被否原因）

- **nsjail / 用户命名空间沙箱**：默认 Docker seccomp profile 下嵌套 user namespace 与相关 mount 操作不可靠——judge 自身就跑在容器里（compose 部署），嵌套命名空间路径在默认 seccomp 下行为不稳定，调试与可移植成本高于容器方案。
- **纯 rlimits 子进程（setrlimit + subprocess）**：只能限 CPU/内存/进程数，**无法断网**（需网络命名空间，等于重做半个容器运行时）；且与玩家代码同内核同文件系统，隔离强度低于容器，不满足"公网承载不可信代码"的安全目标。
- **每用例一个容器**：隔离最彻底，但 5–8 个隐藏用例 × 每次 300–800ms 启动开销，延迟预算超标；容器内子进程级隔离已足够区分超时/超限归因。

## 后果

- **(+)** 四类作弊向量全部在真实隔离下被拒：死循环→timeout、联网→Network is unreachable、内存/进程炸弹→limit-exceeded、部分通过→失败且不泄漏计数；恶意提交后服务存活（容器 `--rm` 即抛）。
- **(+)** 主服务零改动：verdict 仍是字符串 category，引擎对非 `judge-error` 失败一律自损+锁（含新增 `limit-exceeded`，fail-closed 到惩罚路径）。
- **(−)** 判题依赖宿主 docker.sock（compose 挂载；特权面扩大但仅限 judge 服务），Windows 开发机上挂载路径为 `os.path.abspath` 后的字面 Windows 路径（仅相对转绝对，不做盘符/分隔符转换），由 Docker Desktop 直接接受（本票已实测可用）。
- **(−)** 每个容器化测试增加 1–3 秒开销，容器化测试数量克制；CI 可跑（ubuntu 自带 docker），无 docker 环境下容器化测试 skip（`judge-error` 不伪装成通过）。

## 已知递延（归属明确，防止静默丢失）

- 静态卫兵完整版（长度上限 / import 白名单 / 禁用模式，设计 002 §5 步骤 ①）→ 归 **11 号票**（反作弊；已挂入其验收清单）
- 遥测扩展（内存峰值 / 异常类型分布上报，设计 002 §6）→ 归 **11 号票**（反作弊；已挂入其验收清单）
- judge 容器非 root 用户（当前容器内 root + 挂载 docker.sock ≈ 宿主 root，本 ADR 明示接受该风险）→ 归 **08 号票开票时的部署加固**（在线化前收紧）
