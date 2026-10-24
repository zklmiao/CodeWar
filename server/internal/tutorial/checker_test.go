package tutorial

import (
	"testing"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/match"
)

// 目标判定器（10 号票验收 2）：每关"达成→完成 / 未达成→不完成"。
// 事件序列是引擎对外发布的同构载荷；判定器不依赖引擎内部状态。
//
// 删除实现里对应的判定分支（如 L2 的 `!c.hurtThisRound` 守卫），
// 下面对应的负例即变红。

func roundStarted(round int) *matchpb.Envelope {
	return &matchpb.Envelope{Event: &matchpb.Envelope_RoundStarted{
		RoundStarted: &matchpb.RoundStarted{Round: uint32(round)},
	}}
}

func attackResolved(attacker string, damage int32, dodged, shielded bool) *matchpb.Envelope {
	return &matchpb.Envelope{Event: &matchpb.Envelope_ActionResolved{
		ActionResolved: &matchpb.ActionResolved{
			Attacker: attacker, Action: match.ActionAttack,
			Damage: damage, Dodged: dodged, Shielded: shielded,
		},
	}}
}

func skillResolved(attacker string) *matchpb.Envelope {
	return &matchpb.Envelope{Event: &matchpb.Envelope_ActionResolved{
		ActionResolved: &matchpb.ActionResolved{Attacker: attacker, Action: match.ActionSkill},
	}}
}

func hpChanged(player string, delta int32) *matchpb.Envelope {
	return &matchpb.Envelope{Event: &matchpb.Envelope_HpChanged{
		HpChanged: &matchpb.HpChanged{Player: player, Delta: delta},
	}}
}

func feed(t *testing.T, level int, envs ...*matchpb.Envelope) State {
	t.Helper()
	c, err := NewChecker(level, match.DefaultRiskTier)
	if err != nil {
		t.Fatalf("构造判定器失败：%v", err)
	}
	var st State
	for _, e := range envs {
		st = c.Observe(e)
	}
	return st
}

func TestCheckerL1AndL4RequirePlayerDamage(t *testing.T) {
	for _, level := range []int{1, 4} {
		t.Run("达成：玩家攻击命中", func(t *testing.T) {
			st := feed(t, level, roundStarted(1), attackResolved("p1", 12, false, false))
			if !st.Completed || !st.Goals[0].Done {
				t.Fatalf("玩家造成伤害后应完成：%+v", st)
			}
		})
		t.Run("未达成：玩家只移动/防御", func(t *testing.T) {
			st := feed(t, level, roundStarted(1),
				&matchpb.Envelope{Event: &matchpb.Envelope_ActionResolved{ActionResolved: &matchpb.ActionResolved{
					Attacker: "p1", Action: match.ActionMove,
				}}})
			if st.Completed {
				t.Fatalf("没有伤害不得完成：%+v", st)
			}
		})
		t.Run("未达成：只有 Bot 造成伤害", func(t *testing.T) {
			st := feed(t, level, roundStarted(1), attackResolved("p2", 18, false, false), hpChanged("p1", -18))
			if st.Completed {
				t.Fatalf("Bot 的伤害不算玩家达成：%+v", st)
			}
		})
	}
}

func TestCheckerL2RequiresDodgeWithoutDamage(t *testing.T) {
	t.Run("达成：Bot 直线攻击落空且本回合未受伤", func(t *testing.T) {
		st := feed(t, 2, roundStarted(1), attackResolved("p2", 0, true, false))
		if !st.Completed {
			t.Fatalf("闪避后应完成：%+v", st)
		}
	})
	t.Run("未达成：玩家硬扛一次直线攻击", func(t *testing.T) {
		// 删掉判定里的 hurtThisRound 守卫 / 把 Dodged 判断写成 Damage==0，
		// 这条会变红。硬扛 = Dodged=false + 玩家受伤事件。
		st := feed(t, 2, roundStarted(1), attackResolved("p2", 18, false, false), hpChanged("p1", -18))
		if st.Completed {
			t.Fatalf("硬扛伤害不得完成：%+v", st)
		}
	})
	t.Run("未达成：闪避同回合但玩家自损", func(t *testing.T) {
		st := feed(t, 2, roundStarted(1), hpChanged("p1", -3), attackResolved("p2", 0, true, false))
		if st.Completed {
			t.Fatalf("本回合有受伤事件不得完成：%+v", st)
		}
	})
	t.Run("回合作用域：上一回合受伤不影响本回合闪避", func(t *testing.T) {
		st := feed(t, 2,
			roundStarted(1), attackResolved("p2", 18, false, false), hpChanged("p1", -18),
			roundStarted(2), attackResolved("p2", 0, true, false))
		if !st.Completed {
			t.Fatalf("判定按回合重置，第二回合闪避应完成：%+v", st)
		}
	})
	t.Run("未达成：玩家的攻击落空不算闪避目标", func(t *testing.T) {
		st := feed(t, 2, roundStarted(1), attackResolved("p1", 0, true, false))
		if st.Completed {
			t.Fatalf("必须是 Bot 的攻击被闪避：%+v", st)
		}
	})
}

func TestCheckerL3RequiresShieldedHitAndOwnSkill(t *testing.T) {
	t.Run("达成：盾吃攻击 + 自己释放技能", func(t *testing.T) {
		st := feed(t, 3, roundStarted(1), attackResolved("p2", 10, false, true),
			roundStarted(2), skillResolved("p1"))
		if !st.Completed || !st.Goals[0].Done || !st.Goals[1].Done {
			t.Fatalf("两个目标都应完成：%+v", st)
		}
	})
	t.Run("未达成：只挂盾吃伤害，没放技能", func(t *testing.T) {
		st := feed(t, 3, roundStarted(1), attackResolved("p2", 10, false, true))
		if st.Completed || !st.Goals[0].Done {
			t.Fatalf("缺少技能目标不得完成：%+v", st)
		}
	})
	t.Run("未达成：只放技能，没有盾吃伤害", func(t *testing.T) {
		st := feed(t, 3, roundStarted(1), skillResolved("p1"))
		if st.Completed || !st.Goals[1].Done {
			t.Fatalf("缺少护盾目标不得完成：%+v", st)
		}
	})
	t.Run("未达成：未挂盾硬吃攻击（无 Shielded）", func(t *testing.T) {
		st := feed(t, 3, roundStarted(1), attackResolved("p2", 26, false, false), skillResolved("p1"))
		if st.Completed || st.Goals[0].Done {
			t.Fatalf("无护盾折减不算目标 1：%+v", st)
		}
	})
	t.Run("未达成：Bot 的技能命中有 Shielded 也不算玩家目标", func(t *testing.T) {
		st := feed(t, 3, roundStarted(1),
			&matchpb.Envelope{Event: &matchpb.Envelope_ActionResolved{ActionResolved: &matchpb.ActionResolved{
				Attacker: "p2", Action: match.ActionSkill, Damage: 22, Shielded: true,
			}}})
		if st.Completed {
			t.Fatalf("必须是玩家自己的技能事件：%+v", st)
		}
	})
}

func TestCheckerStateMetadata(t *testing.T) {
	c, err := NewChecker(4, 5)
	if err != nil {
		t.Fatal(err)
	}
	st := c.State()
	if st.Level != 4 || st.NextLevel != 0 || st.Cap != 5 || st.CapCoeff != match.RiskCoeff(5) {
		t.Fatalf("L4/cap5 元数据不符：%+v", st)
	}
	if st.CapCoeff <= 1.0 {
		t.Fatalf("L5 伤害系数应大于 1（越高越疼）：%v", st.CapCoeff)
	}
	c3, err := NewChecker(2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if st3 := c3.State(); st3.NextLevel != 3 || st3.Cap != match.DefaultRiskTier {
		t.Fatalf("L2 缺省 cap 应为 L3、下一关 3：%+v", st3)
	}
	if _, err := NewChecker(5, 3); err == nil {
		t.Fatal("未知关号应报错")
	}
}

func TestBankHasFixedQuestionPerAction(t *testing.T) {
	for _, l := range Levels {
		b, err := NewBank(l.N)
		if err != nil {
			t.Fatalf("L%d 题包构造失败：%v", l.N, err)
		}
		for _, action := range []string{match.ActionAttack, match.ActionMove, match.ActionDefend, match.ActionSkill} {
			q := b.q[action]
			if q.ID == "" || q.AnswerKey == "" || len(q.Tests) == 0 || q.StarterCode == "" {
				t.Fatalf("L%d 的 %s 题不完整：%+v", l.N, action, q)
			}
		}
		// Bot 脚本每一步都必须有可提交的题（否则 Bot 会拿到空题被判错）。
		for i, step := range l.Script.Steps {
			if q := b.q[step.Action]; q.AnswerKey == "" {
				t.Fatalf("L%d 脚本第 %d 步（%s）没有参考解", l.N, i+1, step.Action)
			}
		}
	}
	if _, err := NewBank(0); err == nil {
		t.Fatal("未知关号应报错")
	}
}
