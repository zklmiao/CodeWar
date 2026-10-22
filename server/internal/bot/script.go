package bot

import (
	"fmt"

	"codewar/server/internal/match"
)

// Step：教程脚本中的一回合行动（确定性，不经难度分布采样）。
type Step struct {
	Action  string // attack | move | defend | skill
	Correct bool   // true = 提交参考解；false = 提交 starter（真实判题失败）
	SolveMs int64  // 固定解题耗时（调度延迟）
	// Target 非空时覆盖自动瞄准（教程 L2 的直线攻击需要固定方向，
	// 让"走出覆盖格"成为可教学、可复现的动作）。仅 attack/skill 生效。
	Target *match.Targeting
}

// Script：按回合执行的确定性脚本（10 号票教程 Bot）。
// 轮次超出计划长度时重复最后一步——教程对局通常在目标达成时即被
// hub 终止，不需要为 10 回合上限逐一列举。
type Script struct {
	Steps []Step
}

func validAction(a string) bool {
	switch a {
	case match.ActionAttack, match.ActionMove, match.ActionDefend, match.ActionSkill:
		return true
	}
	return false
}

// Validate：装配期校验（构造期 fail-closed，避免教程跑到一半才暴露脚本错误）。
func (s *Script) Validate() error {
	if s == nil || len(s.Steps) == 0 {
		return fmt.Errorf("bot 脚本为空")
	}
	for i, st := range s.Steps {
		if !validAction(st.Action) {
			return fmt.Errorf("bot 脚本第 %d 步行动非法 %q", i+1, st.Action)
		}
		if st.SolveMs < 0 {
			return fmt.Errorf("bot 脚本第 %d 步耗时非法 %d", i+1, st.SolveMs)
		}
	}
	return nil
}

// decision：第 round 回合（1 起）的脚本行动；脚本已耗尽时重复最后一步。
func (s *Script) decision(round int) (Decision, bool) {
	if s == nil || len(s.Steps) == 0 || round < 1 {
		return Decision{}, false
	}
	idx := round - 1
	if idx >= len(s.Steps) {
		idx = len(s.Steps) - 1
	}
	st := s.Steps[idx]
	return Decision{Action: st.Action, Correct: st.Correct, SolveMs: st.SolveMs, Target: st.Target}, true
}
