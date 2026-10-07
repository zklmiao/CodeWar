# -*- coding: utf-8 -*-
"""15 号票：判题契约 v2——结构体类题支持（setup/compare/构造辅助库）。

虚假 pass 零容忍断言：
- 比较器各 false 分支逐个验证（got≠expected 的每种形态）
- setup 在玩家代码 eval 之前可用（setup 定义的绑定对 call 生效）
- 规范化 repr 对环有界（1000 节点环快速完成，不死循环）
- expected 不进容器（test_executor.py::test_calls_payload_has_no_expected 强化版）
- 题作者代码卫兵（超长/禁用调用/语法错 → judge-error，不注入容器）
"""
import json
import time

import pytest

from judgesvc import executor

needs_docker = pytest.mark.skipif(
    not executor.docker_ready(), reason="需要可用的 docker daemon（沙箱判题）",
)

# ===== 纯函数段：构造辅助库与卫兵（无需容器） =====


def _helpers_ns():
    """在测试进程内 exec 辅助库，返回其 EXPORTS（与容器内同源代码）。"""
    ns = {}
    exec(executor._HELPERS, ns)
    return ns


def test_helpers_nodes_roundtrip():
    h = _helpers_ns()
    head = h["list_to_nodes"]([1, 2, 3])
    assert h["nodes_repr"](head) == "[1,2,3]"
    assert h["nodes_repr"](None) == "None"


def test_helpers_cycle_repr_identity_rotation():
    # 环形链表 II 的身份语义：从返回节点出发的规范环序列。
    # [3,2,0,-4] pos=1：尾节点连到 index 1（值 2）——**3 不在环上**（LC 官方
    # 语义），入口的规范环序 = "cycle:[2,0,-4]"
    h = _helpers_ns()
    head = h["list_to_nodes"]([3, 2, 0, -4])
    h["make_cycle"](head, 1)
    entry = head.next  # pos=1
    assert h["nodes_repr"](entry) == "cycle:[2,0,-4]"
    # 同环不同入口 → 不同旋转序列（身份可区分）：从头出发先含 3
    assert h["nodes_repr"](head) == "cycle:[3,2,0,-4]"


def test_helpers_cycle_repr_bounded():
    # 环有界性：1000 节点大环规范化必须快速完成（不死循环）——超时即红
    h = _helpers_ns()
    head = h["list_to_nodes"](list(range(1000)))
    h["make_cycle"](head, 0)
    t0 = time.monotonic()
    r = h["nodes_repr"](head)
    assert time.monotonic() - t0 < 2.0
    assert r.startswith("cycle:[0,1,2,3,4,")
    assert r.endswith("999]")


def test_helpers_build_tree_level_order():
    h = _helpers_ns()
    root = h["build_tree"]([3, 9, 20, None, None, 15, 7])
    assert root.val == 3
    assert root.left.val == 9 and root.right.val == 20
    assert root.right.left.val == 15 and root.right.right.val == 7
    assert h["build_tree"]([]) is None


def test_guard_rejects_oversize():
    with pytest.raises(ValueError):
        executor.guard_author_code("x = 1\n" * 1000, "setup")


def test_guard_rejects_denylist():
    for bad in (
        "open('/etc/passwd')",
        "__import__('os').system('id')",
        "import subprocess",
        "exec('x=1')",
    ):
        with pytest.raises(ValueError):
            executor.guard_author_code(bad, "setup")


def test_guard_accepts_legit_author_code():
    executor.guard_author_code(
        "head = list_to_nodes([1,2,3])\nprev = None", "setup")
    executor.guard_author_code(
        "def _normalize(got):\n    return nodes_repr(got)", "compare")


@needs_docker
def test_guard_violation_maps_to_judge_error():
    # 卫兵违规 = 模板缺陷 → judge-error（fail-closed：不注入、不罚玩家）。
    # 必须 @needs_docker：无 docker 时 docker_available() 先短路返回 judge-error，
    # 测试会走错路径而假绿——卫兵逻辑本身从未被执行（review 发现 1）。
    r = executor.run(
        "def f():\n    return 1",
        [{"name": "t1", "call": "f()", "expected": "1",
          "setup": "open('/etc/passwd')"}],
        timeout_ms=3000,
    )
    assert r["category"] == "judge-error" and r["passed"] is False


@needs_docker
def test_player_definitions_win_over_helpers():
    # 回归钉（review 发现）：构造辅助库只补缺、不覆盖——玩家自定义同名符号
    # （如自建 list_to_nodes）必须优先，否则按玩家实现书写的题面会被判错。
    # 修复前 NS.update(EXPORTS) 会用辅助库覆盖玩家的 list_to_nodes。
    r = executor.run(
        "def list_to_nodes(arr):\n"
        "    return 42\n"
        "def f():\n"
        "    return list_to_nodes([1, 2, 3])",
        [{"name": "t1", "call": "f()", "expected": "42",
          "setup": "head = list_to_nodes([1, 2, 3])"}],
        timeout_ms=3000,
    )
    assert r["passed"] is True and r["category"] == "ok"


# ===== 端到端样题×4（真实容器判题；同时是 12 号票的格式范例） =====

REVERSE_LIST_SOLUTION = (
    "class ListNode:\n"
    "    def __init__(self, val=0, nxt=None):\n"
    "        self.val = val\n"
    "        self.next = nxt\n"
    "def reverseList(head):\n"
    "    prev = None\n"
    "    while head is not None:\n"
    "        head.next, prev, head = prev, head, head.next\n"
    "    return prev\n"
)
REVERSE_LIST_SETUP = "head = list_to_nodes([1,2,3,4,5])"
REVERSE_LIST_COMPARE = "def _normalize(got):\n    return nodes_repr(got)"

CYCLE_SOLUTION = (
    "class ListNode:\n"
    "    def __init__(self, val=0, nxt=None):\n"
    "        self.val = val\n"
    "        self.next = nxt\n"
    "def detectCycle(head):\n"
    "    seen = set()\n"
    "    while head is not None:\n"
    "        if id(head) in seen:\n"
    "            return head\n"
    "        seen.add(id(head))\n"
    "        head = head.next\n"
    "    return None\n"
)
CYCLE_SETUP = "head = list_to_nodes([3,2,0,-4])\nmake_cycle(head, 1)"
CYCLE_COMPARE = "def _normalize(got):\n    return nodes_repr(got)"

LEVEL_ORDER_SOLUTION = (
    "class TreeNode:\n"
    "    def __init__(self, val=0, left=None, right=None):\n"
    "        self.val = val\n"
    "        self.left = left\n"
    "        self.right = right\n"
    "def levelOrder(root):\n"
    "    ans, cur = [], [root] if root else []\n"
    "    while cur:\n"
    "        ans.append([n.val for n in cur])\n"
    "        nxt = []\n"
    "        for n in cur:\n"
    "            if n.left: nxt.append(n.left)\n"
    "            if n.right: nxt.append(n.right)\n"
    "        cur = nxt\n"
    "    return ans\n"
)
LEVEL_ORDER_SETUP = "root = build_tree([3,9,20,None,None,15,7])"

LRU_SOLUTION = (
    "class LRUCache:\n"
    "    def __init__(self, capacity):\n"
    "        self.cap = capacity\n"
    "        self.data = {}\n"
    "        self.order = []\n"
    "    def _touch(self, key):\n"
    "        self.order.remove(key)\n"
    "        self.order.append(key)\n"
    "    def get(self, key):\n"
    "        if key not in self.data:\n"
    "            return -1\n"
    "        self._touch(key)\n"
    "        return self.data[key]\n"
    "    def put(self, key, value):\n"
    "        if key in self.data:\n"
    "            self.data[key] = value\n"
    "            self._touch(key)\n"
    "            return\n"
    "        if len(self.data) >= self.cap:\n"
    "            del self.data[self.order[0]]\n"
    "            self.order.pop(0)\n"
    "        self.data[key] = value\n"
    "        self.order.append(key)\n"
)
LRU_SETUP = "obj = LRUCache(2)"
LRU_CALL = ("[obj.put(1, 1), obj.put(2, 2), obj.get(1),"
            " obj.put(3, 3), obj.get(2), obj.get(3)]")
LRU_EXPECTED = "[None, None, 1, None, -1, 3]"


@needs_docker
def test_e2e_reverse_list_with_comparator():
    # 样题 1（12 号票格式范例）：链表入参经 setup 构造，输出经比较器规范化
    r = executor.run(REVERSE_LIST_SOLUTION, [{
        "name": "odd", "call": "reverseList(head)",
        "expected": "[5,4,3,2,1]", "setup": REVERSE_LIST_SETUP,
        "compare": REVERSE_LIST_COMPARE,
    }], timeout_ms=5000)
    assert r == {"passed": True, "category": "ok",
                 "duration_ms": r["duration_ms"], "detail": ""}


@needs_docker
def test_e2e_reverse_list_wrong_answer_branch():
    # 比较器的 false 分支：错误解法（只反转一半的替身）→ wrong-answer
    bad = REVERSE_LIST_SOLUTION.replace(
        "    return prev\n",
        "    return head\n", 1)
    r = executor.run(bad, [{
        "name": "odd", "call": "reverseList(head)",
        "expected": "[5,4,3,2,1]", "setup": REVERSE_LIST_SETUP,
        "compare": REVERSE_LIST_COMPARE,
    }], timeout_ms=5000)
    assert r["passed"] is False and r["category"] == "wrong-answer"
    assert r["detail"] == ""  # 规范化串不回传，泄漏面与现行协议一致


@needs_docker
def test_e2e_cycle_identity_compare():
    # 样题 2：带环构造 + 按节点身份的规范化比较（142 环形链表 II）
    r = executor.run(CYCLE_SOLUTION, [{
        "name": "cycle", "call": "detectCycle(head)",
        "expected": "cycle:[2,0,-4]", "setup": CYCLE_SETUP,
        "compare": CYCLE_COMPARE,
    }], timeout_ms=5000)
    assert r["passed"] is True and r["category"] == "ok"

    # 返回错节点（返回头而非入口）→ 身份不同 → 规范化序列不同 → wrong-answer
    wrong = CYCLE_SOLUTION.replace("            return head\n",
                                   "            return head.next\n"
                                   "        return None\n", 1)
    r2 = executor.run(wrong, [{
        "name": "cycle", "call": "detectCycle(head)",
        "expected": "cycle:[2,0,-4]", "setup": CYCLE_SETUP,
        "compare": CYCLE_COMPARE,
    }], timeout_ms=5000)
    assert r2["passed"] is False and r2["category"] == "wrong-answer"


@needs_docker
def test_e2e_tree_construction():
    # 样题 3：树经 setup 的 build_tree 构造；输出为原生列表（repr 即规范化）
    r = executor.run(LEVEL_ORDER_SOLUTION, [{
        "name": "levels", "call": "levelOrder(root)", "expected": "[[3], [9, 20], [15, 7]]",
        "setup": LEVEL_ORDER_SETUP,
    }], timeout_ms=5000)
    assert r["passed"] is True and r["category"] == "ok"


@needs_docker
def test_e2e_stateful_call_sequence():
    # 样题 4（LRU，有状态类）：call 为表达式序列（求值结果是每步返回值的列表），
    # expected 为整段序列的期望 repr——零协议新增，首败语义由宿主串比对承担
    r = executor.run(LRU_SOLUTION, [{
        "name": "seq", "call": LRU_CALL, "expected": LRU_EXPECTED,
        "setup": LRU_SETUP,
    }], timeout_ms=5000)
    assert r["passed"] is True and r["category"] == "ok"

    # 序列中一步结果错 → 整串不符 → wrong-answer（首败即停语义不变）
    bad = LRU_SOLUTION.replace("            return -1\n", "            return -2\n", 1)
    r2 = executor.run(bad, [{
        "name": "seq", "call": LRU_CALL, "expected": LRU_EXPECTED,
        "setup": LRU_SETUP,
    }], timeout_ms=5000)
    assert r2["passed"] is False and r2["category"] == "wrong-answer"


@needs_docker
def test_e2e_setup_crash_maps_to_judge_error():
    # 题作者 setup 运行期崩溃 = 模板缺陷 → judge-error（不罚玩家、可重交）
    r = executor.run("def f():\n    return 1", [{
        "name": "t1", "call": "f()", "expected": "1",
        "setup": "head = list_to_nodes(12345)",
    }], timeout_ms=5000)
    assert r["passed"] is False and r["category"] == "judge-error"


@needs_docker
def test_e2e_compare_crash_maps_to_judge_error():
    # 比较器运行期崩溃 = 模板缺陷 → judge-error（绝不归咎玩家）
    r = executor.run("def f():\n    return 1", [{
        "name": "t1", "call": "f()", "expected": "1",
        "compare": "def _normalize(got):\n    raise RuntimeError('模板坏了')",
    }], timeout_ms=5000)
    assert r["passed"] is False and r["category"] == "judge-error"


@needs_docker
def test_e2e_backward_compat_unchanged():
    # 向后兼容：不声明 setup/compare 的用例走现行 eval/repr，一字不改
    r = executor.run(
        "def two_sum(nums, target):\n"
        "    mp = {}\n"
        "    for i, v in enumerate(nums):\n"
        "        if target - v in mp:\n"
        "            return [mp[target - v], i]\n"
        "        mp[v] = i\n",
        [{"name": "basic", "call": "two_sum([2, 7, 11, 15], 9)", "expected": "[0, 1]"}],
        timeout_ms=5000,
    )
    assert r["passed"] is True and r["category"] == "ok"
