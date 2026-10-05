package store

import (
	"context"
	"time"

	"codewar/server/internal/store/storedb"

	"github.com/google/uuid"
)

// RecentMatch：个人页最近对局的一项（结果与 ELO 变化均以查看者视角表达）。
type RecentMatch struct {
	MatchID    string    `json:"match_id"`
	FinishedAt time.Time `json:"finished_at"`
	Player     string    `json:"player"`   // 查看者本场站位 p1/p2
	Opponent   string    `json:"opponent"` // 对手用户名
	Outcome    string    `json:"outcome"`  // win | loss | draw（查看者视角）
	Reason     string    `json:"reason"`   // ko | self-ko | rounds-cap
	EloBefore  int32     `json:"elo_before"`
	EloAfter   int32     `json:"elo_after"`
}

// Profile：个人页聚合（用户名、当前 ELO、总场次/胜率、最近对局）。
type Profile struct {
	UserID   string        `json:"user_id"`
	Username string        `json:"username"`
	Elo      int32         `json:"elo"`
	Total    int32         `json:"total"`
	Wins     int32         `json:"wins"`
	Losses   int32         `json:"losses"`
	Draws    int32         `json:"draws"`
	WinRate  float64       `json:"win_rate"` // wins/total，总场次为 0 时 0
	Recent   []RecentMatch `json:"recent"`
}

// DefaultRecentLimit：个人页最近对局默认条数。
const DefaultRecentLimit = 20

// GetProfile：聚合用户 + 战绩 + 最近对局（一次调用供个人页整页渲染）。
func (s *Store) GetProfile(ctx context.Context, userID uuid.UUID, limit int) (Profile, error) {
	if limit <= 0 || limit > 100 {
		limit = DefaultRecentLimit
	}
	u, err := s.q.GetUserByID(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	stats, err := s.q.GetUserStats(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err != nil {
		return Profile{}, err
	}
	rows, err := s.q.ListRecentMatchesByUser(ctx, storedb.ListRecentMatchesByUserParams{
		UserID: userID,
		Limit:  int32(limit),
	})
	if err != nil {
		return Profile{}, err
	}
	p := Profile{
		UserID:   u.ID.String(),
		Username: u.Username,
		Elo:      u.Elo,
		Total:    stats.Total,
		Wins:     stats.Wins,
		Draws:    stats.Draws,
		Losses:   stats.Total - stats.Wins - stats.Draws,
		Recent:   make([]RecentMatch, 0, len(rows)),
	}
	if p.Total > 0 {
		p.WinRate = float64(p.Wins) / float64(p.Total)
	}
	for _, r := range rows {
		m := RecentMatch{
			MatchID:    r.ID.String(),
			FinishedAt: r.FinishedAt,
			Player:     r.Player,
			Reason:     r.EndReason,
			EloBefore:  r.EloBefore,
			EloAfter:   r.EloAfter,
		}
		// 对手名取另一侧；站位决定胜负视角。
		if r.Player == "p1" {
			m.Opponent = r.P2Username.String
		} else {
			m.Opponent = r.P1Username.String
		}
		switch {
		case !r.Winner.Valid || r.Winner.String == "":
			m.Outcome = "draw"
		case r.Winner.String == r.Player:
			m.Outcome = "win"
		default:
			m.Outcome = "loss"
		}
		p.Recent = append(p.Recent, m)
	}
	return p, nil
}
