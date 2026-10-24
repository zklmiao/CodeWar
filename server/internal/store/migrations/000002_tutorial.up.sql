-- 000002_tutorial：新手教程进度（10 号票）。
-- 进度只由服务端目标判定器写入（客户端无 advance 端点）；tutorial_level =
-- 已完成的最高关（0 = 未开始），tutorial_completed_at 在通关第 4 关时落库。
ALTER TABLE users
    ADD COLUMN tutorial_level INT NOT NULL DEFAULT 0,
    ADD COLUMN tutorial_completed_at TIMESTAMPTZ;

ALTER TABLE users
    ADD CONSTRAINT users_tutorial_level_range CHECK (tutorial_level BETWEEN 0 AND 4);
