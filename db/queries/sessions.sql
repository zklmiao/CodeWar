-- 会话查询（07 号票：服务端不透明令牌，库内存哈希）。

-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, created_at, expires_at;

-- name: GetSessionByHash :one
SELECT s.user_id, s.expires_at, u.username, u.elo
FROM sessions s JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1;

-- name: DeleteSessionByHash :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: CountSessionsByUser :one
SELECT COUNT(*)::int AS n FROM sessions WHERE user_id = $1;
