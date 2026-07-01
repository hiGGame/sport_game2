package repo

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"sport_game2/internal/adapter/apifox"

	_ "github.com/lib/pq"
)

var shanghaiLoc = time.FixedZone("CST", 8*3600)

func joinScore(home, away string) string {
	if home == "" && away == "" {
		return ""
	}
	return home + ":" + away
}

func splitScore(raw string) (string, string) {
	if raw == "" {
		return "", ""
	}
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return raw, ""
}

type DB struct {
	*sql.DB
}

func NewDB(dsn string, maxOpenConns, maxIdleConns int) (*DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if maxOpenConns > 0 {
		db.SetMaxOpenConns(maxOpenConns)
	}
	if maxIdleConns > 0 {
		db.SetMaxIdleConns(maxIdleConns)
	}
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &DB{db}, nil
}

func (db *DB) UpsertMatch(m apifox.MatchBetInfo) error {
	betInfos, _ := json.Marshal(m.LotteryInfo.BetInfos)
	rawData, _ := json.Marshal(m)
	_, err := db.Exec(`
		INSERT INTO matches (match_id, sport_id, lottery_type, match_code, issue, match_time_str, bet_end_time_str,
			league_id, league_name, league_alias, league_color, league_level, league_logo,
			home_team_name, home_team_alias, home_team_logo, home_team_rank,
			away_team_name, away_team_alias, away_team_logo, away_team_rank,
			bet_infos, is_stop_sell, status, raw_data, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,NOW())
		ON CONFLICT (match_id, lottery_type) DO UPDATE SET
			bet_infos = EXCLUDED.bet_infos,
			raw_data = EXCLUDED.raw_data,
			is_stop_sell = EXCLUDED.is_stop_sell,
			status = EXCLUDED.status,
			updated_at = NOW()
	`,
		m.MatchInfo.MatchID, m.MatchInfo.SportID, m.LotteryInfo.LotteryType, m.LotteryInfo.MatchCode,
		m.LotteryInfo.Issue, m.MatchInfo.MatchTimeStr, m.LotteryInfo.BetEndTimeStr,
		m.TournamentInfo.ID, m.TournamentInfo.CnName, m.TournamentInfo.CnAlias, m.TournamentInfo.Color,
		m.TournamentInfo.Level, m.TournamentInfo.LogoURL,
		m.HomeTeamInfo.CnName, m.HomeTeamInfo.CnAlias, m.HomeTeamInfo.LogoURL, m.HomeTeamInfo.TournamentRank,
		m.AwayTeamInfo.CnName, m.AwayTeamInfo.CnAlias, m.AwayTeamInfo.LogoURL, m.AwayTeamInfo.TournamentRank,
		string(betInfos), m.LotteryInfo.IsStopSell, m.MatchInfo.Status, string(rawData),
	)
	if err != nil {
		return fmt.Errorf("upsert match: %w", err)
	}
	return nil
}

func (db *DB) UpsertMatchResult(r apifox.DrawInfoReply) error {
	if !hasSubType(r.GameDrawList, "1") {
		handicap := db.getMatchHandicap(r.MatchID)
		if handicap != 0 {
			homeScore := r.HomeTeamScore.Score
			awayScore := r.AwayTeamScore.Score
			if result, h := calcHHADResult(homeScore, awayScore, fmt.Sprintf("%g", handicap)); result != "" {
				r.GameDrawList = append(r.GameDrawList, apifox.GameDrawInfo{
					SubType:  "1",
					BetCode:  result,
					Odds:     0,
					Handicap: h,
				})
			}
		}
	}

	// sporttery results API 只返回日期不返回时间, 从 matches 表获取完整的 match_time_str。
	matchTimeStr := r.MatchTimeStr
	var dbMatchTime string
	err := db.QueryRow(`SELECT match_time_str FROM matches WHERE match_id = $1 AND lottery_type = '227'`, r.MatchID).Scan(&dbMatchTime)
	if err == nil && dbMatchTime != "" {
		matchTimeStr = dbMatchTime
	}

	gameDraw, _ := json.Marshal(r.GameDrawList)
	rawData, _ := json.Marshal(r)
	normalTimeScore := joinScore(r.HomeTeamScore.NormalTimeScore, r.AwayTeamScore.NormalTimeScore)
	halfTimeScore := joinScore(r.HomeTeamScore.HalfTimeScore, r.AwayTeamScore.HalfTimeScore)
	_, err = db.Exec(`
		INSERT INTO match_results (match_id, match_code, issue, match_time_str, week_day,
			home_team_name, away_team_name, league_name,
			home_score, away_score, normal_time_score, half_time_score,
			is_valid, game_draw_list, raw_data, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NOW())
		ON CONFLICT (match_id) DO UPDATE SET
			match_time_str = EXCLUDED.match_time_str,
			home_score = EXCLUDED.home_score,
			away_score = EXCLUDED.away_score,
			normal_time_score = EXCLUDED.normal_time_score,
			half_time_score = EXCLUDED.half_time_score,
			is_valid = EXCLUDED.is_valid,
			game_draw_list = EXCLUDED.game_draw_list,
			raw_data = EXCLUDED.raw_data,
			updated_at = NOW()
	`,
		r.MatchID, r.MatchCode, "", matchTimeStr, r.WeekDay,
		r.HomeTeamInfo.CnName, r.AwayTeamInfo.CnName, r.TournamentInfo.CnName,
		r.HomeTeamScore.Score, r.AwayTeamScore.Score,
		normalTimeScore, halfTimeScore,
		r.IsValid, string(gameDraw), string(rawData),
	)
	if err != nil {
		return fmt.Errorf("upsert result: %w", err)
	}
	return nil
}

func (db *DB) getMatchHandicap(matchID string) float64 {
	var betInfosJSON sql.NullString
	err := db.QueryRow(`SELECT bet_infos FROM matches WHERE match_id = $1 AND lottery_type = '227'`, matchID).Scan(&betInfosJSON)
	if err != nil || !betInfosJSON.Valid {
		return 0
	}
	return getHHADHandicap(betInfosJSON.String)
}

func (db *DB) GetMatchBetList(lotteryType, subType string) ([]apifox.MatchBetInfo, error) {
	nowStr := time.Now().In(shanghaiLoc).Format("2006-01-02 15:04:05")
	query := `SELECT match_id, sport_id, lottery_type, match_code, issue, match_time_str, bet_end_time_str,
		league_id, league_name, league_alias, league_color, league_level, league_logo,
		home_team_name, home_team_alias, home_team_logo, home_team_rank,
		away_team_name, away_team_alias, away_team_logo, away_team_rank,
		bet_infos, is_stop_sell, status
		FROM matches WHERE match_time_str > $1`
	args := []interface{}{nowStr}
	argIdx := 2

	if lotteryType != "" {
		query += fmt.Sprintf(" AND lottery_type = $%d", argIdx)
		args = append(args, lotteryType)
		argIdx++
	}

	query += " ORDER BY match_time_str ASC, match_code ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query matches: %w", err)
	}
	defer rows.Close()

	var list []apifox.MatchBetInfo
	for rows.Next() {
		var m apifox.MatchBetInfo
		var betInfosJSON string
		var leagueID, homeTeamRank, awayTeamRank string

		err := rows.Scan(
			&m.MatchInfo.MatchID, &m.MatchInfo.SportID, &m.LotteryInfo.LotteryType, &m.LotteryInfo.MatchCode,
			&m.LotteryInfo.Issue, &m.MatchInfo.MatchTimeStr, &m.LotteryInfo.BetEndTimeStr,
			&leagueID, &m.TournamentInfo.CnName, &m.TournamentInfo.CnAlias, &m.TournamentInfo.Color,
			&m.TournamentInfo.Level, &m.TournamentInfo.LogoURL,
			&m.HomeTeamInfo.CnName, &m.HomeTeamInfo.CnAlias, &m.HomeTeamInfo.LogoURL, &homeTeamRank,
			&m.AwayTeamInfo.CnName, &m.AwayTeamInfo.CnAlias, &m.AwayTeamInfo.LogoURL, &awayTeamRank,
			&betInfosJSON, &m.LotteryInfo.IsStopSell, &m.MatchInfo.Status,
		)
		if err != nil {
			continue
		}

		m.TournamentInfo.ID = leagueID
		m.HomeTeamInfo.TournamentRank = homeTeamRank
		m.AwayTeamInfo.TournamentRank = awayTeamRank

		if err := json.Unmarshal([]byte(betInfosJSON), &m.LotteryInfo.BetInfos); err != nil {
			continue
		}

		if subType != "" {
			var filtered []apifox.BetInfo
			for _, bi := range m.LotteryInfo.BetInfos {
				if bi.SubType == subType {
					filtered = append(filtered, bi)
				}
			}
			m.LotteryInfo.BetInfos = filtered
		}

		list = append(list, m)
	}
	return list, nil
}

func (db *DB) GetMatchBetInfo(lotteryType, matchCode string) (*apifox.MatchBetInfo, error) {
	query := `SELECT match_id, sport_id, lottery_type, match_code, issue, match_time_str, bet_end_time_str,
		league_id, league_name, league_alias, league_color, league_level, league_logo,
		home_team_name, home_team_alias, home_team_logo, home_team_rank,
		away_team_name, away_team_alias, away_team_logo, away_team_rank,
		bet_infos, is_stop_sell, status
		FROM matches WHERE lottery_type = $1 AND match_code = $2`

	row := db.QueryRow(query, lotteryType, matchCode)

	var m apifox.MatchBetInfo
	var betInfosJSON string
	var leagueID, homeTeamRank, awayTeamRank string

	err := row.Scan(
		&m.MatchInfo.MatchID, &m.MatchInfo.SportID, &m.LotteryInfo.LotteryType, &m.LotteryInfo.MatchCode,
		&m.LotteryInfo.Issue, &m.MatchInfo.MatchTimeStr, &m.LotteryInfo.BetEndTimeStr,
		&leagueID, &m.TournamentInfo.CnName, &m.TournamentInfo.CnAlias, &m.TournamentInfo.Color,
		&m.TournamentInfo.Level, &m.TournamentInfo.LogoURL,
		&m.HomeTeamInfo.CnName, &m.HomeTeamInfo.CnAlias, &m.HomeTeamInfo.LogoURL, &homeTeamRank,
		&m.AwayTeamInfo.CnName, &m.AwayTeamInfo.CnAlias, &m.AwayTeamInfo.LogoURL, &awayTeamRank,
		&betInfosJSON, &m.LotteryInfo.IsStopSell, &m.MatchInfo.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("query match: %w", err)
	}

	m.TournamentInfo.ID = leagueID
	m.HomeTeamInfo.TournamentRank = homeTeamRank
	m.AwayTeamInfo.TournamentRank = awayTeamRank
	if err := json.Unmarshal([]byte(betInfosJSON), &m.LotteryInfo.BetInfos); err != nil {
		return nil, fmt.Errorf("unmarshal bet_infos: %w", err)
	}

	return &m, nil
}

func (db *DB) GetMatchById(matchId string) (*apifox.MatchBetInfo, error) {
	query := `SELECT match_id, sport_id, lottery_type, match_code, issue, match_time_str, bet_end_time_str,
		league_id, league_name, league_alias, league_color, league_level, league_logo,
		home_team_name, home_team_alias, home_team_logo, home_team_rank,
		away_team_name, away_team_alias, away_team_logo, away_team_rank,
		bet_infos, is_stop_sell, status
		FROM matches WHERE lottery_type = '227' AND match_id = $1
		ORDER BY match_time_str ASC LIMIT 1`

	row := db.QueryRow(query, matchId)

	var m apifox.MatchBetInfo
	var betInfosJSON string
	var leagueID, homeTeamRank, awayTeamRank string

	err := row.Scan(
		&m.MatchInfo.MatchID, &m.MatchInfo.SportID, &m.LotteryInfo.LotteryType, &m.LotteryInfo.MatchCode,
		&m.LotteryInfo.Issue, &m.MatchInfo.MatchTimeStr, &m.LotteryInfo.BetEndTimeStr,
		&leagueID, &m.TournamentInfo.CnName, &m.TournamentInfo.CnAlias, &m.TournamentInfo.Color,
		&m.TournamentInfo.Level, &m.TournamentInfo.LogoURL,
		&m.HomeTeamInfo.CnName, &m.HomeTeamInfo.CnAlias, &m.HomeTeamInfo.LogoURL, &homeTeamRank,
		&m.AwayTeamInfo.CnName, &m.AwayTeamInfo.CnAlias, &m.AwayTeamInfo.LogoURL, &awayTeamRank,
		&betInfosJSON, &m.LotteryInfo.IsStopSell, &m.MatchInfo.Status,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query match: %w", err)
	}

	m.TournamentInfo.ID = leagueID
	m.HomeTeamInfo.TournamentRank = homeTeamRank
	m.AwayTeamInfo.TournamentRank = awayTeamRank
	if err := json.Unmarshal([]byte(betInfosJSON), &m.LotteryInfo.BetInfos); err != nil {
		return nil, fmt.Errorf("unmarshal bet_infos: %w", err)
	}

	return &m, nil
}

func (db *DB) GetDrawList() ([]apifox.LotteryDrawHomeInfo, error) {
	now := time.Now().In(shanghaiLoc)
	fromTime := now.AddDate(0, 0, -1).Format("2006-01-02") + " 11:00:00"
	toTime := now.Format("2006-01-02") + " 11:00:01"
	yesterdayDate := now.AddDate(0, 0, -1).Format("2006-01-02")
	query := `SELECT mr.match_id, mr.match_code, mr.match_time_str, mr.week_day,
		mr.home_team_name, mr.away_team_name, mr.league_name,
		mr.home_score, mr.away_score, mr.normal_time_score, mr.half_time_score,
		mr.is_valid, mr.game_draw_list, mr.lottery_type,
		m.home_team_logo, m.away_team_logo, m.bet_infos
		FROM match_results mr
		LEFT JOIN matches m ON mr.match_id = m.match_id AND m.lottery_type = '227'
		WHERE mr.is_valid = true
		  AND (
		    (m.match_id IS NOT NULL AND m.match_time_str >= $1 AND m.match_time_str < $2)
		    OR
		    (m.match_id IS NULL AND mr.match_time_str >= $3)
		  )
		ORDER BY COALESCE(m.match_time_str, mr.match_time_str) DESC LIMIT 50`

	rows, err := db.Query(query, fromTime, toTime, yesterdayDate)
	if err != nil {
		return nil, fmt.Errorf("query results: %w", err)
	}
	defer rows.Close()

	var list []apifox.LotteryDrawHomeInfo
	for rows.Next() {
		var d apifox.DrawInfoReply
		var gameDrawJSON string
		var lotteryType string
		var normalTimeScoreRaw, halfTimeScoreRaw string
		var betInfosJSON sql.NullString

		err := rows.Scan(
			&d.MatchID, &d.MatchCode, &d.MatchTimeStr, &d.WeekDay,
			&d.HomeTeamInfo.CnName, &d.AwayTeamInfo.CnName, &d.TournamentInfo.CnName,
			&d.HomeTeamScore.Score, &d.AwayTeamScore.Score,
			&normalTimeScoreRaw, &halfTimeScoreRaw,
			&d.IsValid, &gameDrawJSON, &lotteryType,
			&d.HomeTeamInfo.LogoURL, &d.AwayTeamInfo.LogoURL,
			&betInfosJSON,
		)
		if err != nil {
			continue
		}

		d.HomeTeamScore.NormalTimeScore, d.AwayTeamScore.NormalTimeScore = splitScore(normalTimeScoreRaw)
		d.HomeTeamScore.HalfTimeScore, d.AwayTeamScore.HalfTimeScore = splitScore(halfTimeScoreRaw)

		if err := json.Unmarshal([]byte(gameDrawJSON), &d.GameDrawList); err != nil {
			continue
		}

		d.GameDrawList = enrichRQSPF(d.GameDrawList, d.HomeTeamScore.Score, d.AwayTeamScore.Score, normalTimeScoreRaw, betInfosJSON)

		list = append(list, apifox.LotteryDrawHomeInfo{
			LotteryType:  lotteryType,
			LastDrawInfo: d,
		})
	}
	return list, nil
}

func hasSubType(list []apifox.GameDrawInfo, subType string) bool {
	for i := range list {
		if list[i].SubType == subType {
			return true
		}
	}
	return false
}

func parseScores(homeScoreRaw, awayScoreRaw, normalTimeScoreRaw string) (string, string) {
	if awayScoreRaw != "" {
		return homeScoreRaw, awayScoreRaw
	}
	home, away := splitScore(homeScoreRaw)
	if away != "" {
		return home, away
	}
	return splitScore(normalTimeScoreRaw)
}

func getHHADHandicap(betInfosJSON string) float64 {
	var betInfos []apifox.BetInfo
	if err := json.Unmarshal([]byte(betInfosJSON), &betInfos); err != nil {
		return 0
	}
	for i := range betInfos {
		if betInfos[i].SubType == "1" {
			return betInfos[i].Handicap
		}
	}
	return 0
}

func calcHHADResult(homeScore, awayScore, goalLine string) (string, float64) {
	if homeScore == "" || awayScore == "" || goalLine == "" {
		return "", 0
	}
	home, errH := strconv.ParseFloat(homeScore, 64)
	away, errA := strconv.ParseFloat(awayScore, 64)
	handicap, errG := strconv.ParseFloat(goalLine, 64)
	if errH != nil || errA != nil || errG != nil {
		return "", 0
	}
	adjusted := home + handicap
	switch {
	case adjusted > away:
		return "3", handicap
	case adjusted < away:
		return "0", handicap
	default:
		return "1", handicap
	}
}

func enrichRQSPF(gameDrawList []apifox.GameDrawInfo, homeScoreRaw, awayScoreRaw, normalTimeScoreRaw string, betInfosJSON sql.NullString) []apifox.GameDrawInfo {
	if hasSubType(gameDrawList, "1") || !betInfosJSON.Valid {
		return gameDrawList
	}
	handicap := getHHADHandicap(betInfosJSON.String)
	homeScore, awayScore := parseScores(homeScoreRaw, awayScoreRaw, normalTimeScoreRaw)
	if result, h := calcHHADResult(homeScore, awayScore, fmt.Sprintf("%g", handicap)); result != "" {
		gameDrawList = append(gameDrawList, apifox.GameDrawInfo{
			SubType:  "1",
			BetCode:  result,
			Odds:     0,
			Handicap: h,
		})
	}
	return gameDrawList
}

func (db *DB) LogSpiderJob(jobType string, status string, count int, errMsg string) {
	_, _ = db.Exec(`
		INSERT INTO spider_job_log (job_type, status, record_count, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, jobType, status, count, errMsg, time.Now())
}

func (db *DB) GetLastSpiderJobTime() (string, error) {
	var t string
	err := db.QueryRow(`SELECT COALESCE(to_char(created_at, 'YYYY-MM-DD HH24:MI:SS'), '') FROM spider_job_log ORDER BY created_at DESC LIMIT 1`).Scan(&t)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get last spider time: %w", err)
	}
	return t, nil
}

func (db *DB) UpsertTeam(teamID, name, alias, logoURL string) error {
	_, err := db.Exec(`INSERT INTO teams (team_id, cn_name, cn_alias, logo_url)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (team_id) DO UPDATE SET cn_name = EXCLUDED.cn_name, cn_alias = EXCLUDED.cn_alias,
			logo_url = CASE WHEN EXCLUDED.logo_url != '' THEN EXCLUDED.logo_url ELSE teams.logo_url END,
			updated_at = NOW()`,
		teamID, name, alias, logoURL)
	return err
}

func (db *DB) MarkTeamLogoInvalid(teamID string) error {
	_, err := db.Exec("UPDATE teams SET logo_url = '', logo_validated = true, updated_at = NOW() WHERE team_id = $1", teamID)
	return err
}

func (db *DB) MarkTeamLogoValid(teamID string) error {
	_, err := db.Exec("UPDATE teams SET logo_validated = true, updated_at = NOW() WHERE team_id = $1", teamID)
	return err
}

type TeamLogoCheck struct {
	TeamID  string
	LogoURL string
}

func (db *DB) GetUnvalidatedTeams() ([]TeamLogoCheck, error) {
	rows, err := db.Query("SELECT team_id, logo_url FROM teams WHERE logo_url != '' AND NOT logo_validated")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []TeamLogoCheck
	for rows.Next() {
		var t TeamLogoCheck
		if err := rows.Scan(&t.TeamID, &t.LogoURL); err != nil {
			continue
		}
		list = append(list, t)
	}
	return list, nil
}

func (db *DB) UpsertLeague(leagueID, name, alias, color, level string) error {
	_, err := db.Exec(`INSERT INTO leagues (league_id, cn_name, cn_alias, color, level) VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (league_id) DO UPDATE SET cn_name = EXCLUDED.cn_name, cn_alias = EXCLUDED.cn_alias,
			color = EXCLUDED.color, level = EXCLUDED.level, updated_at = NOW()`,
		leagueID, name, alias, color, level)
	return err
}

func (db *DB) BackfillMatchLogos() error {
	_, err := db.Exec(`UPDATE matches m SET
		home_team_logo = COALESCE(NULLIF(t1.logo_url, ''), m.home_team_logo),
		away_team_logo = COALESCE(NULLIF(t2.logo_url, ''), m.away_team_logo)
		FROM teams t1, teams t2
		WHERE m.home_team_logo = '' AND m.home_team_name = t1.cn_name
		  AND m.away_team_logo = '' AND m.away_team_name = t2.cn_name`)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) SyncMatchLogos() error {
	_, err := db.Exec(`UPDATE matches m SET
		home_team_logo = t1.logo_url,
		away_team_logo = t2.logo_url
		FROM teams t1, teams t2
		WHERE t1.logo_url != '' AND m.home_team_name = t1.cn_name
		  AND t2.logo_url != '' AND m.away_team_name = t2.cn_name
		  AND (m.home_team_logo != t1.logo_url OR m.away_team_logo != t2.logo_url)`)
	if err != nil {
		return err
	}
	return nil
}
