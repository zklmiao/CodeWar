-- 对局结果行查询（07 号票：只落结果行，不含逐事件日志）。

-- name: CreateMatch :one
INSERT INTO matches (id, winner, end_reason, duration_ms, p1_user_id, p2_user_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, finished_at;

-- name: GetMatchByID :one
SELECT id, finished_at, winner, end_reason, duration_ms, p1_user_id, p2_user_id
FROM matches WHERE id = $1;

-- name: GetUserStats :one
SELECT
  COUNT(*)::int AS total,
  COUNT(*) FILTER (
    WHERE (m.winner = 'p1' AND m.p1_user_id = $1)
       OR (m.winner = 'p2' AND m.p2_user_id = $1)
  )::int AS wins,
  COUNT(*) FILTER (WHERE m.winner IS NULL)::int AS draws
FROM matches m
WHERE m.p1_user_id = $1 OR m.p2_user_id = $1;

-- name: ListRecentMatchesByUser :many
-- 查看者本人的 elo 行（h）决定其本场站位与 before/after；对手名由另一侧用户表带出。
SELECT m.id, m.finished_at, m.winner, m.end_reason, m.duration_ms,
       m.p1_user_id, m.p2_user_id,
       p1u.username AS p1_username, p2u.username AS p2_username,
       h.player, h.elo_before, h.elo_after
FROM matches m
JOIN elo_history h ON h.match_id = m.id AND h.user_id = $1
LEFT JOIN users p1u ON p1u.id = m.p1_user_id
LEFT JOIN users p2u ON p2u.id = m.p2_user_id
WHERE m.p1_user_id = $1 OR m.p2_user_id = $1
ORDER BY m.finished_at DESC
LIMIT $2;
