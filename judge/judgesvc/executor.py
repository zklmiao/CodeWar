# -*- coding: utf-8 -*-
"""简易判题执行器（01 号票 Tracer 版）。

执行模型：把玩家代码写入临时目录 solution.py，生成 runner 脚本在子进程中
import 玩家代码并逐用例 eval 调用，整体超时由 subprocess.run(timeout) 控制。

边界说明：本版**无隔离**（共享文件系统/网络，仅整体超时）——判题沙箱的
硬化（资源限制、禁网、per-test 超时、防拖垮服务）是 02 号票的 ADR 范围。
"""
from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile
import time

# runner 模板：与玩家代码同目录运行；结果以单行 JSON 打到 stdout
_RUNNER = r'''
import json, sys
sys.path.insert(0, sys.argv[1])
cases = json.loads(sys.argv[2])

def fail(cat, detail=""):
    print(json.dumps({"cat": cat, "detail": detail}))
    raise SystemExit

try:
    import solution  # noqa: E402  玩家代码
except SyntaxError as e:
    fail("syntax", "line {}: {}".format(e.lineno, e.msg))
except BaseException as e:  # 导入期运行时错误（顶层异常）
    fail("runtime-error", type(e).__name__)

NS = vars(solution)  # 用玩家模块的命名空间求值（含其自定义辅助函数）

out = []
ok = True
for c in cases:
    try:
        actual = repr(eval(c["call"], NS))
        passed = actual == c["expected"]
        # 不回传 actual/expected——隐藏用例信息不出判题边界（设计 002 §3）
        out.append({"name": c["name"], "passed": passed})
        ok = ok and passed
    except BaseException as e:
        out.append({"name": c["name"], "passed": False, "err": type(e).__name__})
        ok = False

print(json.dumps({"cat": "ok", "passed": ok, "cases": out}))
'''

_CATEGORIES_SYNTAX = "syntax"
_CATEGORIES_RUNTIME = "runtime-error"
_CATEGORIES_TIMEOUT = "timeout"
_CATEGORIES_WRONG = "wrong-answer"
_CATEGORIES_OK = "ok"


def run(code: str, test_cases: list[dict], timeout_ms: int = 8000) -> dict:
    """判一次提交。返回 {passed, category, duration_ms, detail}。"""
    if not test_cases:
        return {"passed": False, "category": _CATEGORIES_WRONG, "duration_ms": 0, "detail": "no test cases"}

    with tempfile.TemporaryDirectory(prefix="codewar-judge-") as tmp:
        sol = os.path.join(tmp, "solution.py")
        runner = os.path.join(tmp, "_runner.py")
        with open(sol, "w", encoding="utf-8") as f:
            f.write(code)
        with open(runner, "w", encoding="utf-8") as f:
            f.write(_RUNNER)

        started = time.monotonic()
        try:
            proc = subprocess.run(
                [sys.executable, runner, tmp, json.dumps(test_cases)],
                capture_output=True, text=True, timeout=timeout_ms / 1000.0,
                cwd=tmp, env={**os.environ, "PYTHONIOENCODING": "utf-8"},
            )
        except subprocess.TimeoutExpired:
            return {"passed": False, "category": _CATEGORIES_TIMEOUT,
                    "duration_ms": int((time.monotonic() - started) * 1000), "detail": ""}

        duration_ms = int((time.monotonic() - started) * 1000)

        # runner 的结构化结果取 stdout 最后一行非空输出
        lines = [ln for ln in (proc.stdout or "").strip().splitlines() if ln.strip()]
        if not lines:
            return {"passed": False, "category": _CATEGORIES_RUNTIME,
                    "duration_ms": duration_ms, "detail": (proc.stderr or "")[-300:]}
        try:
            result = json.loads(lines[-1])
        except json.JSONDecodeError:
            return {"passed": False, "category": _CATEGORIES_RUNTIME,
                    "duration_ms": duration_ms, "detail": (proc.stderr or "")[-300:]}

        cat = result.get("cat")
        if cat == _CATEGORIES_OK:
            return {"passed": bool(result.get("passed")), "category": _CATEGORIES_OK if result.get("passed") else _CATEGORIES_WRONG,
                    "duration_ms": duration_ms, "detail": ""}
        return {"passed": False, "category": cat, "duration_ms": duration_ms, "detail": result.get("detail", "")}
