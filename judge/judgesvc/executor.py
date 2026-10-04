# -*- coding: utf-8 -*-
"""沙箱判题执行器（02 号票：一次性容器，docker-out-of-docker）。

执行模型（ADR-0003）：每次提交启动一个一次性容器执行全部用例；
容器内 harness（_HARNESS）为每个用例 fork 一个 worker 子进程并施加
每用例超时，首败即停；容器整体另有墙钟上限（第二道保险）。

隔离（容器级，见 ADR-0003 与 Dockerfile.judge / compose 挂载）：
--network=none 断网；--memory=256m + --pids-limit=64 + --cpus=0.5 配额；
--cap-drop=ALL；--read-only 只读根文件系统 + PYTHONDONTWRITEBYTECODE=1；
玩家代码经只读挂载注入（用例文件名随机化，防玩家代码窥探用例文件）。

失败反馈契约（设计 002 §3）：verdict 只含 {passed, category, duration_ms}，
detail 仅 syntax 携带行号；其余类别一律空字符串，绝不回传隐藏用例信息。
fail-closed：docker 不可用时返回 judge-error，绝不回退到无隔离执行。
"""
from __future__ import annotations

import json
import os
import shutil
import subprocess
import tempfile
import time
import uuid

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
    inspect = subprocess.run(
        ["docker", "image", "inspect", SANDBOX_IMAGE],
        capture_output=True, timeout=30,
    )
    if inspect.returncode == 0:
        return True
    pull = subprocess.run(
        ["docker", "pull", SANDBOX_IMAGE],
        capture_output=True, timeout=300,
    )
    return pull.returncode == 0


# ---- 容器内 harness：父进程（逐用例 fork worker，首败即停） ----
_HARNESS = r'''
import json, subprocess, sys, time

per_test_ms = int(sys.argv[1])
cases_path = sys.argv[2]
with open(cases_path, encoding="utf-8") as f:
    cases = json.load(f)

def emit(passed, cat, detail, t0):
    print(json.dumps({"passed": passed, "category": cat,
                      "duration_ms": int((time.monotonic() - t0) * 1000),
                      "detail": detail}, ensure_ascii=False))
    sys.stdout.flush()

t0 = time.monotonic()

# 静态语法预检：compile 不执行代码，安全；行号信息无害可回传
try:
    with open("/sandbox/solution.py", encoding="utf-8") as f:
        src = f.read()
    compile(src, "solution.py", "exec")
except SyntaxError as e:
    emit(False, "syntax", "line {}: {}".format(e.lineno, e.msg), t0)
    raise SystemExit(0)
except OSError:
    emit(False, "runtime-error", "", t0)
    raise SystemExit(0)

per_test_s = per_test_ms / 1000.0
for i in range(len(cases)):
    try:
        proc = subprocess.run(
            [sys.executable, "/sandbox/_worker.py", cases_path, str(i)],
            capture_output=True, text=True, timeout=per_test_s,
        )
    except subprocess.TimeoutExpired:
        emit(False, "timeout", "", t0)  # 时钟杀
        raise SystemExit(0)
    lines = [ln for ln in (proc.stdout or "").splitlines() if ln.strip()]
    res = None
    if lines:
        try:
            res = json.loads(lines[-1])
        except (ValueError, TypeError):
            res = None
    if not isinstance(res, dict) or "ok" not in res:
        # worker 无结构化输出：被 OOM-killer 处决 → 配额杀；其余 → 运行崩溃
        if proc.returncode in (-9, 137):
            emit(False, "limit-exceeded", "", t0)
        else:
            emit(False, "runtime-error", "", t0)
        raise SystemExit(0)
    if res.get("ok"):
        continue
    phase, err = res.get("phase"), res.get("err")
    # 配额杀：MemoryError，或 errno 11（EAGAIN：进程/线程配额耗尽；
    # err 名可能是 OSError/BlockingIOError 等子类，以 errno 为准）
    if err == "MemoryError" or res.get("errno") == 11:
        emit(False, "limit-exceeded", "", t0)
    elif phase == "import":
        emit(False, "runtime-error", "", t0)  # 导入期顶层崩溃
    else:
        emit(False, "wrong-answer", "", t0)  # 用例逻辑失败/调用期异常
    raise SystemExit(0)

emit(True, "ok", "", t0)
'''

# ---- 容器内 worker：单用例执行者（每个用例全新进程，全新 import） ----
_WORKER = r'''
import io, json, sys

cases_path, idx = sys.argv[1], int(sys.argv[2])
with open(cases_path, encoding="utf-8") as f:
    case = json.load(f)[idx]

class _ImportFailed(Exception):
    def __init__(self, cause):
        super().__init__()
        self.cause = cause

class _CallFailed(Exception):
    def __init__(self, cause):
        super().__init__()
        self.cause = cause

def emit(ok, phase="", err="", errno=None):
    sys.stdout.write(json.dumps({"ok": ok, "phase": phase,
                                 "err": err, "errno": errno}) + "\n")
    sys.stdout.flush()

# 纵深配额（best-effort）：大内存分配转 MemoryError；NPROC 尝试禁建进程
# （注：容器内为 root 时内核豁免 NPROC 检查，真正生效的是容器级 --pids-limit；
# 此处保留设置以覆盖将来 --user 运行；fork 炸弹在任一层都会转 EAGAIN/errno 11）
# 注：沙箱内玩家代码禁建进程/线程（解题不需要），误伤一律 fail-closed。
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
        raise _ImportFailed(e)
    NS = vars(solution)  # 玩家模块命名空间求值（含其自定义辅助函数）
    try:
        passed = repr(eval(case["call"], dict(NS))) == case["expected"]
    except BaseException as e:
        raise _CallFailed(e)
    result = (True, "call", "", None) if passed else (False, "call", "WrongAnswer", None)
except _ImportFailed as e:
    cause = e.cause
    result = (False, "import", type(cause).__name__, getattr(cause, "errno", None))
except _CallFailed as e:
    cause = e.cause
    result = (False, "call", type(cause).__name__, getattr(cause, "errno", None))
finally:
    sys.stdout, sys.stderr = old_out, old_err

emit(*result)
'''


def run(code: str, test_cases: list[dict], timeout_ms: int = 8000) -> dict:
    """判一次提交。返回 {passed, category, duration_ms, detail}（白名单四键）。

    timeout_ms：每用例超时（毫秒）；<=0 则用 judge 侧常量；大于常量则钳制
    到常量（延迟预算见 ADR-0003）。容器整体另有墙钟上限（第二道保险）。
    duration_ms：容器内 harness 实测的全部用例总墙钟（与解题耗时无关）。
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

    with tempfile.TemporaryDirectory(prefix="codewar-judge-") as tmp:
        # 用例文件名随机化：玩家代码即使枚举 /sandbox 也猜不到用例路径
        cases_name = "cases-{}.json".format(uuid.uuid4().hex)
        with open(os.path.join(tmp, "solution.py"), "w", encoding="utf-8") as f:
            f.write(code or "")
        with open(os.path.join(tmp, cases_name), "w", encoding="utf-8") as f:
            json.dump(test_cases, f)
        with open(os.path.join(tmp, "_harness.py"), "w", encoding="utf-8") as f:
            f.write(_HARNESS)
        with open(os.path.join(tmp, "_worker.py"), "w", encoding="utf-8") as f:
            f.write(_WORKER)

        cmd = [
            "docker", "run", "--rm",
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
            "/sandbox/" + cases_name,
        ]
        started = time.monotonic()
        try:
            proc = subprocess.run(
                cmd, capture_output=True, text=True, timeout=outer_s,
            )
        except subprocess.TimeoutExpired:
            # 整体墙钟熔断（第二道保险）：按时钟杀归因
            return {"passed": False, "category": "timeout",
                    "duration_ms": int((time.monotonic() - started) * 1000),
                    "detail": ""}
        except OSError:
            # docker 调用失败（无 daemon 等 infra 故障）
            return _judge_error(int((time.monotonic() - started) * 1000))

        if proc.returncode != 0:
            return _judge_error(int((time.monotonic() - started) * 1000))

        lines = [ln for ln in (proc.stdout or "").splitlines() if ln.strip()]
        if not lines:
            return _judge_error(int((time.monotonic() - started) * 1000))
        try:
            result = json.loads(lines[-1])
        except (ValueError, TypeError):
            return _judge_error(int((time.monotonic() - started) * 1000))
        if not isinstance(result, dict):
            return _judge_error(int((time.monotonic() - started) * 1000))

        # 白名单重建 verdict：容器输出只采信已知字段，多一个键都丢弃
        category = result.get("category")
        if category not in _ALLOWED_CATEGORIES:
            return _judge_error(0)
        try:
            duration_ms = max(int(result.get("duration_ms", 0) or 0), 0)
        except (ValueError, TypeError):
            duration_ms = 0
        detail = result.get("detail") if category == "syntax" else ""
        if not isinstance(detail, str):
            detail = ""
        passed = bool(result.get("passed")) and category == "ok"
        if result.get("passed") and category != "ok":
            category = "wrong-answer"  # 容器自称通过却非 ok：不采信，判失败
            passed = False
        return {"passed": passed, "category": category,
                "duration_ms": duration_ms, "detail": detail}
