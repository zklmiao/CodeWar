#!/usr/bin/env bash
# 01 正常判题：正确代码两用例全过 → passed=True, category=ok
docker exec deploy-judge-1 python -c '
from judgesvc import executor
r = executor.run(
    "def is_palindrome(s: str) -> bool:\n    return s == s[::-1]",
    [{"name": "t1", "call": "is_palindrome(\"level\")", "expected": "True"},
     {"name": "t2", "call": "is_palindrome(\"abba\")", "expected": "True"}],
    timeout_ms=5000,
)
print(r)
'
