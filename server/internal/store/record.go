package store

import (
	"context"
	"database/sql"

	"codewar/server/internal/elo"
	"codewar/server/internal/store/storedb"

	"github.com/google/uuid"
)

// MatchResult：hub 在引擎终局后提交的结算输入（调用方已确认双方均为已认证用户）。
type MatchResult struct {
	MatchID    uuid.UUID // 本场唯一 ID（hub 按场生成；主键防并发双写）
	Winner     string    // "p1" | "p2" | ""=平局
	Reason     string    // ko | self-ko | rounds-cap | resign（31 号票起；无枚举校验，透传入库）
	DurationMs int32     // 开赛到终局的毫秒数
	P1UserID   uuid.UUID
	P2UserID   uuid.UUID
	// 难度风险系数（05 号票）：结算奖励（ELO delta）按所选难度档放大/缩小
	// 后再入库——库内历史、快照与终局广播始终同一口径。0 = 未指定 → 按 1.0
	// 处理（07 号票既有调用方零改动）。
	RiskCoeff float64
}

// EloDelta：单名玩家的 ELO 变化（直接对应 proto PlayerElo）。
type EloDelta struct {
	Player string
	UserID uuid.UUID
	Before int32
	After  int32
}

// RecordMatch：ELO 结算 + 结果落库（单事务）。
// 锁序按 userID 排序防死锁；用户行 SELECT … FOR UPDATE 保证并发结算读到最新 ELO；
// 同一 matchID 并发写入时第二个提交因主键冲突失败——调用方应降级为"无 ELO 广播"，
// 不得重试（重试会写出第二行历史，违背"一场 exactly 两行"）。
func (s *Store) RecordMatch(ctx context.Context, res MatchResult) (p1, p2 EloDelta, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return p1, p2, err
	}
	defer tx.Rollback()
	q := s.q.WithTx(tx)

	// 锁序固定：先小后大，避免双场交叉结算死锁。
	first, second := res.P1UserID, res.P2UserID
	if second.String() < first.String() {
		first, second = second, first
	}
	u1, err := q.GetUserByIDForUpdate(ctx, first)
	if err != nil {
		return p1, p2, err
	}
	u2, err := q.GetUserByIDForUpdate(ctx, second)
	if err != nil {
		return p1, p2, err
	}
	byID := map[uuid.UUID]int32{u1.ID: u1.Elo, u2.ID: u2.Elo}

	after1, after2 := elo.Settle(int(byID[res.P1UserID]), int(byID[res.P2UserID]), res.Winner)
	// 难度风险系数（05 号票）：delta × coeff 后四舍五入再落库——平局 delta=0
	// 天然仍 +0；coeff 未指定（0）按 1.0 处理，07 号票行为不变。
	coeff := res.RiskCoeff
	if coeff <= 0 {
		coeff = 1
	}
	after1 = int(byID[res.P1UserID]) + elo.ScaleDelta(after1-int(byID[res.P1UserID]), coeff)
	after2 = int(byID[res.P2UserID]) + elo.ScaleDelta(after2-int(byID[res.P2UserID]), coeff)
	if err := q.UpdateUserElo(ctx, storedb.UpdateUserEloParams{ID: res.P1UserID, Elo: int32(after1)}); err != nil {
		return p1, p2, err
	}
	if err := q.UpdateUserElo(ctx, storedb.UpdateUserEloParams{ID: res.P2UserID, Elo: int32(after2)}); err != nil {
		return p1, p2, err
	}

	winner := sql.NullString{String: res.Winner, Valid: res.Winner != ""}
	if _, err := q.CreateMatch(ctx, storedb.CreateMatchParams{
		ID:         res.MatchID,
		Winner:     winner,
		EndReason:  res.Reason,
		DurationMs: res.DurationMs,
		P1UserID:   uuid.NullUUID{UUID: res.P1UserID, Valid: true},
		P2UserID:   uuid.NullUUID{UUID: res.P2UserID, Valid: true},
	}); err != nil {
		return p1, p2, err
	}
	for _, d := range []EloDelta{
		{Player: "p1", UserID: res.P1UserID, Before: byID[res.P1UserID], After: int32(after1)},
		{Player: "p2", UserID: res.P2UserID, Before: byID[res.P2UserID], After: int32(after2)},
	} {
		if _, err := q.CreateEloHistory(ctx, storedb.CreateEloHistoryParams{
			MatchID:   res.MatchID,
			UserID:    d.UserID,
			Player:    d.Player,
			EloBefore: d.Before,
			EloAfter:  d.After,
		}); err != nil {
			return p1, p2, err
		}
	}
	if err := tx.Commit(); err != nil {
		return p1, p2, err
	}
	p1 = EloDelta{Player: "p1", UserID: res.P1UserID, Before: byID[res.P1UserID], After: int32(after1)}
	p2 = EloDelta{Player: "p2", UserID: res.P2UserID, Before: byID[res.P2UserID], After: int32(after2)}
	return p1, p2, nil
}
