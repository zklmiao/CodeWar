#!/usr/bin/env bash
# 03 联网尝试 → 容器 --network=none 断网，连接即失败：
# passed=False（category 落在 wrong-answer/runtime-error 均符合契约，关键是不为 ok）
docker exec deploy-judge-1 python -c '
from judgesvc import executor
r = executor.run(
    "import socket\nsocket.socket().connect((\"8.8.8.8\", 53))",
    [{"name": "t1", "call": "f()", "expected": "None"}],
    timeout_ms=3000,
)
print(r)
'
