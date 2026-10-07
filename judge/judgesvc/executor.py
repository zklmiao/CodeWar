# -*- coding: utf-8 -*-
"""沙箱判题执行器（02 号票：一次性容器，docker-out-of-docker）。

执行模型（ADR-0003）：每次提交启动一个一次性容器执行全部用例；
容器内 harness（经 stdin 注入）为每个用例 fork 一个 worker 子进程并施加
每用例超时，首败即停；容器整体另有墙钟上限（第二道保险）。

隔离与防泄漏（容器级，见 ADR-0003）：
--network=none 断网；--memory=256m + --pids-limit=64 + --cpus=0.5 配额；
--cap-drop=ALL；--read-only 只读根文件系统 + PYTHONDONTWRITEBYTECODE=1。
**零宿主路径挂载**：玩家代码、worker 脚本与 call 表达式清单全部经 stdin
注入（docker-out-of-docker 下 -v 的路径由宿主 daemon 解析，judge 容器内
的临时目录在宿主上不存在——挂载方案在嵌套形态下必然失败，浏览器验收
发现）；期望值是秘密，一分钟都不进容器：repr 与 expected 的比对、
category 汇总全部在宿主侧完成（worker 只能见到自己的输入，见不到任何
期望值）。

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
import time

# judge 侧每用例超时常量（ADR-0003）：调用方传更小值时兼容采用，默认 3 秒
PER_TEST_TIMEOUT_MS = 3000
# 容器整体墙钟 slack（秒）：用例数×每用例超时 + 本值，宿主侧施加
WALL_CLOCK_SLACK_S = 30
# 沙箱镜像（固定版本，compose/Dockerfile 预拉取；环境变量仅供测试覆盖）
SANDBOX_IMAGE = os.environ.get("CODEWAR_SANDBOX_IMAGE", "python:3.12-slim")

# syntax detail 白名单：只接受 harness 按格式拼出的行号摘要
_SYNTAX_DETAIL_RE = re.compile(r"^line \d+: .{1,200}$")

# 采集整体输出后兜底等待上限（秒）：外层墙钟熔断 kill 后再收已 flush 的行
_DRAIN_TIMEOUT_S = 5


def _judge_error(duration_ms: int = 0) -> dict:
    return {"passed": False, "category": "judge-error",
            "duration_ms": duration_ms, "detail": ""}


def docker_available() -> bool:
    """宿主 docker CLI 是否存在（判题 fail-closed 快速判断，不探针 daemon）。"""
    return shutil.which("docker") is not None


def docker_ready() -> bool:
    """CLI 存在且 daemon 可达——仅测试门控用（收集期一次 `docker info`）。

    区别于 docker_available：CLI 在而 daemon 宕（Docker Desktop 重启窗口/
    服务挂掉）时，沙箱判题只会 fail-closed 返回 judge-error，测试必然失败。
    该形态下应 skip 而非误报 19 连挂（本轮回归审查第二次踩中）。
    """
    if not docker_available():
        return False
    try:
        proc = subprocess.run(
            ["docker", "info", "--format", "{{.ServerVersion}}"],
            capture_output=True, timeout=10,
        )
        return proc.returncode == 0
    except (OSError, subprocess.TimeoutExpired):
        return False


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


def build_stdin_payload(code: str, test_cases: list[dict], worker_src: str,
                        helpers_src: str) -> str:
    """构造进容器的 stdin 载荷：解法源码 + worker 脚本 + 构造辅助库 + 用例清单。

    只含用例名、call 表达式与题作者 setup/compare 声明，不含期望值；worker
    与辅助库脚本经 stdin 注入而非挂载（零宿主路径依赖，见模块 docstring）。
    """
    return json.dumps({
        "solution": code or "",
        "worker": worker_src,
        "helpers": helpers_src,
        "calls": [{"name": t["name"], "call": t["call"],
                   "setup": t.get("setup", ""), "compare": t.get("compare", "")}
                  for t in test_cases],
    })


def build_sandbox_cmd(per_test_ms: int, harness_src: str) -> list[str]:
    """构造沙箱容器命令（纯函数，供契约测试断言）。

    禁止出现任何 -v/--mount：嵌套容器解析不了宿主路径（浏览器验收发现）；
    解法与脚本经 stdin 注入，运行期文件全部写入 /tmp tmpfs。harness 本体
    经 `python -c` 加载——stdin 必须完整留给 payload（`python -` 会把
    stdin 当脚本吃掉）。
    """
    return [
        "docker", "run", "--rm", "-i",
        "--network=none",
        "--memory=256m",
        "--pids-limit=64",
        "--cpus=0.5",
        "--cap-drop=ALL",
        "--read-only",
        "--tmpfs", "/tmp:size=16m,exec",
        "-e", "PYTHONDONTWRITEBYTECODE=1",
        SANDBOX_IMAGE,
        "python", "-c", harness_src, str(per_test_ms),
    ]


# ---- 沙箱构造辅助库（15 号票）：随 payload 注入容器，题作者 setup/compare 可用 ----
# 所有遍历对"玩家可能造出的环"有界（visited 按 id 判定 + 纵深上限）；
# 规范化 repr 采用紧凑风格（"[1,2,3]"），题作者据同一风格书写 expected。
_HELPERS = r'''
import json

class ListNode:
    def __init__(self, val=0, nxt=None):
        self.val = val
        self.next = nxt

def list_to_nodes(arr):
    dummy = ListNode()
    cur = dummy
    for v in arr:
        cur.next = ListNode(v)
        cur = cur.next
    return dummy.next

def make_cycle(head, pos):
    """尾节点.next 指向第 pos 个节点（0 起）；pos<0 或空链 = 无环。"""
    if pos is None or pos < 0 or head is None:
        return head
    entry = head
    for _ in range(pos):
        entry = entry.next
    tail = head
    while tail.next is not None:
        tail = tail.next
    tail.next = entry
    return head

def _walk_bounded(node, limit=100000):
    """沿 next 行走，返回 (值序列, 环入口在序列中的下标或 None)。
    visited 按 id 判定——环有界；limit 为纵深防御。"""
    seen = {}
    vals = []
    cur = node
    while cur is not None and id(cur) not in seen and len(vals) < limit:
        seen[id(cur)] = len(vals)
        vals.append(cur.val)
        cur = cur.next
    entry = seen.get(id(cur)) if cur is not None else None
    return vals, entry

def nodes_repr(node):
    """规范化 repr：None→"None"；无环→"[1,2,3]"；node 为环入口→
    "cycle:[2,0,-4,3]"（从 node 出发走到 id 回到 node 为止——对唯一值
    测试数据即节点身份的规范化表达）。"""
    if node is None:
        return "None"
    vals, entry = _walk_bounded(node)
    if entry is None:
        return "[" + ",".join(map(str, vals)) + "]"
    return "cycle:[" + ",".join(map(str, vals)) + "]"

class TreeNode:
    def __init__(self, val=0, left=None, right=None):
        self.val = val
        self.left = left
        self.right = right

def build_tree(arr):
    """层序数组（None=空位）构造二叉树，返回根或 None。"""
    if not arr or arr[0] is None:
        return None
    root = TreeNode(arr[0])
    queue = [root]
    i = 1
    while queue and i < len(arr):
        node = queue.pop(0)
        if i < len(arr):
            v = arr[i]
            i += 1
            if v is not None:
                node.left = TreeNode(v)
                queue.append(node.left)
        if i < len(arr):
            v = arr[i]
            i += 1
            if v is not None:
                node.right = TreeNode(v)
                queue.append(node.right)
    return root

def tree_repr(root, limit=100000):
    """规范化 repr：层序数组含 None 占位（尾部裁剪），紧凑风格；节点总数
    以 limit 限界（树本无环，防玩家造环导致 BFS 不终止）。"""
    out = []
    queue = [root]
    while queue and len(out) < limit:
        node = queue.pop(0)
        if node is None:
            out.append(None)
            continue
        out.append(node.val)
        queue.append(node.left)
        queue.append(node.right)
    while out and out[-1] is None:
        out.pop()
    return json.dumps(out, separators=(",", ":"))

EXPORTS = {"ListNode": ListNode, "list_to_nodes": list_to_nodes,
           "make_cycle": make_cycle, "nodes_repr": nodes_repr,
           "TreeNode": TreeNode, "build_tree": build_tree,
           "tree_repr": tree_repr}
'''

# ---- 题作者代码静态卫兵（纵深防御；真实边界是沙箱本身） ----
_AUTHOR_CODE_MAX_LEN = 4000
_AUTHOR_DENYLIST = re.compile(
    r"__import__|open\s*\(|os\.system|subprocess|shutil|importlib|eval\s*\(|exec\s*\(")


def guard_author_code(src: str, what: str) -> None:
    """setup/compare 是题作者代码（可信侧），但注入沙箱前仍过静态卫兵：
    长度上限、禁用调用清单、可编译。违规抛 ValueError → judge-error
    （模板缺陷属基础设施故障，不罚玩家）。"""
    if len(src) > _AUTHOR_CODE_MAX_LEN:
        raise ValueError(what + " 超过长度上限 " + str(_AUTHOR_CODE_MAX_LEN))
    if src.strip() and _AUTHOR_DENYLIST.search(src):
        raise ValueError(what + " 含禁用调用")
    compile(src, what, "exec")


# ---- 容器内 harness：父进程（逐用例 fork worker，首败即停） ----
# 程序本体经 `python -c` 加载（stdin 完整留给 payload）；stdin JSON 携带
# 解法源码 + worker 脚本 + call 清单（无期望值）；harness 把解法与 worker
# 写入 /tmp tmpfs，每行输出一个用例结论；正常结束（即使首败即停）必以
# summary 行收尾；
# 全程 exit 0（非零退出一律视为 infra 故障）。
_HARNESS = (
    "import json, os, subprocess, sys, time\n"
    + inspect.getsource(_parse_json_lines)
    + r'''
payload = json.load(sys.stdin)
per_test_ms = int(sys.argv[1])
per_test_s = per_test_ms / 1000.0
t0 = time.monotonic()

def emit(obj):
    print(json.dumps(obj, ensure_ascii=False))
    sys.stdout.flush()

def summary():
    emit({"i": "done", "duration_ms": int((time.monotonic() - t0) * 1000)})

# 注入文件到 /tmp tmpfs（exec 开启）：解法 + worker + 构造辅助库，零挂载
with open("/tmp/solution.py", "w", encoding="utf-8") as f:
    f.write(payload["solution"])
with open("/tmp/_worker.py", "w", encoding="utf-8") as f:
    f.write(payload["worker"])
with open("/tmp/_helpers.py", "w", encoding="utf-8") as f:
    f.write(payload["helpers"])

# 静态语法预检：compile 不执行代码，安全；行号信息无害可回传
try:
    compile(payload["solution"], "solution.py", "exec")
except SyntaxError as e:
    emit({"i": -1, "status": "syntax",
          "detail": "line {}: {}".format(e.lineno, e.msg)[:210]})
    summary()
    raise SystemExit(0)
except (OSError, ValueError):
    emit({"i": -1, "status": "runtime-error"})
    summary()
    raise SystemExit(0)

calls = payload["calls"]
for idx, c in enumerate(calls):
    try:
        proc = subprocess.run(
            [sys.executable, "/tmp/_worker.py"],
            input=json.dumps({"i": idx, "call": c["call"],
                              "setup": c.get("setup", ""),
                              "compare": c.get("compare", "")}),
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
    if outcome == "infra-error":
        # 15 号票：构造辅助库/题作者 setup/比较器崩溃=基础设施故障，不罚玩家
        emit({"i": idx, "status": "judge-error"})
        break
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
# 脚本源码由 harness 经 stdin 注入的 payload 落到 /tmp；stdin 只收自己的
# {"i", "call", "setup", "compare"}；全程见不到期望值（__main__ 命名空间里
# 就没有）。15 号票 v2 流程：import 玩家代码 → NS 并入构造辅助库 → exec
# setup（题作者声明，失败=infra 不罚玩家）→ eval(call) → 若声明 compare
# 则用其 _normalize 把 got 规范化为 repr 字符串回传（值比对在宿主）。
_WORKER = r'''
import io, json, sys

job = json.load(sys.stdin)
idx = job["i"]
call = job["call"]
setup_src = job.get("setup", "")
compare_src = job.get("compare", "")

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

# 构造辅助库：harness 已落盘 /tmp/_helpers.py。失败=基础设施故障（非玩家
# 代码问题），按 infra-error 归因，harness 转判 judge-error。
sys.path.insert(0, "/tmp")
try:
    import _helpers as _h
except Exception:
    emit({"i": idx, "outcome": "infra-error"})
    raise SystemExit(0)

# 玩家代码的打印/报错截留在内存中，绝不进 worker 协议流（防协议污染与信息泄漏）
buf_out, buf_err = io.StringIO(), io.StringIO()
old_out, old_err = sys.stdout, sys.stderr
sys.stdout, sys.stderr = buf_out, buf_err
result = None
try:
    try:
        import solution  # noqa: E402  玩家代码（每 worker 全新 import）
    except BaseException as e:
        result = {"i": idx, "outcome": "import-error",
                  "err": type(e).__name__, "errno": getattr(e, "errno", None)}
    else:
        NS = dict(vars(solution))  # 玩家模块命名空间（含其自定义辅助函数）
        # 构造辅助库只补缺、不覆盖：玩家自定义同名符号（如自建 ListNode /
        # list_to_nodes）优先——题面 call 是按玩家自己的实现写的（review 发现）。
        for _k, _v in _h.EXPORTS.items():
            NS.setdefault(_k, _v)
        if setup_src:
            try:
                exec(setup_src, NS)  # 题作者构造声明：为 call 绑定结构体入参
            except BaseException:
                # setup 失败=题作者代码缺陷（模板问题），infra 不罚玩家
                result = {"i": idx, "outcome": "infra-error"}
        if result is None:
            try:
                got = eval(call, dict(NS))
            except BaseException as e:
                result = {"i": idx, "outcome": "call-error",
                          "err": type(e).__name__, "errno": getattr(e, "errno", None)}
            if result is None:
                if compare_src:
                    try:
                        cns = dict(_h.EXPORTS)
                        exec(compare_src, cns)
                        nr = cns["_normalize"](got)
                        if not isinstance(nr, str):
                            nr = repr(nr)
                        result = {"i": idx, "outcome": "ok", "repr": nr}
                    except BaseException:
                        # 比较器/规范化崩溃=题作者代码缺陷（infra），绝不归咎玩家
                        result = {"i": idx, "outcome": "infra-error"}
                else:
                    result = {"i": idx, "outcome": "ok", "repr": repr(got)}
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

    neg = by_i.get(-1)
    if isinstance(neg, dict) and neg.get("status") == "syntax":
        detail = neg.get("detail", "")
        if not isinstance(detail, str) or not _SYNTAX_DETAIL_RE.match(detail):
            detail = ""
        return verdict("syntax", detail)
    if isinstance(neg, dict) and neg.get("status") == "runtime-error":
        # harness 前置故障（宿主读 solution.py 失败等）：发生在任何玩家代码
        # 执行之前，属基础设施故障——按 spec 的 judge-error 语义不罚不锁可重交
        return verdict("judge-error")

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
        if status == "judge-error":
            return verdict("judge-error")  # harness 转发的 infra 故障（辅助库/setup/比较器崩溃）
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
    # 题作者 setup/compare 静态卫兵：违规（超长/禁用调用/语法错误）=模板缺陷
    # → judge-error，不注入容器（纵深防御；真实边界是沙箱本身，见 ADR-0004）。
    for t in test_cases:
        for key in ("setup", "compare"):
            src = t.get(key, "")
            if src:
                try:
                    guard_author_code(src, key)
                except Exception:
                    return _judge_error(0)
    # 零挂载：解法源码 + worker 脚本 + 构造辅助库 + 用例清单（仅 call，无期
    # 望值）全部经 stdin 注入；运行期文件只落在容器内 /tmp tmpfs——嵌套容器
    # 解析不了宿主路径，任何 -v 挂载都会静默拿到空目录（浏览器验收发现的 P1）。
    payload = build_stdin_payload(code, test_cases, _WORKER, _HELPERS)
    cmd = build_sandbox_cmd(per_test_ms, _HARNESS)

    started = time.monotonic()
    try:
        proc = subprocess.Popen(
            cmd, stdin=subprocess.PIPE,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
        )
    except OSError:
        # docker 二进制在可用性检查后消失等竞态：infra 故障，不裸抛
        return _judge_error(0)
    try:
        out, _ = proc.communicate(input=payload, timeout=outer_s)
    except subprocess.TimeoutExpired:
        # 整体墙钟熔断（第二道保险）：kill 后收已 flush 的行，已完成的
        # 用例按其结论计入，未完成的在汇总时判 timeout。
        proc.kill()
        try:
            out, _ = proc.communicate(timeout=_DRAIN_TIMEOUT_S)
        except subprocess.TimeoutExpired:
            out = ""
    except OSError:
        # docker 调用失败（无 daemon 等 infra 故障）
        return _judge_error(int((time.monotonic() - started) * 1000))

    elapsed_ms = int((time.monotonic() - started) * 1000)
    lines = _parse_json_lines(out)
    if not lines:
        # 空输出 = 容器未产出任何结论行（daemon 故障/镜像缺失等）→ infra 故障计
        return _judge_error(elapsed_ms)
    return _assemble_verdict(lines, test_cases, elapsed_ms)
