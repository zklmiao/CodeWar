#!/usr/bin/env bash
# 06 语法错误 → syntax + 行号 detail（唯一允许携带 detail 的类别）
docker exec deploy-judge-1 python -c '
from judgesvc import executor
r = executor.run(
    "def is_palindrome(s: str) -> bool:\n    return s === s",
    [{"name": "t1", "call": "is_palindrome(\"level\")", "expected": "True"}],
    timeout_ms=5000,
)
print(r)
'
