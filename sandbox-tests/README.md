# 判题沙箱人工验收脚本（02 号票存档）

对 judge 容器（`deploy-judge-1`）执行 `docker exec`，走 **docker-out-of-docker 真实链路**（judge → 宿主 daemon → 一次性沙箱容器）逐项验证判题契约。与 `judge/tests/test_executor.py` 的自动化用例互补：这套脚本在**部署形态**下人工执行。

## 前置条件

1. Git Bash（PowerShell 引号规则不同，勿直接粘贴）
2. docker 可用，且栈已按 02 分支代码启动：`docker compose -f deploy/docker-compose.yml up -d --build`（judge 镜像必须含 docker-cli，见 `deploy/Dockerfile.judge`）
3. judge 容器名 `deploy-judge-1`（compose 项目名 `deploy`；若改过项目名，批量替换脚本里的容器名）

## 用法

```bash
bash 01-ok.sh            # 单项执行，人工判读 verdict
./run-all.sh             # 一键跑全部，自动断言输出 PASS/FAIL 汇总
```

## 预期结果表

| # | 脚本 | 场景 | 预期 verdict |
|---|---|---|---|
| 01 | 01-ok.sh | 正确代码两用例全过 | `passed: True, category: ok` |
| 02 | 02-deadloop-timeout.sh | `while True: pass` 死循环 | `passed: False, category: timeout, duration_ms ≈ 3000` |
| 03 | 03-network-blocked.sh | `socket().connect()` 联网尝试 | `passed: False`（`--network=none` 断网；category 落在 wrong-answer/runtime-error 均符合契约） |
| 04 | 04-memory-limit.sh | 300MB 字符串分配（RLIMIT_AS 256MB） | `passed: False, category: limit-exceeded` |
| 05 | 05-partial-pass-no-leak.sh | 两用例一过一错（期望值故意错） | `passed: False, category: wrong-answer, detail: ""`（**不得出现通过计数或用例信息**） |
| 06 | 06-syntax-detail.sh | `s === s` 语法错误 | `passed: False, category: syntax, detail: "line 2: ..."`（唯一允许带 detail 的类别） |
| 07 | 07-service-alive.sh | 连发两轮死循环后立刻正常判题 | 第二段输出仍为 `ok`（服务不垮、判题无污染） |
| 08 | 08-fork-bomb-limit.sh | 循环 `os.fork()` 10 万次（进程炸弹） | `passed: False, category: limit-exceeded`（容器级 `pids-limit=64` 配额杀，宿主受保护不垮） |

## 甄别要点

- `duration_ms` 是**容器内实测墙钟**（正常判题 ~100-300ms）；02 的 `≈3000` 证明超时来自沙箱内部每用例时钟而非外层熔断。
- 05 的 `detail` 必须为空字符串——隐藏用例的通过情况一分钟都不允许离开宿主进程。
- 全程判题延迟 ~1-3 秒属正常（每次提交一个一次性容器的启动开销，ADR-0003）。

## 维护提示

- 这些文件含多行 `-c` 脚本，**保存为 LF 行尾**（CRLF 会让 bash 报 `\r: command not found`）。
- 修改判题契约（verdict 字段/类别集合）时同步更新本目录与 `judge/tests/test_executor.py`。
