-- 000001_init：MVP 初始 schema 脚手架。
-- 01 号票（Tracer）只落 schema，不做持久化接入（账号/ELO 是 07 号票；
-- 对局事件日志按 ADR-0002 服务端重放要求预留）。

CREATE TABLE matches (
    id           UUID PRIMARY KEY,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at  TIMESTAMPTZ,
    winner       TEXT,             -- 'p1' | 'p2' | NULL=平局
    end_reason   TEXT              -- ko | self-ko | rounds-cap
);

CREATE TABLE match_events (
    id          BIGSERIAL PRIMARY KEY,
    match_id    UUID NOT NULL REFERENCES matches (id),
    seq         INT  NOT NULL,     -- 单调递增（ADR-0002）
    unix_ms     BIGINT NOT NULL,
    payload     JSONB NOT NULL,    -- Envelope 原文
    UNIQUE (match_id, seq)
);

-- 审计友好的部分索引示例（设计 002 §6）：只索引终结事件
CREATE INDEX idx_match_events_end
    ON match_events (match_id)
    WHERE payload ->> 'matchEnded' IS NOT NULL;
