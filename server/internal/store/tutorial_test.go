package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// 教程进度持久化（10 号票验收 4）：真实 DB + 迁移 000002。
// 门控与隔离沿用 openTestStore（CODEWAR_TEST_DSN，库名含 test）。
func TestTutorialProgressPersistence(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	u, _, err := s.Register(ctx, "tutorial-user", "password1")
	if err != nil {
		t.Fatalf("注册失败：%v", err)
	}

	// 新账号：迁移默认值 level=0、未通关。
	p, err := s.GetTutorialProgress(ctx, u.ID)
	if err != nil {
		t.Fatalf("读取初始进度失败：%v", err)
	}
	if p.Level != 0 || p.Completed || p.CompletedAt != nil {
		t.Fatalf("新账号初始进度应为 0/未通关：%+v", p)
	}

	// 完成第 1 关：level 推进，但未通关（completed_at 仍为空）。
	if p, err = s.AdvanceTutorial(ctx, u.ID, 1, 4); err != nil {
		t.Fatalf("推进第 1 关失败：%v", err)
	}
	if p.Level != 1 || p.Completed || p.CompletedAt != nil {
		t.Fatalf("第 1 关后应为 level=1/未通关：%+v", p)
	}

	// 乱序回退不得生效：先到 3，再报 2 仍是 3（GREATEST 单调）。
	if _, err := s.AdvanceTutorial(ctx, u.ID, 3, 4); err != nil {
		t.Fatal(err)
	}
	if p, err = s.AdvanceTutorial(ctx, u.ID, 2, 4); err != nil {
		t.Fatal(err)
	}
	if p.Level != 3 {
		t.Fatalf("乱序推进不得回退：level=%d", p.Level)
	}

	// 完成第 4 关：落完成时刻。
	p, err = s.AdvanceTutorial(ctx, u.ID, 4, 4)
	if err != nil {
		t.Fatalf("推进第 4 关失败：%v", err)
	}
	if p.Level != 4 || !p.Completed || p.CompletedAt == nil {
		t.Fatalf("第 4 关后应通关：%+v", p)
	}
	first := *p.CompletedAt

	// 重复达成：完成时刻保持不变（只写一次）。
	time.Sleep(5 * time.Millisecond)
	p2, err := s.AdvanceTutorial(ctx, u.ID, 4, 4)
	if err != nil {
		t.Fatal(err)
	}
	if p2.CompletedAt == nil || !p2.CompletedAt.Equal(first) {
		t.Fatalf("重复达成不得覆盖完成时刻：first=%v second=%v", first, p2.CompletedAt)
	}

	// 未知用户：fail-closed。
	if _, err := s.AdvanceTutorial(ctx, uuid.New(), 1, 4); err == nil {
		t.Fatal("未知用户推进应报错")
	}
	// 参数越界：不会静默写出非法进度。
	if _, err := s.AdvanceTutorial(ctx, u.ID, 5, 4); err == nil {
		t.Fatal("level > total 应被拒绝")
	}
}
