-- 种子数据：资深大拿
INSERT INTO experts (name, avatar_url, title, description, win_rate, total_predictions, correct_predictions, status)
VALUES
  ('球王老张', 'https://example.com/avatar/zhang.png', '足彩分析师', '10年竞彩分析经验，擅长胜平负和让球玩法', 68.50, 500, 342, 1),
  ('篮球飞人李', 'https://example.com/avatar/li.png', '篮彩专家', '前职业球员，专注NBA和CBA赛事分析', 72.30, 380, 275, 1),
  ('数据大师王', 'https://example.com/avatar/wang.png', '数据分析师', '统计学背景，用数学模型预测赛果', 65.80, 600, 395, 1)
ON CONFLICT DO NOTHING;
