# -*- coding: utf-8 -*-
"""judge gRPC 服务入口：python -m judgesvc.server"""
from __future__ import annotations

import logging
import os
import sys
from concurrent import futures

import grpc

# 生成的 pb 在 judge/gen 下，按包路径导入
sys.path.insert(0, os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "gen"))

from codewar.judge.v1 import judge_pb2, judge_pb2_grpc  # noqa: E402
from judgesvc import executor  # noqa: E402

_LOG = logging.getLogger("judge")


class JudgeService(judge_pb2_grpc.JudgeServiceServicer):
    def Judge(self, request, context):
        cases = [{"name": t.name, "call": t.call, "expected": t.expected} for t in request.test_cases]
        timeout_ms = request.timeout_ms if request.timeout_ms > 0 else 8000
        r = executor.run(request.code, cases, timeout_ms)
        _LOG.info("judged: category=%s passed=%s duration_ms=%s", r["category"], r["passed"], r["duration_ms"])
        return judge_pb2.JudgeVerdict(
            passed=r["passed"],
            category=r["category"],
            duration_ms=r["duration_ms"],
            detail=r["detail"],
        )


def main():
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(name)s %(levelname)s %(message)s")
    port = os.environ.get("JUDGE_PORT", "50051")
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    judge_pb2_grpc.add_JudgeServiceServicer_to_server(JudgeService(), server)
    server.add_insecure_port(f"[::]:{port}")
    server.start()
    _LOG.info("judge gRPC 服务启动 :%s", port)
    server.wait_for_termination()


if __name__ == "__main__":
    main()
