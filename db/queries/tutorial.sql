-- 新手教程进度（10 号票）：服务端目标判定器达成后写入，客户端只读。

-- name: GetTutorialProgress :one
SELECT tutorial_level, tutorial_completed_at FROM users WHERE id = $1;

-- name: AdvanceTutorial :one
-- 单调推进（GREATEST）：重复达成/乱序达成不会回退进度；
-- 只有完成最后一关（$2 >= $3）才落完成时刻，且只写一次。
UPDATE users
SET tutorial_level = GREATEST(tutorial_level, $2),
    tutorial_completed_at = CASE
        WHEN $2 >= $3::int AND tutorial_completed_at IS NULL THEN now()
        ELSE tutorial_completed_at
    END
WHERE id = $1
RETURNING tutorial_level, tutorial_completed_at;
