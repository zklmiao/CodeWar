#!/usr/bin/env bash
# 05 部分通过 = 判失败，且 detail 必须为空（不泄漏通过计数/用例信息）：
# t1 期望正确（通过），t2 期望值故意错（mismatch）→ wrong-answer
docker exec deploy-judge-1 python -c '
from judgesvc import executor
r = executor.run(
    "def is_palindrome(s: str) -> bool:\n    return s == s[::-1]",
    [{"name": "t1", "call": "is_palindrome(\"level\")", "expected": "True"},
     {"name": "t2", "call": "is_palindrome(\"hello\")", "expected": "True"}],
    timeout_ms=5000,
)
print(r)
'
