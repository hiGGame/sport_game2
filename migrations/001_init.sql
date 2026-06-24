-- Migration 001: Initial schema for sport_game2

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 赛事表
CREATE TABLE IF NOT EXISTS matches (
    id BIGSERIAL PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL,
    sport_id VARCHAR(10) NOT NULL DEFAULT '1',
    lottery_type VARCHAR(10) NOT NULL,
    match_code VARCHAR(20) NOT NULL,
    issue VARCHAR(20) NOT NULL DEFAULT '',
    match_time_str VARCHAR(40) NOT NULL DEFAULT '',
    bet_end_time_str VARCHAR(40) NOT NULL DEFAULT '',
    league_id VARCHAR(20) NOT NULL DEFAULT '',
    league_name VARCHAR(100) NOT NULL DEFAULT '',
    league_alias VARCHAR(100) NOT NULL DEFAULT '',
    league_color VARCHAR(20) NOT NULL DEFAULT '',
    league_level VARCHAR(10) NOT NULL DEFAULT '',
    league_logo TEXT NOT NULL DEFAULT '',
    home_team_name VARCHAR(100) NOT NULL DEFAULT '',
    home_team_alias VARCHAR(100) NOT NULL DEFAULT '',
    home_team_logo TEXT NOT NULL DEFAULT '',
    home_team_rank VARCHAR(10) NOT NULL DEFAULT '',
    away_team_name VARCHAR(100) NOT NULL DEFAULT '',
    away_team_alias VARCHAR(100) NOT NULL DEFAULT '',
    away_team_logo TEXT NOT NULL DEFAULT '',
    away_team_rank VARCHAR(10) NOT NULL DEFAULT '',
    bet_infos JSONB NOT NULL DEFAULT '[]',
    odds_data JSONB NOT NULL DEFAULT '{}',
    is_stop_sell BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(10) NOT NULL DEFAULT '',
    raw_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(match_id, lottery_type)
);

CREATE INDEX IF NOT EXISTS idx_matches_lottery_type ON matches(lottery_type);
CREATE INDEX IF NOT EXISTS idx_matches_match_code ON matches(match_code);
CREATE INDEX IF NOT EXISTS idx_matches_issue ON matches(issue);
CREATE INDEX IF NOT EXISTS idx_matches_match_time ON matches(match_time_str);
CREATE INDEX IF NOT EXISTS idx_matches_bet_infos ON matches USING GIN (bet_infos);

-- 开奖结果表
CREATE TABLE IF NOT EXISTS match_results (
    id BIGSERIAL PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL UNIQUE,
    match_code VARCHAR(20) NOT NULL DEFAULT '',
    issue VARCHAR(20) NOT NULL DEFAULT '',
    match_time_str VARCHAR(40) NOT NULL DEFAULT '',
    week_day VARCHAR(10) NOT NULL DEFAULT '',
    round VARCHAR(10) NOT NULL DEFAULT '',
    home_team_name VARCHAR(100) NOT NULL DEFAULT '',
    away_team_name VARCHAR(100) NOT NULL DEFAULT '',
    league_name VARCHAR(100) NOT NULL DEFAULT '',
    home_score VARCHAR(20) NOT NULL DEFAULT '',
    away_score VARCHAR(20) NOT NULL DEFAULT '',
    normal_time_score VARCHAR(20) NOT NULL DEFAULT '',
    half_time_score VARCHAR(20) NOT NULL DEFAULT '',
    is_valid BOOLEAN NOT NULL DEFAULT false,
    game_draw_list JSONB NOT NULL DEFAULT '[]',
    lottery_type VARCHAR(10) NOT NULL DEFAULT '',
    raw_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_results_match_code ON match_results(match_code);
CREATE INDEX IF NOT EXISTS idx_results_is_valid ON match_results(is_valid);
CREATE INDEX IF NOT EXISTS idx_results_match_time ON match_results(match_time_str);

-- 爬虫任务日志表
CREATE TABLE IF NOT EXISTS spider_job_log (
    id BIGSERIAL PRIMARY KEY,
    job_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    record_count INT NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_spider_log_type ON spider_job_log(job_type);
CREATE INDEX IF NOT EXISTS idx_spider_log_created ON spider_job_log(created_at);

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    open_id VARCHAR(128) UNIQUE NOT NULL,
    union_id VARCHAR(128) NOT NULL DEFAULT '',
    nickname VARCHAR(100) NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    phone VARCHAR(20) NOT NULL DEFAULT '',
    credits INT NOT NULL DEFAULT 1000,
    total_bets INT NOT NULL DEFAULT 0,
    wins INT NOT NULL DEFAULT 0,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_open_id ON users(open_id);

-- 用户竞猜表
CREATE TABLE IF NOT EXISTS predictions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    match_id VARCHAR(64) NOT NULL,
    lottery_type VARCHAR(10) NOT NULL,
    match_code VARCHAR(20) NOT NULL DEFAULT '',
    sub_type VARCHAR(10) NOT NULL,
    bet_code VARCHAR(20) NOT NULL,
    handicap NUMERIC(5,2) NOT NULL DEFAULT 0,
    points INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    is_correct BOOLEAN,
    settled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_predictions_user_id ON predictions(user_id);
CREATE INDEX IF NOT EXISTS idx_predictions_match_id ON predictions(match_id);
CREATE INDEX IF NOT EXISTS idx_predictions_status ON predictions(status);
-- 防止同一用户对同一赛事同一玩法重复下注（仅限 pending 状态）
CREATE UNIQUE INDEX IF NOT EXISTS idx_predictions_user_match_unique ON predictions(user_id, match_id, lottery_type, sub_type) WHERE status = 'pending';

-- AI 预测表
CREATE TABLE IF NOT EXISTS ai_predictions (
    id BIGSERIAL PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL,
    lottery_type VARCHAR(10) NOT NULL,
    sub_type VARCHAR(10) NOT NULL,
    bet_code VARCHAR(20) NOT NULL,
    confidence NUMERIC(5,4) NOT NULL DEFAULT 0,
    reasoning TEXT NOT NULL DEFAULT '',
    model_name VARCHAR(50) NOT NULL DEFAULT 'rule',
    is_correct BOOLEAN,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(match_id, lottery_type, sub_type)
);

CREATE INDEX IF NOT EXISTS idx_ai_pred_match_id ON ai_predictions(match_id);

-- 资深大拿表
CREATE TABLE IF NOT EXISTS experts (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    avatar_url TEXT NOT NULL DEFAULT '',
    title VARCHAR(50) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    win_rate NUMERIC(5,2) NOT NULL DEFAULT 0,
    total_predictions INT NOT NULL DEFAULT 0,
    correct_predictions INT NOT NULL DEFAULT 0,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 大拿预测表
CREATE TABLE IF NOT EXISTS expert_predictions (
    id BIGSERIAL PRIMARY KEY,
    expert_id BIGINT NOT NULL REFERENCES experts(id),
    match_id VARCHAR(64) NOT NULL,
    lottery_type VARCHAR(10) NOT NULL,
    sub_type VARCHAR(10) NOT NULL,
    bet_code VARCHAR(20) NOT NULL,
    handicap NUMERIC(5,2) NOT NULL DEFAULT 0,
    confidence NUMERIC(5,4) NOT NULL DEFAULT 0,
    reasoning TEXT NOT NULL DEFAULT '',
    is_correct BOOLEAN,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(expert_id, match_id, lottery_type, sub_type)
);

CREATE INDEX IF NOT EXISTS idx_expert_pred_match ON expert_predictions(match_id);

-- 积分变动日志表
CREATE TABLE IF NOT EXISTS credit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    change_amount INT NOT NULL DEFAULT 0,
    balance_after INT NOT NULL DEFAULT 0,
    reason VARCHAR(50) NOT NULL DEFAULT '',
    ref_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_credit_logs_user_id ON credit_logs(user_id);

-- 球队信息缓存表（爬虫抓取后缓存，为 logo 补全和后续查询提供关联）
CREATE TABLE IF NOT EXISTS teams (
    id BIGSERIAL PRIMARY KEY,
    team_id VARCHAR(64) UNIQUE NOT NULL,
    cn_name VARCHAR(100) NOT NULL DEFAULT '',
    cn_alias VARCHAR(100) NOT NULL DEFAULT '',
    logo_url TEXT NOT NULL DEFAULT '',
    logo_validated BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_teams_team_id ON teams(team_id);
CREATE INDEX IF NOT EXISTS idx_teams_name ON teams(cn_name);

-- 联赛/杯赛信息缓存表
CREATE TABLE IF NOT EXISTS leagues (
    id BIGSERIAL PRIMARY KEY,
    league_id VARCHAR(64) UNIQUE NOT NULL,
    cn_name VARCHAR(100) NOT NULL DEFAULT '',
    cn_alias VARCHAR(100) NOT NULL DEFAULT '',
    color VARCHAR(20) NOT NULL DEFAULT '',
    level VARCHAR(10) NOT NULL DEFAULT '',
    logo_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_leagues_league_id ON leagues(league_id);

-- updated_at 自动更新触发器
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS trigger_matches_updated ON matches;
DROP TRIGGER IF EXISTS trigger_results_updated ON match_results;
DROP TRIGGER IF EXISTS trigger_users_updated ON users;
DROP TRIGGER IF EXISTS trigger_predictions_updated ON predictions;
CREATE TRIGGER trigger_matches_updated BEFORE UPDATE ON matches FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trigger_results_updated BEFORE UPDATE ON match_results FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trigger_users_updated BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trigger_predictions_updated BEFORE UPDATE ON predictions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
