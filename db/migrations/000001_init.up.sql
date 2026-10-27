-- 000001_init：账号、战绩与 ELO 段位 schema（07 号票原地重写 01 号票脚手架）。
-- 01 号票的 matches/match_events 脚手架是零实例占位，按本票裁决重写为真实 schema，
-- 无需保留迁移历史。对局逐事件日志持久化归 08 号票，本票只落结果行
-- （matches 一行 + elo_history 两行）。
-- 用户表 ELO 默认 1000（spec 附注初稿，调参改 server/internal/elo）。

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 用户：用户名大小写不敏感唯一（username_lower 唯一约束），密码只存 bcrypt 哈希。
CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username       TEXT NOT NULL CHECK (char_length(username) BETWEEN 2 AND 32),
    username_lower TEXT NOT NULL UNIQUE,
    password_hash  TEXT NOT NULL,
    elo            INT NOT NULL DEFAULT 1000,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 会话：服务端不透明令牌（32 字节随机），库内只存 SHA-256 哈希；
-- 同一用户可多会话并存（两窗口两账号 / 同账号多端互不挤）。
CREATE TABLE sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_sessions_user_id ON sessions (user_id);

-- 对局结果行：仅双方均为已认证用户时写入（任一方匿名只走事件流，不落库）；
-- id 由服务端按场生成（UUID），主键天然防止同一场并发结算产生双行。
CREATE TABLE matches (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    winner       TEXT,                 -- 'p1' | 'p2' | NULL=平局
    end_reason   TEXT NOT NULL,        -- ko | self-ko | rounds-cap | resign（31 号票；写入侧无枚举校验，注释即契约）
    duration_ms  INT NOT NULL DEFAULT 0, -- 开赛（双方就位）到终局的毫秒数
    p1_user_id   UUID REFERENCES users (id),
    p2_user_id   UUID REFERENCES users (id)
);
CREATE INDEX idx_matches_p1_finished ON matches (p1_user_id, finished_at DESC);
CREATE INDEX idx_matches_p2_finished ON matches (p2_user_id, finished_at DESC);

-- ELO 历史：每场已结算对局两行（双方各一），记录 before/after 快照；
-- (match_id, user_id) 唯一，防止并发结算重复记账。
CREATE TABLE elo_history (
    id          BIGSERIAL PRIMARY KEY,
    match_id    UUID NOT NULL REFERENCES matches (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    player      TEXT NOT NULL,         -- 'p1' | 'p2'（本场站位）
    elo_before  INT NOT NULL,
    elo_after   INT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (match_id, user_id)
);
CREATE INDEX idx_elo_history_user_created ON elo_history (user_id, created_at DESC);
