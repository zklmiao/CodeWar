-- ELO 历史查询（07 号票：每场已结算对局双方各一行）。

-- name: CreateEloHistory :one
INSERT INTO elo_history (match_id, user_id, player, elo_before, elo_after)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at;

-- name: ListEloHistoryByMatch :many
SELECT match_id, user_id, player, elo_before, elo_after, created_at
FROM elo_history WHERE match_id = $1 ORDER BY player;
