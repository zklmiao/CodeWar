#!/usr/bin/env bash
# 08 fork 炸弹 → 容器级 pids-limit=64 配额杀：passed=False, category=limit-exceeded
# 防线不在 worker 内（root 豁免 NPROC），而在 --pids-limit：fork 满 64 进程后
# EAGAIN(errno 11) 由 harness 归因；宿主受容器配额保护，不垮。
docker exec deploy-judge-1 python -c '
from judgesvc import executor
r = executor.run(
    "import os\ndef two_sum(nums, target):\n    for _ in range(100000):\n        os.fork()\n    return []",
    [{"name": "basic", "call": "two_sum([2, 7, 11, 15], 9)", "expected": "[0, 1]"}],
    timeout_ms=5000,
)
print(r)
'
