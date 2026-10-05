#!/usr/bin/env bash
# 02 死循环 → 每用例 3 秒时钟杀：passed=False, category=timeout, duration_ms≈3000
docker exec deploy-judge-1 python -c '
from judgesvc import executor
r = executor.run(
    "def is_palindrome(s: str) -> bool:\n    while True: pass",
    [{"name": "t1", "call": "is_palindrome(\"level\")", "expected": "True"}],
    timeout_ms=3000,
)
print(r)
'
