package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// ===== 难度风险系数 × 结算落库（05 号票；DB 门控同 store_test.go）=====
// delta × coeff 四舍五入后落库：用户当前分与 ELO 历史始终同一口径
// （与终局广播的 elo_after 一致，不允许"库里 16、屏上 19"的漂移）。

// 风险系数 1.2：同分 p1 胜（原始 delta +16/−16）应放大为 +19/−19（round(19.2)/round(−19.2)）。
func TestRecordMatchAppliesRiskCoeff(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	a, _, err := s.Register(ctx, "risk-a", "password1")
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := s.Register(ctx, "risk-b", "password2")
	if err != nil {
		t.Fatal(err)
	}

	mid := uuid.New()
	p1, p2, err := s.RecordMatch(ctx, MatchResult{
		MatchID: mid, Winner: "p1", Reason: "ko", DurationMs: 60000,
		P1UserID: a.ID, P2UserID: b.ID,
		RiskCoeff: 1.2,
	})
	if err != nil {
		t.Fatalf("结算落库失败：%v", err)
	}
	if p1.After != 1019 || p2.After != 981 {
		t.Fatalf("系数 1.2 下应 1019/981（round(16×1.2)=19），实际 %+v %+v", p1, p2)
	}

	ua, _ := s.q.GetUserByID(ctx, a.ID)
	ub, _ := s.q.GetUserByID(ctx, b.ID)
	if ua.Elo != 1019 || ub.Elo != 981 {
		t.Fatalf("用户当前分应与放大后 delta 一致：%d / %d", ua.Elo, ub.Elo)
	}

	hist, err := s.q.ListEloHistoryByMatch(ctx, mid)
	if err != nil || len(hist) != 2 {
		t.Fatalf("ELO 历史应恰两行：%v %+v", err, hist)
	}
	for _, h := range hist {
		want := int32(1019)
		if h.Player == "p2" {
			want = 981
		}
		if h.EloAfter != want {
			t.Fatalf("历史行口径漂移：%+v", h)
		}
	}
}

// 平局 + RiskCoeff：delta=0 与系数相乘仍 +0（spec 附注硬规则）。
func TestRecordMatchRiskCoeffDrawStaysZero(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	a, _, err := s.Register(ctx, "risk-draw-a", "password1")
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := s.Register(ctx, "risk-draw-b", "password2")
	if err != nil {
		t.Fatal(err)
	}

	p1, p2, err := s.RecordMatch(ctx, MatchResult{
		MatchID: uuid.New(), Winner: "", Reason: "rounds-cap", DurationMs: 60000,
		P1UserID: a.ID, P2UserID: b.ID,
		RiskCoeff: 1.2,
	})
	if err != nil {
		t.Fatalf("结算落库失败：%v", err)
	}
	if p1.After != p1.Before || p2.After != p2.Before {
		t.Fatalf("平局在风险系数下仍应 +0：%+v %+v", p1, p2)
	}
}

// RiskCoeff 未指定（0）按 1.0 处理：07 号票既有调用方行为不变（回归锚）。
func TestRecordMatchZeroRiskCoeffUnchanged(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	a, _, err := s.Register(ctx, "risk-zero-a", "password1")
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := s.Register(ctx, "risk-zero-b", "password2")
	if err != nil {
		t.Fatal(err)
	}

	p1, p2, err := s.RecordMatch(ctx, MatchResult{
		MatchID: uuid.New(), Winner: "p1", Reason: "ko", DurationMs: 60000,
		P1UserID: a.ID, P2UserID: b.ID,
	})
	if err != nil {
		t.Fatalf("结算落库失败：%v", err)
	}
	if p1.After != 1016 || p2.After != 984 {
		t.Fatalf("未指定系数应与 07 号票一致（1016/984）：%+v %+v", p1, p2)
	}
}
