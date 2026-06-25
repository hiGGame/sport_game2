-- 清理 experts 表: 仅保留大拿(老委鬼)需要的数据, 但大拿走 robot_expert 用户路径,
-- 不再使用 experts 表, 清空冗余 seed 数据。
-- 同时清理对应的 expert_predictions 表数据(若有)。

DELETE FROM expert_predictions WHERE expert_id IN (SELECT id FROM experts);
DELETE FROM experts;
