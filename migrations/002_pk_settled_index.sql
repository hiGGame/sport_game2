-- 为 /lottery/pk 接口补充表达式索引
-- GetDailyPK 使用 (settled_at AT TIME ZONE 'Asia/Shanghai')::date 作为过滤条件，
-- 普通索引无法覆盖，需要表达式索引。
-- 同时加 WHERE status IN ('won','lost') 使其为部分索引，进一步缩小体积。

CREATE INDEX IF NOT EXISTS idx_predictions_user_settled_tz
    ON predictions (user_id, ((settled_at AT TIME ZONE 'Asia/Shanghai')::date))
    WHERE status IN ('won', 'lost');
