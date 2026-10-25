# -*- coding: utf-8 -*-
"""executor 的输入输出行为测试（spec「测试决策」Judge 节；作弊向量覆盖）。

全部走真实容器沙箱（ADR-0003）：每个容器化测试有 1-3 秒开销，数量克制。
无 docker 环境时容器化测试 skip（fail-closed，不伪装通过）。
"""
import json
import subprocess
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from judgesvc import executor  # noqa: E402

needs_docker = pytest.mark.skipif(
    not executor.docker_ready(), reason="需要可用的 docker daemon（沙箱判题）",
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

# 窥探者：枚举 /sandbox 找用例文件，找到就按实参精确返回期望值。
# 期望值反序列化用 json.loads（把泄漏的 repr 串变回值，对本组 JSON 兼容
# 的 expected 与 eval 等效）——测试字符串自身不构成动态执行报点（26 号票）；
# 被测性质不变：无泄漏则只能退化为硬编码 → 部分通过判失败。
SNOOP = (
    "import os, json\n"
    "def two_sum(nums, target):\n"
    "    key = 'two_sum(%r, %r)' % (nums, target)\n"
    "    for fn in os.listdir('/sandbox'):\n"
    "        if fn.endswith('.json'):\n"
    "            for c in json.load(open('/sandbox/' + fn)):\n"
    "                if c['call'] == key:\n"
    "                    return json.loads(c['expected'])\n"
    "    return [0, 1]\n"
)

# 进程内窃取者：扫描 __main__ 命名空间找期望值，找到就地返回正确结果。
# 反序列化同样用 json.loads 而非 eval（26 号票：消除测试字符串报点）。
# 期望值不出宿主时只能退化为硬编码 → 部分通过判失败；
# 若期望值泄漏进容器，本测试会判 passed=True 而变红（真回归测试）。
STEALER = (
    "import sys, json\n"
    "def two_sum(nums, target):\n"
    "    ns = getattr(sys.modules.get('__main__'), '__dict__', {})\n"
    "    for v in list(ns.values()):\n"
    "        if isinstance(v, dict) and isinstance(v.get('expected'), str):\n"
    "            return json.loads(v['expected'])\n"
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
    # 容器级断网：--network=none 下 connect 即时抛 OSError(ENETUNREACH)，
    # 不等待封包超时（实测 3/3 次 ~1s 内返回，均为容器启动开销）；
    # 调用期异常按值失败语义判 wrong-answer（机理确定，pin 单值）。
    r = executor.run(NETWORK, CASES, timeout_ms=5000)
    assert r["passed"] is False
    assert r["category"] == "wrong-answer"
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
    # 遏制即契约：正常时 EAGAIN 先到（limit-exceeded）；64 个进程在 0.5 CPU
    # 配额下自旋时 worker 可能被饥饿拖到每用例时钟先到期（timeout）。
    # 两者都证明炸弹被容器配额杀死，宿主无恙。
    assert r["category"] in ("limit-exceeded", "timeout")
    assert r["detail"] == ""


@needs_docker
def test_snoop_cannot_read_cases_from_disk():
    # 用例经 stdin 注入、全程不落盘：枚举 /sandbox 找不到任何用例文件，
    # 窥探代码只能退化为硬编码 → 部分通过判失败
    r = executor.run(SNOOP, CASES, timeout_ms=5000)
    assert r["passed"] is False
    assert r["category"] == "wrong-answer"


def test_calls_payload_has_no_expected():
    # 进容器的 stdin 载荷 = 解法源码 + worker 脚本 + 构造辅助库 + calls：
    # 键白名单断言（多一个键都算泄漏——尤其不允许出现 expected）。
    # 15 号票 v2：calls 条目携带 setup/compare（题作者声明，非秘密），
    # 但 expected 依旧一秒都不进容器——删除任何一行实现本测试必须变红。
    payload = json.loads(executor.build_stdin_payload(GOOD, CASES, "", executor._HELPERS))
    assert set(payload) == {"solution", "worker", "helpers", "calls"}
    assert all(set(c) == {"name", "call", "setup", "compare"} for c in payload["calls"])
    assert "expected" not in payload["solution"]
    assert '"expected"' not in payload  # 整个载荷字符串级不含期望值键名

    # v2 传播：声明了 setup/compare 的用例，声明内容逐字进载荷
    cases_v2 = [{"name": "t", "call": "f(head)", "expected": "X",
                 "setup": "head = list_to_nodes([1,2])",
                 "compare": "def _normalize(got): return nodes_repr(got)"}]
    p2 = json.loads(executor.build_stdin_payload(GOOD, cases_v2, "", executor._HELPERS))
    assert p2["calls"][0]["setup"] == "head = list_to_nodes([1,2])"
    assert p2["calls"][0]["compare"].startswith("def _normalize")
    assert "X" not in p2  # 期望值绝不随载荷进容器


def test_harness_prefailure_maps_to_judge_error():
    # harness 前置故障行（宿主读 solution.py 失败，玩家代码未被执行）：
    # 纯函数单测，无需容器。属基础设施故障 → judge-error（不罚不锁可重交），
    # 归 runtime-error/timeout 会让无辜玩家自损
    r = executor._assemble_verdict(
        [{"i": -1, "status": "runtime-error"}], CASES, 5)
    assert r == {"passed": False, "category": "judge-error",
                 "duration_ms": 5, "detail": ""}


def test_empty_drain_output_maps_to_judge_error(monkeypatch):
    # 熔断后 drain 出 0 条结论行 = harness 挂死/夭折（infra）→ judge-error，
    # 而不是把全部用例判 timeout 惩罚玩家（走 run() 全链路的 mock 测试）
    class _MuteContainer:
        def communicate(self, input=None, timeout=None):
            if input is not None:  # 首次 communicate（携带用例）：整体墙钟熔断
                raise subprocess.TimeoutExpired(cmd="docker", timeout=1)
            return ("", "")  # drain：空输出

        def kill(self):
            return None

    monkeypatch.setattr(subprocess, "Popen", lambda *a, **k: _MuteContainer())
    r = executor.run(GOOD, CASES, timeout_ms=5000)
    assert r["passed"] is False
    assert r["category"] == "judge-error"
    assert r["detail"] == ""


@needs_docker
def test_worker_cannot_steal_expected_in_process():
    # 期望值不进容器：__main__ 命名空间里就没有，进程内扫描拿不到任何
    # expected repr，窃取代码退化为硬编码 → 部分通过判失败
    r = executor.run(STEALER, CASES, timeout_ms=5000)
    assert r["passed"] is False
    assert r["category"] == "wrong-answer"
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


def test_popen_failure_is_judge_error(monkeypatch):
    # docker 二进制在可用性检查后消失：Popen 抛 OSError → judge-error，不裸抛
    def _gone(*a, **k):
        raise OSError("docker binary vanished")

    monkeypatch.setattr(subprocess, "Popen", _gone)
    r = executor.run(GOOD, CASES, timeout_ms=5000)
    assert r == {"passed": False, "category": "judge-error",
                 "duration_ms": 0, "detail": ""}


@needs_docker
def test_outer_timeout_drains_completed_lines(monkeypatch):
    # 熔断路径（mock，不依赖真实竞态）：容器超时被 kill 后，已 flush 的行
    # （case0 值对）按行结论计入汇总，未完成的 case1 判 timeout。
    flushed = '{"i": 0, "status": "ok", "repr": "[0, 1]"}\n'
    state = {"calls": 0}

    class _HungContainer:
        def communicate(self, input=None, timeout=None):
            state["calls"] += 1
            if state["calls"] == 1:
                raise subprocess.TimeoutExpired(cmd="docker", timeout=1)
            return (flushed, "")

        def kill(self):
            return None

    monkeypatch.setattr(subprocess, "Popen", lambda *a, **k: _HungContainer())
    r = executor.run(GOOD, CASES, timeout_ms=5000)
    assert state["calls"] == 2  # 熔断 + drain 各一次 communicate
    assert r["passed"] is False
    assert r["category"] == "timeout"
    assert set(r.keys()) == {"passed", "category", "duration_ms", "detail"}
    assert r["detail"] == ""
