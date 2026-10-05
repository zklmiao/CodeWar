// 首批模板注册表测试 = 模板校验工具（go test 形式，设计 002 §4 机械复核）：
// 四维齐全合法 + 参考解行数↔档位 + 陷阱追溯 + 双形式 + 真实判题冒烟。
package templates

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"codewar/server/internal/match"
	"codewar/server/internal/question"
)

func testRegistry(t *testing.T) []question.Template {
	t.Helper()
	all := All()
	if len(all) < 20 || len(all) > 30 {
		t.Fatalf("首批模板应为 20-30 个，实际 %d", len(all))
	}
	seen := map[string]bool{}
	for _, tm := range all {
		m := tm.Meta()
		if seen[m.ID] {
			t.Fatalf("模板 ID 重复：%s", m.ID)
		}
		seen[m.ID] = true
		if err := question.ValidateMeta(m); err != nil {
			t.Fatalf("meta 非法：%v", err)
		}
	}
	return all
}

// 全模板生成校验（固定种子）：用例数 + 行数 + 陷阱 + 双形式。
func TestAllTemplatesValidate(t *testing.T) {
	for _, tm := range testRegistry(t) {
		in, err := tm.Generate(question.NewRNG(question.DeriveSeed("validate", tm.Meta().ID)))
		if err != nil {
			t.Fatalf("模板 %s 生成失败：%v", tm.Meta().ID, err)
		}
		in.Seed = "validate|" + tm.Meta().ID
		if err := question.ValidateInstance(in); err != nil {
			t.Fatalf("校验失败：%v", err)
		}
	}
}

// 攻击题覆盖 6 题型（每题型 ≥2 模板）。
func TestAttackCoversSixTopics(t *testing.T) {
	byTopic := map[question.Topic]int{}
	for _, tm := range testRegistry(t) {
		m := tm.Meta()
		if m.Action == question.ActionAttack {
			byTopic[m.Topic]++
		}
	}
	for _, tp := range question.AllTopics {
		if byTopic[tp] < 2 {
			t.Fatalf("题型 %s 攻击模板仅 %d 个（要求 ≥2）", tp, byTopic[tp])
		}
	}
}

// 确定性：同 seed 同实例；随机化：异 seed 异实例。
func TestTemplatesDeterministicAndRandom(t *testing.T) {
	for _, tm := range testRegistry(t) {
		m := tm.Meta()
		a, err := tm.Generate(question.NewRNG(question.DeriveSeed("s1", m.ID)))
		if err != nil {
			t.Fatal(err)
		}
		b, err := tm.Generate(question.NewRNG(question.DeriveSeed("s1", m.ID)))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("模板 %s 同 seed 实例不一致", m.ID)
		}
		// 异 seed：5 个种子至少 3 种题面（随机化不退化）。
		stmts := map[string]bool{}
		for i := 0; i < 5; i++ {
			in, err := tm.Generate(question.NewRNG(question.DeriveSeed("seed", m.ID, string(rune('a'+i)))))
			if err != nil {
				t.Fatal(err)
			}
			stmts[in.Statement] = true
		}
		if len(stmts) < 3 {
			t.Fatalf("模板 %s 5 个 seed 仅 %d 种实例，随机化退化", m.ID, len(stmts))
		}
	}
}

// 双形式：both 模板两种渲染共享隐藏用例，挖空 ≤3 核心行。
func TestDualFormRendering(t *testing.T) {
	n := 0
	for _, tm := range testRegistry(t) {
		m := tm.Meta()
		if m.Form != question.FormBoth {
			continue
		}
		n++
		in, err := tm.Generate(question.NewRNG(question.DeriveSeed("form", m.ID)))
		if err != nil {
			t.Fatal(err)
		}
		c, f := in.ToQuestion(question.FormComplete), in.ToQuestion(question.FormFillBlank)
		if len(c.Tests) != len(f.Tests) {
			t.Fatalf("模板 %s 双形式用例不一致", m.ID)
		}
		if c.StarterCode == f.StarterCode {
			t.Fatalf("模板 %s 双形式起始代码无区别", m.ID)
		}
		if in.BlankLines < 1 || in.BlankLines > 3 || !in.BlankIsCore {
			t.Fatalf("模板 %s 挖空不合规：%d 行 core=%v", m.ID, in.BlankLines, in.BlankIsCore)
		}
	}
	if n == 0 {
		t.Fatal("攻击题应有 both 双渲染模板")
	}
	t.Logf("双渲染模板 %d 个", n)
}

// 冒烟：每个模板参考解跑自家全部用例（真实 executor，不走 gRPC）。
// 理由：gRPC 需起服务进程，测试只关心“参考解正确”，子进程直调 executor.run
// 与生产判题同代码路径；CI 无 python 时跳过，本地必须全过。
func TestSmokeAnswerKeyPassesOwnTests(t *testing.T) {
	judgeDir := filepath.Join("..", "..", "..", "..", "judge")
	if _, err := os.Stat(filepath.Join(judgeDir, "judgesvc", "executor.py")); err != nil {
		t.Skipf("无 judge 目录，跳过冒烟：%v", err)
	}
	py := "python"
	if _, err := exec.LookPath(py); err != nil {
		py = "python3"
		if _, err := exec.LookPath(py); err != nil {
			t.Skip("无 python，跳过冒烟")
		}
	}
	for _, tm := range testRegistry(t) {
		for _, seed := range []string{"smoke-1", "smoke-2"} {
			in, err := tm.Generate(question.NewRNG(question.DeriveSeed(seed, tm.Meta().ID)))
			if err != nil {
				t.Fatal(err)
			}
			if err := runExecutor(py, judgeDir, in.AnswerKey, in.Tests()); err != nil {
				t.Fatalf("模板 %s seed %s 参考解未通过自家用例：%v", tm.Meta().ID, seed, err)
			}
		}
	}
}

// runExecutor：子进程直调 judgesvc.executor.run（与 judge 服务同函数）。
func runExecutor(py, judgeDir, code string, tests []match.TestCase) error {
	triples := make([]map[string]string, len(tests))
	for i, tc := range tests {
		triples[i] = map[string]string{"name": tc.Name, "call": tc.Call, "expected": tc.Expected}
	}
	payload, _ := json.Marshal(map[string]any{"code": code, "cases": triples})
	script := `import json, sys
sys.path.insert(0, sys.argv[1])
from judgesvc.executor import run
p = json.loads(sys.argv[2])
r = run(p["code"], p["cases"], timeout_ms=8000)
print(json.dumps(r))
if not (r["passed"] and r["category"] == "ok"):
    raise SystemExit(1)
`
	cmd := exec.Command(py, "-c", script, judgeDir, string(payload))
	if out, err := cmd.CombinedOutput(); err != nil {
		return &smokeError{msg: string(out)}
	}
	return nil
}

type smokeError struct{ msg string }

func (e *smokeError) Error() string { return e.msg }
