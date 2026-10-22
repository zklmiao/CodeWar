package bot

// 真 judge 冒烟（09 号票）：参考解/错误分支走与生产同代码路径的 Python
// executor（judgesvc.executor.run），证明 AnswerKey 真能通过判题，不是只在
// 假判题器下成立。
//
// 门控与 judge 包测试一致：CLI 在而 docker daemon 宕时 executor 只会
// fail-closed 返回 judge-error，此时应 skip 而非误报；CI 无 python/judge
// 目录时同样跳过。冒烟只做一次"参考解通过 + starter 失败"，不再跑整局
// （整局链路由假判题器的引擎 seam 集成测试覆盖；真判题只需证明参考答案
// 与真实执行器兼容）。

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"codewar/server/internal/match"
	"codewar/server/internal/question"
	"codewar/server/internal/question/templates"
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

// dockerReady：与 judge/tests 的 skip 门控同义（CLI + daemon 都可用的探针）。
func dockerReady(t *testing.T, py, judgeDir string) bool {
	t.Helper()
	script := `import sys
sys.path.insert(0, sys.argv[1])
from judgesvc.executor import docker_ready
raise SystemExit(0 if docker_ready() else 1)
`
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, py, "-c", script, judgeDir).Run() == nil
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
	if !dockerReady(t, py, judgeDir) {
		t.Skip("docker daemon 不可用，跳过真判题冒烟（与 judge 测试同一门控）")
	}

	factory := question.NewFactory(question.Config{
		MaxDifficulty: question.L3,
		Strategy:      question.StrategyClimb,
		MatchSeed:     "real-judge-smoke",
	})
	factory.RegisterAll(templates.All()...)
	q := factory.AttackQuestion(1)
	if q.AnswerKey == "" {
		t.Fatalf("攻击题没有参考解：%s", q.ID)
	}

	j := realJudge{py: py, judgeDir: judgeDir}
	pass := j.Judge(context.Background(), q.AnswerKey, q.Tests)
	if !pass.Passed || pass.Category != match.CategoryOK {
		t.Fatalf("参考解未通过真判题：%+v", pass)
	}
	bad := j.Judge(context.Background(), q.StarterCode, q.Tests)
	if bad.Passed {
		t.Fatal("starter 不应通过真判题")
	}
	if bad.Category == match.CategoryJudgeError {
		t.Fatalf("判题基础设施错误（docker 门控应已跳过）：%+v", bad)
	}
	t.Logf("真判题冒烟：%s 参考解通过（%dms），starter 判 %s", q.ID, pass.DurationMs, bad.Category)
}
