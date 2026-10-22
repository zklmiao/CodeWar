package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"codewar/server/internal/store/storedb"

	"github.com/google/uuid"
)

// TutorialProgress：新手教程进度（10 号票）。
// 只由服务端目标判定器在关卡达成后推进——客户端没有任何写入入口，
// 只经 GET /api/tutorial 读取。
type TutorialProgress struct {
	Level       int32      `json:"level"`        // 已完成的最高关（0 = 未开始）
	CompletedAt *time.Time `json:"completed_at"` // 全部通关时刻；nil = 未通关
	Completed   bool       `json:"completed"`    // CompletedAt != nil 的便捷视图
}

// GetTutorialProgress：读取用户的教程进度（用户不存在返回 sql.ErrNoRows）。
func (s *Store) GetTutorialProgress(ctx context.Context, userID uuid.UUID) (TutorialProgress, error) {
	row, err := s.q.GetTutorialProgress(ctx, userID)
	if err != nil {
		return TutorialProgress{}, err
	}
	return newTutorialProgress(row.TutorialLevel, row.TutorialCompletedAt), nil
}

// AdvanceTutorial：把用户进度单调推进到第 level 关（level 为本关序号，1 起），
// 达到 totalLevels 时落完成时刻。重复/乱序调用不会回退，也不会重复写完成时刻。
func (s *Store) AdvanceTutorial(ctx context.Context, userID uuid.UUID, level, totalLevels int32) (TutorialProgress, error) {
	if level < 1 || totalLevels < 1 || level > totalLevels {
		return TutorialProgress{}, fmt.Errorf("教程进度参数非法：level=%d total=%d", level, totalLevels)
	}
	row, err := s.q.AdvanceTutorial(ctx, storedb.AdvanceTutorialParams{
		ID:            userID,
		TutorialLevel: level,
		Column3:       totalLevels,
	})
	if err != nil {
		return TutorialProgress{}, err
	}
	return newTutorialProgress(row.TutorialLevel, row.TutorialCompletedAt), nil
}

func newTutorialProgress(level int32, at sql.NullTime) TutorialProgress {
	p := TutorialProgress{Level: level, Completed: at.Valid}
	if at.Valid {
		t := at.Time
		p.CompletedAt = &t
	}
	return p
}
