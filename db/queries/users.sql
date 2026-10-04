-- 用户查询（07 号票：注册 / 登录 / 鉴权 / ELO 更新）。

-- name: CreateUser :one
INSERT INTO users (username, username_lower, password_hash, elo)
VALUES ($1, $2, $3, $4)
RETURNING id, username, elo, created_at;

-- name: GetUserByLower :one
SELECT id, username, username_lower, password_hash, elo, created_at
FROM users WHERE username_lower = $1;

-- name: GetUserByID :one
SELECT id, username, elo, created_at
FROM users WHERE id = $1;

-- name: GetUserByIDForUpdate :one
SELECT id, username, elo, created_at
FROM users WHERE id = $1 FOR UPDATE;

-- name: UpdateUserElo :exec
UPDATE users SET elo = $2 WHERE id = $1;
