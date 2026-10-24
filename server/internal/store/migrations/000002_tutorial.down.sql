-- 000002_tutorial 回滚：移除教程进度列与约束。
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_tutorial_level_range;
ALTER TABLE users
    DROP COLUMN IF EXISTS tutorial_completed_at,
    DROP COLUMN IF EXISTS tutorial_level;
