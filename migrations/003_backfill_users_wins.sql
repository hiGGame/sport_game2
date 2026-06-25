-- 回填 users.wins 字段
-- 背景: SettleAllForMatch/SettlePrediction 历史上只更新 predictions.is_correct,
-- 未同步累加 users.wins, 导致 /lottery/leaderboard 的 GetTopUsers 排序失效。
-- 本脚本一次性回填历史数据; 之后由代码在结算时实时维护。

UPDATE users u
SET wins = COALESCE(c.w, 0)
FROM (
    SELECT user_id, COUNT(*) AS w
    FROM predictions
    WHERE is_correct = true
    GROUP BY user_id
) c
WHERE u.id = c.user_id AND u.wins <> c.w;
