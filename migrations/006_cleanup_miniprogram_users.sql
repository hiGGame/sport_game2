-- 清理微信小程序登录用户(已弃用小程序登录方式)
-- 微信小程序 open_id 以 "oS_" 开头, 现仅使用微信公众号登录
DELETE FROM predictions WHERE user_id IN (SELECT id FROM users WHERE open_id LIKE 'oS_%');
DELETE FROM credit_logs WHERE user_id IN (SELECT id FROM users WHERE open_id LIKE 'oS_%');
DELETE FROM users WHERE open_id LIKE 'oS_%';
