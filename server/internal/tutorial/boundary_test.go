package tutorial

// 10 号票验收 3：教程无平行实现——引擎包零 tutorial 引用，
// 且教程题包直接插在同一 *match.Engine 的标准事件流上（RoundStarted/PanelReady）。

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/match"
)

type passJudge struct{}

func (passJudge) Judge(context.Context, string, []match.TestCase) match.JudgeResult {
	return match.JudgeResult{Passed: true, Category: match.CategoryOK}
}

func TestMatchPackageHasNoTutorialReferences(t *testing.T) {
	dir := filepath.Join("..", "match")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取 match 包目录失败：%v", err)
	}
	var hits []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("读取 %s 失败：%v", e.Name(), err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			// "tutorial" 不是任何常见单词的子串，无需词边界。
			if strings.Contains(strings.ToLower(line), "tutorial") {
				hits = append(hits, filepath.Join(e.Name())+":"+strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
			}
		}
	}
	if len(hits) > 0 {
		t.Fatalf("引擎包不得出现 tutorial 引用（教程必须零侵入引擎）：\n%s", strings.Join(hits, "\n"))
	}
}

func TestTutorialRunsOnSameMatchEngine(t *testing.T) {
	bank, err := NewBank(1)
	if err != nil {
		t.Fatal(err)
	}
	var events []*matchpb.Envelope
	eng := match.NewEngineWithSetup(match.RealClock{}, bank, passJudge{}, func(env *matchpb.Envelope) {
		events = append(events, env)
	}, match.Setup{P1Class: "warrior", P2Class: "warrior", RiskTier: match.DefaultRiskTier})
	// 同一引擎实例类型（教程与 PvP/PvE 共用 MatchEngine，无平行实现）。
	var _ *match.Engine = eng
	eng.StartMatch()
	eng.Stop()

	if len(events) < 2 {
		t.Fatalf("教程题包接入后应发布标准事件流，实际 %d 条", len(events))
	}
	if events[0].GetRoundStarted() == nil || events[0].GetRoundStarted().GetRound() != 1 {
		t.Fatalf("首条事件应为第 1 回合开始：%v", events[0])
	}
	if pr := events[1].GetPanelReady(); pr == nil || pr.GetAttack().GetId() == "" {
		t.Fatalf("第二条事件应为含固定题包的面板：%v", events[1])
	}
}
