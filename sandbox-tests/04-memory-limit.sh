#!/usr/bin/env bash
# 04 内存炸弹：300MB 字符串分配撞 worker 的 RLIMIT_AS 256MB → MemoryError
# 被 harness 归因为配额杀：passed=False, category=limit-exceeded
docker exec deploy-judge-1 python -c '
from judgesvc import executor
r = executor.run(
    "x = \"x\" * (300 << 20)",
    [{"name": "t1", "call": "x", "expected": "x"}],
    timeout_ms=3000,
)
print(r)
'
