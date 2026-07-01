-- 为 /lottery/pk 接口补充索引
-- GetDailyPK 按 matches.match_time_str 的 11:00 周期范围过滤 (前日 11:00 ~ 当日 11:00),
-- 对应竞彩开奖周期,而非自然日或 settled_at。
-- matches 表已有 UNIQUE(match_id, lottery_type) 约束和 idx_matches_match_time 索引,
-- 此部分索引在 predictions 表上覆盖 user_id + status 过滤,加速 JOIN 后的筛选。

CREATE INDEX IF NOT EXISTS idx_predictions_user_status
    ON predictions (user_id, status)
    WHERE status IN ('won', 'lost');
