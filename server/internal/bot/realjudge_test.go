package bot

// 真 judge 冒烟（09 号票）：Bot 的参考解/错误分支走与生产同代码路径的
// Python executor（judgesvc.executor.run），证明 AnswerKey 真能通过判题，
// 不是只在假判题器下成立。CI 无 python/judge 目录时跳过。

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"codewar/server/internal/match"
)

type realJudge struct {
	py       string
	judgeDir string
}

func (j realJudge) Judge(_ context.Context, code string, tests []match.TestCase) match.JudgeResult {
	triples := make([]map[string]string, len(tests))
	for i, tc := range tests {
		triples[i] = map[string]string{"name": tc.Name, "call": tc.Call, "expected": tc.Expected}
	}
	payload, _ := json.Marshal(map[string]any{"code": code, "cases": triples})
	script := `import json, sys
sys.path.insert(0, sys.argv[1])
from judgesvc.executor import run
p = json.loads(sys.argv[2])
print(json.dumps(run(p["code"], p["cases"], timeout_ms=8000)))
`
	out, err := exec.Command(j.py, "-c", script, j.judgeDir, string(payload)).Output()
	if err != nil {
		return match.JudgeResult{Passed: false, Category: match.CategoryJudgeError, Detail: err.Error()}
	}
	var r struct {
		Passed   bool   `json:"passed"`
		Category string `json:"category"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		return match.JudgeResult{Passed: false, Category: match.CategoryJudgeError, Detail: string(out)}
	}
	return match.JudgeResult{Passed: r.Passed, Category: r.Category}
}

func TestRealJudgeSmoke(t *testing.T) {
	judgeDir := filepath.Join("..", "..", "..", "judge")
	if _, err := os.Stat(filepath.Join(judgeDir, "judgesvc", "executor.py")); err != nil {
		t.Skipf("无 judge 目录，跳过真判题冒烟：%v", err)
	}
	py := "python"
	if _, err := exec.LookPath(py); err != nil {
		py = "python3"
		if _, err := exec.LookPath(py); err != nil {
			t.Skip("无 python，跳过真判题冒烟")
		}
	}

	h := newMatchHarnessWithJudge(t, "real-judge-smoke", Hard, realJudge{py: py, judgeDir: judgeDir})
	h.play(30 * time.Second)

	if h.ended == nil {
		t.Fatalf("真判题整局未终局（events=%d）", len(h.events))
	}
	if h.botPass == 0 {
		var cats []string
		for _, env := range h.events {
			if j := env.GetSubmissionJudged(); j != nil && j.GetPlayer() == match.P2 {
				cats = append(cats, j.GetCategory())
			}
		}
		t.Fatalf("Bot 参考解未通过真判题；判题类别序列：%s", strings.Join(cats, ","))
	}
	if h.locked != 0 || h.rejected != 0 {
		t.Fatalf("Bot 出现锁定/拒绝提交：locked=%d rejected=%d", h.locked, h.rejected)
	}
	t.Logf("真判题冒烟：Bot 通过 %d / 失败 %d，终局 winner=%q reason=%q",
		h.botPass, h.botFail, h.ended.GetWinner(), h.ended.GetReason())
}
