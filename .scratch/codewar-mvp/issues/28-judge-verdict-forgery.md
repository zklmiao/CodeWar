# 28: 判题结论可被玩家代码伪造——atexit 伪造协议行绕过错误惩罚

**What to build:** `judge/judgesvc/executor.py` 容器内 harness 取 worker 输出中**最后一行**带 `outcome` 的 JSON 当作结论（`_HARNESS` 里 `for v in reversed(_parse_json_lines(proc.stdout))`）。玩家代码可以注册 `atexit`，在 worker 的 `emit(result)` 之后再往 `sys.__stdout__` 写一行伪造协议，把任意错误答案变成 `judge-error`。引擎对 `judge-error` 不罚不锁（`server/internal/match/engine.go` 的 `Submit`），因此"答错自损 3 HP + 锁 5 秒"（spec 用户故事 14）被完全绕过，玩家还能借此无代价地探测隐藏用例输入。缺陷已复现（错误答案 + atexit 伪造 → judge-error）。

修复要求（fail-closed）：
1. `_WORKER`：`emit(result)` 之后立即 `os._exit(0)`，跳过 atexit 和解释器收尾。
2. `_HARNESS`：每个用例的 worker stdout 中，带 `outcome` 键的 JSON 行必须**恰好一行**；0 行或多于 1 行一律判 `runtime-error`（fail-closed，走惩罚路径），不能判 `judge-error`。
3. 不改变正常路径的行为：玩家 `print` 仍截留在内存里；结构体题的 setup/compare 契约（ADR-0004）不变。
4. 更新 ADR-0003 中描述"真值行恒为末行"的段落，改为新规则；顺带把文中已过时的 `-v <代码目录>:/sandbox:ro` 挂载描述改为现行的"零挂载、stdin 注入"（以 `build_sandbox_cmd` 为准）。

范围外（待人裁决，本票不实现）：
- 玩家可以返回一个方法会抛异常的对象，让 compare 的 `_normalize` 崩溃；或定义同名函数，让 setup 崩溃（worker 用 `NS.setdefault`，玩家定义优先）。这两种情况现在会归为 `infra-error`，即同样免罚。修复它与 ADR-0004"setup/compare 失败 = 基础设施故障，不罚玩家"冲突，需要人裁决。
- 引擎侧是否限制连续 `judge-error` 的次数（属于新规则）。

**Blocked by:** None

**Status:** ready-for-agent

- [x] atexit 伪造回归：错误答案 + atexit 伪造 outcome 行 → `wrong-answer`（不得为 judge-error/ok）
- [x] import 期向 `sys.__stdout__` 写伪造 outcome 行 → `runtime-error`（不得为 judge-error/ok）
- [x] `os.fork()` 子进程 sleep 后写伪造行 → `runtime-error` 或 `timeout`（不得为 judge-error/ok）
- [x] 对照组：正确答案 → `ok`；普通错误答案 → `wrong-answer`；玩家 `print` 大量输出时正确答案仍为 `ok`
- [x] ADR-0003 更新落地（"真值行恒为末行"段落改新规则；挂载描述改"零挂载、stdin 注入"）
- [x] `python -m pytest judge/tests -q` 全绿（docker 门控按 skipif 语义处理）
- [x] `cd server; go test ./internal/question/templates/ -count=1` 全模板参考解冒烟绿（偶发 judge-error 如实记录重跑结果，不掩盖）

> **改号记录（2026-10-26，人裁决）**：本票开票时按提示词显式路径编号 27（`27-judge-verdict-forgery.md`），与已关票 `27-skill-shape-tests-and-semantics.md` 撞号，违反 issue-tracker「一票一号」惯例；经人裁决改号为 **28**（对齐提示词原标号）。修复提交 `e721108` 标题中的 "ticket 27" 字样不可改，以本记录为准；ADR-0003 / executor.py / test_executor.py 中的旧号自引用已随改号同步修订为 28。

> **交付记录（2026-10-26，勾票）**：`ticket/27-judge-verdict-forgery` 分支提交 `e721108`（worker `emit` 后 `os._exit(0)` 硬退 + harness「每个用例恰好一行结论行」fail-closed 规则 + ADR-0003 三处更新）与 `75cb809`（自引用改号）。四条新测试先写后修、修复前实测：atexit 伪造 → judge-error（免罚，缺陷本体）；import 期伪造 → ok（虚假通过，比免罚更危险）；fork 子进程伪造 → wrong-answer 断言失败；print 截留对照组绿。修复后四条全绿（真实容器）。回归：`python -m pytest judge/tests -q` 44 passed（30.51s，docker 实跑；既有 fork-bomb 归因修订经人裁决接受——子进程继承管道写端各自 emit 结论行，「恰好一行」规则必先触发，同为惩罚路径）；全模板参考解冒烟 `go test ./internal/question/templates/ -count=1` ok 172.554s 一次通过、无偶发 judge-error。范围外维持：compare/setup 崩溃免罚向量与 judge-error 连续次数限制，另票收编。
