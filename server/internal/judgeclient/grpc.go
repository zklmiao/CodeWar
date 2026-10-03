// Package judgeclient：JudgePort 的 gRPC 生产实现（ADR-0001：judge 为独立微服务）。
package judgeclient

import (
	"context"
	"time"

	"codewar/server/gen/judgepb"
	"codewar/server/internal/match"
)

type GRPCJudge struct {
	client    judgepb.JudgeServiceClient
	timeoutMs int32
}

// NewGRPCJudge：defaultTimeoutMs 为单次判题的执行超时预算（02 号票硬化前简易值）。
func NewGRPCJudge(client judgepb.JudgeServiceClient, defaultTimeoutMs int32) *GRPCJudge {
	if defaultTimeoutMs <= 0 {
		defaultTimeoutMs = 8000
	}
	return &GRPCJudge{client: client, timeoutMs: defaultTimeoutMs}
}

func (g *GRPCJudge) Judge(ctx context.Context, code string, tests []match.TestCase) match.JudgeResult {
	cases := make([]*judgepb.TestCase, len(tests))
	for i, t := range tests {
		cases[i] = &judgepb.TestCase{Name: t.Name, Call: t.Call, Expected: t.Expected}
	}
	// RPC 预算 = 执行超时 × 2（留出传输与排队余量）；判题失败即 judge-error，引擎不罚不锁
	cctx, cancel := context.WithTimeout(ctx, time.Duration(g.timeoutMs)*time.Millisecond*2)
	defer cancel()
	v, err := g.client.Judge(cctx, &judgepb.JudgeRequest{
		Code: code, TestCases: cases, TimeoutMs: g.timeoutMs,
	})
	if err != nil {
		return match.JudgeResult{Passed: false, Category: match.CategoryJudgeError, Detail: err.Error()}
	}
	return match.JudgeResult{Passed: v.GetPassed(), Category: v.GetCategory(), DurationMs: v.GetDurationMs(), Detail: v.GetDetail()}
}
