# -*- coding: utf-8 -*-
"""沙箱判题执行器（02 号票：一次性容器，docker-out-of-docker）。

执行模型（ADR-0003）：每次提交启动一个一次性容器执行全部用例；
容器内 harness（_HARNESS）为每个用例 fork 一个 worker 子进程并施加
每用例超时，首败即停；容器整体另有墙钟上限（第二道保险）。

隔离与防泄漏（容器级，见 ADR-0003）：
--network=none 断网；--memory=256m + --pids-limit=64 + --cpus=0.5 配额；
--cap-drop=ALL；--read-only 只读根文件系统 + PYTHONDONTWRITEBYTECODE=1。
玩家代码经只读挂载注入；容器 stdin 只接收 call 表达式列表——期望值
是秘密，一分钟都不进容器：repr 与 expected 的比对、category 汇总全部
在宿主侧完成（worker 只能见到自己的输入，见不到任何期望值）。

失败反馈契约（设计 002 §3）：verdict 只含 {passed, category, duration_ms}，
detail 仅 syntax 携带行号；其余类别一律空字符串，绝不回传隐藏用例信息。
fail-closed：docker 不可用时返回 judge-error，绝不回退到无隔离执行。
"""
from __future__ import annotations

import inspect
import json
import os
import re
import shutil
import subprocess
import tempfile
import time

# judge 侧每用例超时常量（ADR-0003）：调用方传更小值时兼容采用，默认 3 秒
PER_TEST_TIMEOUT_MS = 3000
# 容器整体墙钟 slack（秒）：用例数×每用例超时 + 本值，宿主侧施加
WALL_CLOCK_SLACK_S = 30
# 沙箱镜像（固定版本，compose/Dockerfile 预拉取；环境变量仅供测试覆盖）
SANDBOX_IMAGE = os.environ.get("CODEWAR_SANDBOX_IMAGE", "python:3.12-slim")

_ALLOWED_CATEGORIES = frozenset({
    "ok", "wrong-answer", "runtime-error", "timeout",
    "limit-exceeded", "syntax", "judge-error",
})

# syntax detail 白名单：只接受 harness 按格式拼出的行号摘要
_SYNTAX_DETAIL_RE = re.compile(r"^line \d+: .{1,200}$")

# 采集整体输出后兜底等待上限（秒）：外层墙钟熔断 kill 后再收已 flush 的行
_DRAIN_TIMEOUT_S = 5


def _judge_error(duration_ms: int = 0) -> dict:
    return {"passed": False, "category": "judge-error",
            "duration_ms": duration_ms, "detail": ""}


def docker_available() -> bool:
    """宿主 docker CLI 是否可用（测试 skip 探针与 server 启动预检共用）。"""
    return shutil.which("docker") is not None


def ensure_sandbox_image() -> bool:
    """服务启动时确保沙箱镜像已缓存：缺失则 best-effort 拉取。

    离线且已缓存 → True（离线可判）；拉取失败且无缓存 → False。
    绝不因本函数异常拖垮服务启动（调用方仍可 try/except）。
    """
    if not docker_available():
        return False
    inspect_proc = subprocess.run(
        ["docker", "image", "inspect", SANDBOX_IMAGE],
        capture_output=True, timeout=30,
    )
    if inspect_proc.returncode == 0:
        return True
    pull = subprocess.run(
        ["docker", "pull", SANDBOX_IMAGE],
        capture_output=True, timeout=120,
    )
    return pull.returncode == 0


def _parse_json_lines(text):
    """逐行解析 JSONL 文本，返回解析成功的 dict 列表（顺序保持，非 dict 行丢弃）。

    共用函数：宿主侧直接调用；容器内 harness 通过内嵌的同源代码调用
    （容器无法 import 宿主模块，故 _HARNESS 模板内嵌本函数源码）。
    """
    out = []
    for ln in (text or "").splitlines():
        ln = ln.strip()
        if not ln:
            continue
        try:
            v = json.loads(ln)
        except ValueError:
            continue
        if isinstance(v, dict):
            out.append(v)
    return out


def build_calls_payload(test_cases: list[dict]) -> str:
    """构造进容器的 stdin 载荷：只含用例名与 call 表达式，不含期望值。"""
    return json.dumps([{"name": t["name"], "call": t["call"]} for t in test_cases])


# ---- 容器内 harness：父进程（逐用例 fork worker，首败即停） ----
# stdin 只收 call 列表（无期望值）；每行输出一个用例结论；正常结束（即使
# 首败即停）必以 summary 行收尾；全程 exit 0（非零退出一律视为 infra 故障）。
_HARNESS = (
    "import json, subprocess, sys, time\n"
    + inspect.getsource(_parse_json_lines)
    + r'''
calls = json.load(sys.stdin)
per_test_ms = int(sys.argv[1])
per_test_s = per_test_ms / 1000.0
t0 = time.monotonic()

def emit(obj):
    print(json.dumps(obj, ensure_ascii=False))
    sys.stdout.flush()

def summary():
    emit({"i": "done", "duration_ms": int((time.monotonic() - t0) * 1000)})

# 静态语法预检：compile 不执行代码，安全；行号信息无害可回传
try:
    with open("/sandbox/solution.py", encoding="utf-8") as f:
        src = f.read()
    compile(src, "solution.py", "exec")
except SyntaxError as e:
    emit({"i": -1, "status": "syntax",
          "detail": "line {}: {}".format(e.lineno, e.msg)[:210]})
    summary()
    raise SystemExit(0)
except OSError:
    emit({"i": -1, "status": "runtime-error"})
    summary()
    raise SystemExit(0)

for idx, c in enumerate(calls):
    try:
        proc = subprocess.run(
            [sys.executable, "/sandbox/_worker.py"],
            input=json.dumps({"i": idx, "call": c["call"]}),
            capture_output=True, text=True, timeout=per_test_s,
        )
    except subprocess.TimeoutExpired:
        emit({"i": idx, "status": "timeout"})  # 时钟杀
        break
    if proc.returncode in (-9, 137):
        emit({"i": idx, "status": "limit-exceeded"})  # 配额杀：OOM-killer
        break
    got = None
    for v in reversed(_parse_json_lines(proc.stdout)):
        if isinstance(v.get("outcome"), str):
            got = v
            break
    if got is None:
        emit({"i": idx, "status": "runtime-error"})  # worker 无结构化输出
        break
    outcome = got["outcome"]
    if outcome == "ok":
        emit({"i": idx, "status": "ok", "repr": got.get("repr", "")})
        continue
    if outcome in ("import-error", "call-error"):
        # 配额杀：内存耗尽或进程/线程配额耗尽（errno 11 = EAGAIN）；
        # 其余：导入期顶层崩溃 → runtime-error，调用期异常 → call-error。
        if got.get("err") == "MemoryError" or got.get("errno") == 11:
            emit({"i": idx, "status": "limit-exceeded"})
        elif outcome == "import-error":
            emit({"i": idx, "status": "runtime-error"})
        else:
            emit({"i": idx, "status": "call-error",
                  "err": got.get("err", "")})
        break
    emit({"i": idx, "status": "runtime-error"})  # 未知 outcome，fail-closed
    break

summary()
'''
)

# ---- 容器内 worker：单用例执行者（每个用例全新进程，全新 import） ----
# stdin 只收自己的 {"i", "call"}；全程见不到期望值（__main__ 命名空间里就没有）。
_WORKER = r'''
import io, json, sys

job = json.load(sys.stdin)
idx, call = job["i"], job["call"]

def emit(obj):
    sys.stdout.write(json.dumps(obj, ensure_ascii=False) + "\n")
    sys.stdout.flush()

# 纵深配额（best-effort）：大内存分配转 MemoryError。
# NPROC 注：容器内为 root 时内核豁免 NPROC 检查（此设置对当前运行形态无效，
# 保留以覆盖将来 --user 运行）；进程/线程创建实际受容器级 --pids-limit 约束，
# fork 炸弹在该层转 EAGAIN/errno 11，由 harness 按 errno 归因。
# 注：解题不需要建进程/线程，配额误伤一律 fail-closed。
try:
    import resource
    resource.setrlimit(resource.RLIMIT_AS, (256 << 20, 256 << 20))
    try:
        resource.setrlimit(resource.RLIMIT_NPROC, (0, 0))
    except (ValueError, OSError):
        pass
except (ImportError, OSError):
    pass

# 玩家代码的打印/报错截留在内存中，绝不进 worker 协议流（防协议污染与信息泄漏）
buf_out, buf_err = io.StringIO(), io.StringIO()
sys.path.insert(0, "/sandbox")
old_out, old_err = sys.stdout, sys.stderr
sys.stdout, sys.stderr = buf_out, buf_err
try:
    try:
        import solution  # noqa: E402  玩家代码（每 worker 全新 import）
    except BaseException as e:
        result = {"i": idx, "outcome": "import-error",
                  "err": type(e).__name__, "errno": getattr(e, "errno", None)}
    else:
        NS = vars(solution)  # 玩家模块命名空间求值（含其自定义辅助函数）
        try:
            result = {"i": idx, "outcome": "ok",
                      "repr": repr(eval(call, dict(NS)))}
        except BaseException as e:
            result = {"i": idx, "outcome": "call-error",
                      "err": type(e).__name__, "errno": getattr(e, "errno", None)}
finally:
    sys.stdout, sys.stderr = old_out, old_err

emit(result)
'''


def _assemble_verdict(lines: list[dict], test_cases: list[dict],
                       fallback_ms: int) -> dict:
    """宿主侧汇总：按用例序号顺序取首个失败归因；repr 比对在此完成。

    lines：harness 输出的已解析行；fallback_ms：无 summary 行时的耗时兜底。
    单一判定点：每个分支直接返回 verdict，不存在"先算 passed 再改写"。
    """
    by_i = {}
    duration_ms = fallback_ms
    for v in lines:
        i = v.get("i")
        if isinstance(i, bool):
            continue
        if i == "done":
            try:
                duration_ms = max(int(v.get("duration_ms", 0) or 0), 0)
            except (ValueError, TypeError):
                pass
        elif isinstance(i, int) and i not in by_i:
            by_i[i] = v  # 首行 wins：单写者协议，多行只可能是 fork 残留

    def verdict(category, detail=""):
        return {"passed": category == "ok", "category": category,
                "duration_ms": duration_ms, "detail": detail}

    syn = by_i.get(-1)
    if isinstance(syn, dict) and syn.get("status") == "syntax":
        detail = syn.get("detail", "")
        if not isinstance(detail, str) or not _SYNTAX_DETAIL_RE.match(detail):
            detail = ""
        return verdict("syntax", detail)

    for idx, tc in enumerate(test_cases):
        r = by_i.get(idx)
        if r is None:
            return verdict("timeout")  # 未完成（外层熔断/容器夭折）
        status = r.get("status")
        if status == "ok":
            if not isinstance(r.get("repr"), str) or r["repr"] != tc["expected"]:
                return verdict("wrong-answer")  # 首个值不符即停
            continue
        if status == "timeout":
            return verdict("timeout")  # 时钟杀
        if status == "limit-exceeded":
            return verdict("limit-exceeded")  # 配额杀
        if status == "runtime-error":
            return verdict("runtime-error")  # 导入期顶层崩溃/协议外崩溃
        if status == "call-error":
            # 调用期异常 = 没算出值，不可能正确；配额类异常单独归因
            if r.get("err") == "MemoryError" or r.get("errno") == 11:
                return verdict("limit-exceeded")
            return verdict("wrong-answer")
        return verdict("judge-error")  # 未知 status：不采信，infra 故障计

    return verdict("ok")


def run(code: str, test_cases: list[dict], timeout_ms: int = 8000) -> dict:
    """判一次提交。返回 {passed, category, duration_ms, detail}（白名单四键）。

    timeout_ms：每用例超时（毫秒）；<=0 则用 judge 侧常量；大于常量则钳制
    到常量（延迟预算见 ADR-0003）。容器整体另有墙钟上限（第二道保险）。
    duration_ms：容器内 harness 实测的用例总墙钟（summary 行；容器被熔断时
    退化为宿主实测墙钟）；与玩家解题耗时是两个量，沙箱侧不换算只上报。
    """
    if not test_cases:
        return {"passed": False, "category": "wrong-answer",
                "duration_ms": 0, "detail": ""}
    if not docker_available():
        return _judge_error()  # fail-closed：无 docker 不判（绝不裸奔执行）

    if timeout_ms and timeout_ms > 0:
        per_test_ms = min(timeout_ms, PER_TEST_TIMEOUT_MS)
    else:
        per_test_ms = PER_TEST_TIMEOUT_MS
    outer_s = len(test_cases) * (per_test_ms / 1000.0) + WALL_CLOCK_SLACK_S
    calls_json = build_calls_payload(test_cases)

    with tempfile.TemporaryDirectory(prefix="codewar-judge-") as tmp:
        # 挂载目录只含 solution.py 与 harness/worker 脚本；用例（仅 call）
        # 经 stdin 管道注入、全程不落盘；期望值不出宿主进程。
        with open(os.path.join(tmp, "solution.py"), "w", encoding="utf-8") as f:
            f.write(code or "")
        with open(os.path.join(tmp, "_harness.py"), "w", encoding="utf-8") as f:
            f.write(_HARNESS)
        with open(os.path.join(tmp, "_worker.py"), "w", encoding="utf-8") as f:
            f.write(_WORKER)

        cmd = [
            "docker", "run", "--rm", "-i",
            "--network=none",
            "--memory=256m",
            "--pids-limit=64",
            "--cpus=0.5",
            "--cap-drop=ALL",
            "--read-only",
            "--tmpfs", "/tmp:size=16m,exec",
            "-v", "{}:/sandbox:ro".format(os.path.abspath(tmp)),
            "-e", "PYTHONDONTWRITEBYTECODE=1",
            SANDBOX_IMAGE,
            "python", "/sandbox/_harness.py", str(per_test_ms),
        ]
        started = time.monotonic()
        proc = subprocess.Popen(
            cmd, stdin=subprocess.PIPE,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
        )
        try:
            out, _ = proc.communicate(input=calls_json, timeout=outer_s)
            returncode = proc.returncode
        except subprocess.TimeoutExpired:
            # 整体墙钟熔断（第二道保险）：kill 后收已 flush 的行，已完成的
            # 用例按其结论计入，未完成的在汇总时判 timeout。
            proc.kill()
            try:
                out, _ = proc.communicate(timeout=_DRAIN_TIMEOUT_S)
            except subprocess.TimeoutExpired:
                out = ""
            returncode = proc.returncode
        except OSError:
            # docker 调用失败（无 daemon 等 infra 故障）
            return _judge_error(int((time.monotonic() - started) * 1000))

        elapsed_ms = int((time.monotonic() - started) * 1000)
        if returncode != 0:
            return _judge_error(elapsed_ms)

        lines = _parse_json_lines(out)
        if not lines:
            return _judge_error(elapsed_ms)
        return _assemble_verdict(lines, test_cases, elapsed_ms)
