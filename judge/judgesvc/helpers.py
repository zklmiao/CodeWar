# -*- coding: utf-8 -*-
"""沙箱构造辅助库（15 号票引入，26 号票抽出为可导入模块）。

executor 经 inspect.getsource 取本文件源码随 payload 注入容器（worker 落盘
/tmp/_helpers.py 后 `import _helpers` 加载）；测试进程直接 import 本模块
（与容器内注入源码同源：executor._HELPERS 即 getsource(本模块)）。
本模块必须自包含——只允许标准库 import，容器内没有 judgesvc 可导。

所有遍历对"玩家可能造出的环"有界（visited 按 id 判定 + 纵深上限）；
规范化 repr 采用紧凑风格（"[1,2,3]"），题作者据同一风格书写 expected。
"""
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
