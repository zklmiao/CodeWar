#!/usr/bin/env bash
# 07 服务存活：连发两轮死循环后立刻正常判题——第二段必须仍为 ok
# （恶意提交不垮服务、判题管线无污染的直接证据）

echo "== 第一轮死循环 =="
docker exec deploy-judge-1 python -c '
from judgesvc import executor
r = executor.run(
    "def is_palindrome(s: str) -> bool:\n    while True: pass",
    [{"name": "t1", "call": "is_palindrome(\"level\")", "expected": "True"}],
    timeout_ms=3000,
)
print(r)
'

echo "== 第二轮死循环 =="
docker exec deploy-judge-1 python -c '
from judgesvc import executor
r = executor.run(
    "def is_palindrome(s: str) -> bool:\n    while True: pass",
    [{"name": "t1", "call": "is_palindrome(\"level\")", "expected": "True"}],
    timeout_ms=3000,
)
print(r)
'

echo "== 紧接着正常判题（必须 ok）=="
docker exec deploy-judge-1 python -c '
from judgesvc import executor
r = executor.run(
    "def is_palindrome(s: str) -> bool:\n    return s == s[::-1]",
    [{"name": "t1", "call": "is_palindrome(\"level\")", "expected": "True"}],
    timeout_ms=5000,
)
print(r)
'
