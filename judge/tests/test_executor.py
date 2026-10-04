# -*- coding: utf-8 -*-
"""executor 的输入输出行为测试（spec「测试决策」Judge 节；作弊向量覆盖）。

全部走真实容器沙箱（ADR-0003）：每个容器化测试有 1-3 秒开销，数量克制。
无 docker 环境时容器化测试 skip（fail-closed，不伪装通过）。
"""
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from judgesvc import executor  # noqa: E402

needs_docker = pytest.mark.skipif(
    not executor.docker_available(), reason="需要 docker 沙箱",
)

CASES = [
    {"name": "basic", "call": "two_sum([2, 7, 11, 15], 9)", "expected": "[0, 1]"},
    {"name": "middle", "call": "two_sum([3, 2, 4], 6)", "expected": "[1, 2]"},
]

GOOD = (
    "def two_sum(nums, target):\n"
    "    seen = {}\n"
    "    for i, n in enumerate(nums):\n"
    "        if target - n in seen:\n"
    "            return [seen[target - n], i]\n"
    "        seen[n] = i\n"
)

SYNTAX = "def two_sum(nums, target)\n    return [0, 1]"  # 缺冒号

DEADLOOP = (
    "def two_sum(nums, target):\n"
    "    while True:\n"
    "        pass\n"
)

TOPLEVEL_CRASH = "raise ValueError('boom')\n" + GOOD

NETWORK = (
    "import socket\n"
    "def two_sum(nums, target):\n"
    "    socket.create_connection(('8.8.8.8', 53), timeout=3)\n"
    "    return [0, 1]\n"
)

MEMORY_BOMB = (
    "def two_sum(nums, target):\n"
    "    x = 'A' * (1024 * 1024 * 1024)\n"
    "    return [0, 1]\n"
)

FORK_BOMB = (
    "import os\n"
    "def two_sum(nums, target):\n"
    "    for _ in range(100000):\n"
    "        os.fork()\n"
    "    return [0, 1]\n"
)


@needs_docker
def test_correct_code_passes():
    r = executor.run(GOOD, CASES, timeout_ms=5000)
    assert r["passed"] is True
    assert r["category"] == "ok"
    # 容器内实测的用例总墙钟（python 启动即 >0ms）；`>= 0` 对 int 恒真
    assert r["duration_ms"] > 0


@needs_docker
def test_syntax_error_reports_category_and_detail():
    r = executor.run(SYNTAX, CASES, timeout_ms=5000)
    assert r["passed"] is False
    assert r["category"] == "syntax"
    assert "line" in r["detail"]


@needs_docker
def test_deadloop_killed_by_timeout():
    r = executor.run(DEADLOOP, CASES, timeout_ms=1500)
    assert r["passed"] is False
    assert r["category"] == "timeout"


@needs_docker
def test_runtime_error():
    r = executor.run(TOPLEVEL_CRASH, CASES, timeout_ms=5000)
    assert r["passed"] is False
    assert r["category"] == "runtime-error"


@needs_docker
def test_partial_pass_counts_as_fail():
    # 只对第一个用例正确的解：部分通过必须判失败（不允许部分伤害）
    partial = (
        "def two_sum(nums, target):\n"
        "    return [0, 1]\n"  # 硬编码第一个用例
    )
    r = executor.run(partial, CASES, timeout_ms=5000)
    assert r["passed"] is False
    assert r["category"] == "wrong-answer"


def test_empty_cases_rejected():
    r = executor.run(GOOD, [], timeout_ms=5000)
    assert r["passed"] is False


@needs_docker
def test_hidden_case_info_not_leaked():
    # 失败 detail 不得包含用例的期望值/实参（设计 002 §3）
    r = executor.run("def two_sum(nums, target):\n    return [1, 2]\n", CASES, timeout_ms=5000)
    assert r["passed"] is False
    assert "[0, 1]" not in r["detail"]
    assert "[2, 7, 11, 15]" not in r["detail"]


@needs_docker
@pytest.mark.parametrize("bad", [None, ""])
def test_empty_code_fails(bad):
    r = executor.run(bad or "", CASES, timeout_ms=5000)
    assert r["passed"] is False


@needs_docker
def test_network_attempt_blocked():
    # 容器级断网：socket 建连立即失败，提交判失败且服务存活
    r = executor.run(NETWORK, CASES, timeout_ms=5000)
    assert r["passed"] is False
    assert r["category"] in ("wrong-answer", "timeout")
    assert r["detail"] == ""


@needs_docker
def test_memory_bomb_limit_exceeded():
    # 1GB 字符串：配额杀，与时钟杀 timeout 区分，且容器不垮
    r = executor.run(MEMORY_BOMB, CASES, timeout_ms=5000)
    assert r["passed"] is False
    assert r["category"] == "limit-exceeded"
    assert r["detail"] == ""


@needs_docker
def test_fork_bomb_limit_exceeded():
    # 无限进程：pids 配额杀，容器不垮
    r = executor.run(FORK_BOMB, CASES, timeout_ms=5000)
    assert r["passed"] is False
    assert r["category"] == "limit-exceeded"
    assert r["detail"] == ""


@needs_docker
def test_service_survives_malicious_submission():
    # 恶意提交（死循环）之后，紧接着的正常提交仍正确判题（服务不垮的直接证据）
    bad = executor.run(DEADLOOP, CASES, timeout_ms=1500)
    assert bad["passed"] is False
    good = executor.run(GOOD, CASES, timeout_ms=5000)
    assert good["passed"] is True
    assert good["category"] == "ok"


@needs_docker
def test_verdict_contract_whitelist():
    # 契约测试：verdict 字段白名单——多一个字段都算泄漏（设计 002 §3）
    for code in (GOOD, SYNTAX, DEADLOOP):
        r = executor.run(code, CASES, timeout_ms=1500)
        assert set(r.keys()) == {"passed", "category", "duration_ms", "detail"}
    r = executor.run(GOOD, CASES, timeout_ms=5000)
    assert r["detail"] == ""
    r = executor.run(SYNTAX, CASES, timeout_ms=5000)
    assert r["detail"].startswith("line ")


def test_no_docker_is_fail_closed(monkeypatch):
    # docker 不可用 → judge-error，绝不回退到无隔离执行
    monkeypatch.setattr(executor, "docker_available", lambda: False)
    r = executor.run(GOOD, CASES, timeout_ms=5000)
    assert r == {"passed": False, "category": "judge-error",
                 "duration_ms": 0, "detail": ""}
