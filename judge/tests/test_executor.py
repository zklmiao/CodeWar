# -*- coding: utf-8 -*-
"""executor 的输入输出行为测试（spec「测试决策」Judge 节；作弊向量覆盖）。"""
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from judgesvc import executor  # noqa: E402

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


def test_correct_code_passes():
    r = executor.run(GOOD, CASES, timeout_ms=5000)
    assert r["passed"] is True
    assert r["category"] == "ok"
    # 真实子进程执行的耗时计量（python 启动即 >0ms）；`>= 0` 对 int 恒真
    assert r["duration_ms"] > 0


def test_syntax_error_reports_category_and_detail():
    r = executor.run(SYNTAX, CASES, timeout_ms=5000)
    assert r["passed"] is False
    assert r["category"] == "syntax"
    assert "line" in r["detail"]


def test_deadloop_killed_by_timeout():
    r = executor.run(DEADLOOP, CASES, timeout_ms=1500)
    assert r["passed"] is False
    assert r["category"] == "timeout"


def test_runtime_error():
    r = executor.run(TOPLEVEL_CRASH, CASES, timeout_ms=5000)
    assert r["passed"] is False
    assert r["category"] == "runtime-error"


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


def test_hidden_case_info_not_leaked():
    # 失败 detail 不得包含用例的期望值/实参（设计 002 §3）
    r = executor.run("def two_sum(nums, target):\n    return [1, 2]\n", CASES, timeout_ms=5000)
    assert r["passed"] is False
    assert "[0, 1]" not in r["detail"]
    assert "[2, 7, 11, 15]" not in r["detail"]


@pytest.mark.parametrize("bad", [None, ""])
def test_empty_code_fails(bad):
    r = executor.run(bad or "", CASES, timeout_ms=5000)
    assert r["passed"] is False
