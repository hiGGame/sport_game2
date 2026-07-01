package repo

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type User struct {
	ID        int64
	OpenID    string
	UnionID   string
	Nickname  string
	AvatarURL string
	Phone     string
	Credits   int
	TotalBets int
	Wins      int
	Status    int
}

type Prediction struct {
	ID           int64
	UserID       int64
	MatchID      string
	LotteryType  string
	MatchCode    string
	SubType      string
	BetCode      string
	Handicap     float64
	Points       int
	Status       string
	IsCorrect    *bool
}

type AIPrediction struct {
	ID           int64
	MatchID      string
	LotteryType  string
	SubType      string
	BetCode      string
	Confidence   float64
	Reasoning    string
	ModelName    string
}

type Expert struct {
	ID        int64
	Name      string
	AvatarURL string
	Title     string
	Description string
	WinRate   float64
}

type ExpertPrediction struct {
	ID          int64
	ExpertID    int64
	MatchID     string
	LotteryType string
	SubType     string
	BetCode     string
	Handicap    float64
	Confidence  float64
	Reasoning   string
	IsCorrect   *bool
	ExpertName  string
	AvatarURL   string
	Title       string
}

func (db *DB) GetUserByOpenID(openID string) (*User, error) {
	row := db.QueryRow(`SELECT id, open_id, COALESCE(union_id,''), COALESCE(nickname,''), COALESCE(avatar_url,''),
		COALESCE(phone,''), credits, total_bets, wins, status FROM users WHERE open_id = $1`, openID)

	var u User
	err := row.Scan(&u.ID, &u.OpenID, &u.UnionID, &u.Nickname, &u.AvatarURL, &u.Phone, &u.Credits, &u.TotalBets, &u.Wins, &u.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

func (db *DB) CreateUser(openID, unionID string, initialCredits int) (*User, error) {
	var u User
	err := db.QueryRow(`INSERT INTO users (open_id, union_id, credits) VALUES ($1, $2, $3)
		RETURNING id, open_id, COALESCE(union_id,''), COALESCE(nickname,''), COALESCE(avatar_url,''), COALESCE(phone,''), credits, total_bets, wins, status`,
		openID, unionID, initialCredits).Scan(
		&u.ID, &u.OpenID, &u.UnionID, &u.Nickname, &u.AvatarURL, &u.Phone, &u.Credits, &u.TotalBets, &u.Wins, &u.Status)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (db *DB) GetUserByID(id int64) (*User, error) {
	row := db.QueryRow(`SELECT id, open_id, COALESCE(union_id,''), COALESCE(nickname,''), COALESCE(avatar_url,''),
		COALESCE(phone,''), credits, total_bets, wins, status FROM users WHERE id = $1`, id)

	var u User
	err := row.Scan(&u.ID, &u.OpenID, &u.UnionID, &u.Nickname, &u.AvatarURL, &u.Phone, &u.Credits, &u.TotalBets, &u.Wins, &u.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

func (db *DB) IncrementTotalBets(userID int64) error {
	_, err := db.Exec("UPDATE users SET total_bets = total_bets + 1 WHERE id = $1", userID)
	if err != nil {
		return fmt.Errorf("increment total bets: %w", err)
	}
	return nil
}

func (db *DB) UpdateUserProfile(id int64, nickname, avatarURL string) error {
	_, err := db.Exec("UPDATE users SET nickname = $1, avatar_url = $2 WHERE id = $3", nickname, avatarURL, id)
	return err
}

func (db *DB) ResetAllCredits(defaultCredits int) (int64, error) {
	result, err := db.Exec("UPDATE users SET credits = $1", defaultCredits)
	if err != nil {
		return 0, fmt.Errorf("reset credits: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

func (db *DB) CreatePrediction(p *Prediction) (int64, error) {
	var id int64
	err := db.QueryRow(`INSERT INTO predictions (user_id, match_id, lottery_type, match_code, sub_type, bet_code, handicap, points, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending') RETURNING id`,
		p.UserID, p.MatchID, p.LotteryType, p.MatchCode, p.SubType, p.BetCode, p.Handicap, p.Points).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create prediction: %w", err)
	}
	return id, nil
}

func (db *DB) CheckDuplicatePrediction(userID int64, matchID, lotteryType, subType string) (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM predictions WHERE user_id = $1 AND match_id = $2 AND lottery_type = $3 AND sub_type = $4 AND status = 'pending'`,
		userID, matchID, lotteryType, subType).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check duplicate: %w", err)
	}
	return count > 0, nil
}

func (db *DB) RobotCreatePrediction(userID int64, matchID, lotteryType, matchCode, subType, betCode string) (int64, error) {
	var id int64
	err := db.QueryRow(`INSERT INTO predictions (user_id, match_id, lottery_type, match_code, sub_type, bet_code, handicap, points, status)
		VALUES ($1, $2, $3, $4, $5, $6, 0, 0, 'pending')
		ON CONFLICT (user_id, match_id, lottery_type, sub_type) WHERE status = 'pending' DO NOTHING
		RETURNING id`,
		userID, matchID, lotteryType, matchCode, subType, betCode).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("robot create prediction: %w", err)
	}
	return id, nil
}

func (db *DB) GetPredictionsByMatch(matchID string) ([]Prediction, error) {
	rows, err := db.Query(`SELECT id, user_id, match_id, lottery_type, match_code, sub_type, bet_code, handicap, points, status, is_correct
		FROM predictions WHERE match_id = $1 ORDER BY created_at DESC`, matchID)
	if err != nil {
		return nil, fmt.Errorf("get predictions: %w", err)
	}
	defer rows.Close()

	var list []Prediction
	for rows.Next() {
		var p Prediction
		err := rows.Scan(&p.ID, &p.UserID, &p.MatchID, &p.LotteryType, &p.MatchCode, &p.SubType, &p.BetCode, &p.Handicap, &p.Points, &p.Status, &p.IsCorrect)
		if err != nil {
			continue
		}
		list = append(list, p)
	}
	return list, nil
}

func (db *DB) GetPredictionsByUser(userID int64, limit int) ([]Prediction, error) {
	rows, err := db.Query(`SELECT id, user_id, match_id, lottery_type, match_code, sub_type, bet_code, handicap, points, status, is_correct
		FROM predictions WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get user predictions: %w", err)
	}
	defer rows.Close()

	var list []Prediction
	for rows.Next() {
		var p Prediction
		err := rows.Scan(&p.ID, &p.UserID, &p.MatchID, &p.LotteryType, &p.MatchCode, &p.SubType, &p.BetCode, &p.Handicap, &p.Points, &p.Status, &p.IsCorrect)
		if err != nil {
			continue
		}
		list = append(list, p)
	}
	return list, nil
}

func (db *DB) SettlePrediction(id int64, isCorrect bool) error {
	status := "lost"
	if isCorrect {
		status = "won"
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec("UPDATE predictions SET status = $1, is_correct = $2, settled_at = NOW() WHERE id = $3", status, isCorrect, id); err != nil {
		return fmt.Errorf("update prediction: %w", err)
	}
	if isCorrect {
		if _, err := tx.Exec("UPDATE users SET wins = wins + 1 WHERE id = (SELECT user_id FROM predictions WHERE id = $1)", id); err != nil {
			return fmt.Errorf("update user wins: %w", err)
		}
	}
	return tx.Commit()
}

func (db *DB) SettleExpertPredictions(matchID string, drawMap map[string]string) error {
	for subType, actualCode := range drawMap {
		_, err := db.Exec(`UPDATE expert_predictions SET is_correct = (bet_code = $1) WHERE match_id = $2 AND sub_type = $3 AND is_correct IS NULL`, actualCode, matchID, subType)
		if err != nil {
			return fmt.Errorf("settle expert predictions: %w", err)
		}
	}
	return nil
}

func (db *DB) SettleAIPredictions(matchID string, drawMap map[string]string) error {
	for subType, actualCode := range drawMap {
		_, err := db.Exec(`UPDATE ai_predictions SET is_correct = (bet_code = $1) WHERE match_id = $2 AND sub_type = $3 AND is_correct IS NULL`, actualCode, matchID, subType)
		if err != nil {
			return fmt.Errorf("settle ai predictions: %w", err)
		}
	}
	return nil
}

func (db *DB) SettleAllForMatch(matchID string) (int, error) {
	resultData, err := db.GetMatchResult(matchID)
	if err != nil {
		return 0, fmt.Errorf("get match result: %w", err)
	}
	if resultData == nil || !resultData.IsValid {
		return 0, fmt.Errorf("match result not available or invalid")
	}

	var gameDraws []struct {
		SubType string `json:"subType"`
		BetCode string `json:"betCode"`
	}
	if err := json.Unmarshal([]byte(resultData.GameDrawJSON), &gameDraws); err != nil {
		return 0, fmt.Errorf("parse game draws: %w", err)
	}

	drawMap := make(map[string]string)
	for _, gd := range gameDraws {
		drawMap[gd.SubType] = gd.BetCode
	}

	settled := 0
	rows, err := db.Query(`SELECT id, user_id, sub_type, bet_code FROM predictions WHERE match_id = $1 AND status = 'pending'`, matchID)
	if err != nil {
		return 0, fmt.Errorf("query pending predictions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var userID int64
		var subType, betCode string
		if err := rows.Scan(&id, &userID, &subType, &betCode); err != nil {
			continue
		}
		actualCode, ok := drawMap[subType]
		isCorrect := ok && betCode == actualCode
		status := "lost"
		if isCorrect {
			status = "won"
		}
		if _, err := db.Exec("UPDATE predictions SET status = $1, is_correct = $2, settled_at = NOW() WHERE id = $3", status, isCorrect, id); err != nil {
			continue
		}
		settled++
		if isCorrect {
			if _, err := db.Exec("UPDATE users SET wins = wins + 1 WHERE id = $1", userID); err != nil {
				return settled, fmt.Errorf("update user wins: %w", err)
			}
		}
	}

	db.SettleExpertPredictions(matchID, drawMap)
	db.SettleAIPredictions(matchID, drawMap)

	return settled, nil
}

func (db *DB) GetAIPrediction(matchID string) ([]AIPrediction, error) {
	rows, err := db.Query(`SELECT id, match_id, lottery_type, sub_type, bet_code, confidence, reasoning, model_name
		FROM ai_predictions WHERE match_id = $1`, matchID)
	if err != nil {
		return nil, fmt.Errorf("get ai predictions: %w", err)
	}
	defer rows.Close()

	var list []AIPrediction
	for rows.Next() {
		var a AIPrediction
		err := rows.Scan(&a.ID, &a.MatchID, &a.LotteryType, &a.SubType, &a.BetCode, &a.Confidence, &a.Reasoning, &a.ModelName)
		if err != nil {
			continue
		}
		list = append(list, a)
	}
	return list, nil
}

func (db *DB) UpsertAIPrediction(a *AIPrediction) error {
	_, err := db.Exec(`INSERT INTO ai_predictions (match_id, lottery_type, sub_type, bet_code, confidence, reasoning, model_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (match_id, lottery_type, sub_type) DO UPDATE SET
			bet_code = EXCLUDED.bet_code, confidence = EXCLUDED.confidence, reasoning = EXCLUDED.reasoning`,
		a.MatchID, a.LotteryType, a.SubType, a.BetCode, a.Confidence, a.Reasoning, a.ModelName)
	return err
}

func (db *DB) UpsertExpertPrediction(ep *ExpertPrediction) error {
	_, err := db.Exec(`INSERT INTO expert_predictions (expert_id, match_id, lottery_type, sub_type, bet_code, handicap, confidence, reasoning)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (expert_id, match_id, lottery_type, sub_type) DO UPDATE SET
			bet_code = EXCLUDED.bet_code, confidence = EXCLUDED.confidence, reasoning = EXCLUDED.reasoning`,
		ep.ExpertID, ep.MatchID, ep.LotteryType, ep.SubType, ep.BetCode, ep.Handicap, ep.Confidence, ep.Reasoning)
	return err
}

func (db *DB) GetExperts() ([]Expert, error) {
	rows, err := db.Query(`SELECT id, name, COALESCE(avatar_url,''), COALESCE(title,''), COALESCE(description,''), win_rate FROM experts WHERE status = 1`)
	if err != nil {
		return nil, fmt.Errorf("get experts: %w", err)
	}
	defer rows.Close()

	var list []Expert
	for rows.Next() {
		var e Expert
		err := rows.Scan(&e.ID, &e.Name, &e.AvatarURL, &e.Title, &e.Description, &e.WinRate)
		if err != nil {
			continue
		}
		list = append(list, e)
	}
	return list, nil
}

func (db *DB) GetExpertPredictions(matchID string) ([]ExpertPrediction, error) {
	rows, err := db.Query(`SELECT ep.id, ep.expert_id, ep.match_id, ep.lottery_type, ep.sub_type, ep.bet_code, ep.handicap, ep.confidence, ep.reasoning,
		e.name as expert_name, e.avatar_url, e.title
		FROM expert_predictions ep
		JOIN experts e ON ep.expert_id = e.id
		WHERE ep.match_id = $1`, matchID)
	if err != nil {
		return nil, fmt.Errorf("get expert predictions: %w", err)
	}
	defer rows.Close()

	var list []ExpertPrediction
	for rows.Next() {
		var ep ExpertPrediction
		err := rows.Scan(&ep.ID, &ep.ExpertID, &ep.MatchID, &ep.LotteryType, &ep.SubType, &ep.BetCode, &ep.Handicap, &ep.Confidence, &ep.Reasoning,
			&ep.ExpertName, &ep.AvatarURL, &ep.Title)
		if err != nil {
			continue
		}
		list = append(list, ep)
	}
	return list, nil
}

func (db *DB) GetMatchResult(matchID string) (*DrawResultData, error) {
	var d DrawResultData
	var gameDrawJSON string
	err := db.QueryRow(`SELECT match_id, match_code, match_time_str, week_day, home_team_name, away_team_name, league_name,
		home_score, away_score, normal_time_score, half_time_score, is_valid, game_draw_list, lottery_type
		FROM match_results WHERE match_id = $1`, matchID).
		Scan(&d.MatchID, &d.MatchCode, &d.MatchTimeStr, &d.WeekDay, &d.HomeTeamName, &d.AwayTeamName, &d.LeagueName,
			&d.HomeScore, &d.AwayScore, &d.NormalTimeScore, &d.HalfTimeScore,
			&d.IsValid, &gameDrawJSON, &d.LotteryType)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get match result: %w", err)
	}
	d.GameDrawJSON = gameDrawJSON
	return &d, nil
}

func (db *DB) GetTopUsers(limit int) ([]User, error) {
	rows, err := db.Query(`SELECT id, open_id, COALESCE(union_id,''), COALESCE(nickname,''), COALESCE(avatar_url,''), COALESCE(phone,''), credits, total_bets, wins, status FROM users WHERE total_bets > 0 ORDER BY wins DESC, ROUND(CASE WHEN total_bets > 0 THEN wins::float/total_bets ELSE 0 END, 4) DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("get top users: %w", err)
	}
	defer rows.Close()
	var list []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.OpenID, &u.UnionID, &u.Nickname, &u.AvatarURL, &u.Phone, &u.Credits, &u.TotalBets, &u.Wins, &u.Status); err != nil {
			continue
		}
		list = append(list, u)
	}
	return list, nil
}

func (db *DB) GetSettledUserPreds(userID int64) ([]Prediction, error) {
	rows, err := db.Query(`SELECT p.id, p.user_id, p.match_id, p.lottery_type, p.match_code, p.sub_type, p.bet_code, p.handicap, p.points, p.status, p.is_correct
		FROM predictions p JOIN matches m ON p.match_id = m.match_id AND p.lottery_type = m.lottery_type
		WHERE p.user_id = $1 AND p.status IN ('won','lost') ORDER BY m.match_time_str ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("get settled user predictions: %w", err)
	}
	defer rows.Close()
	var list []Prediction
	for rows.Next() {
		var p Prediction
		if err := rows.Scan(&p.ID, &p.UserID, &p.MatchID, &p.LotteryType, &p.MatchCode, &p.SubType, &p.BetCode, &p.Handicap, &p.Points, &p.Status, &p.IsCorrect); err != nil {
			continue
		}
		list = append(list, p)
	}
	return list, nil
}

func (db *DB) GetSettledExpertPreds(expertID int64) ([]ExpertPrediction, error) {
	rows, err := db.Query(`SELECT ep.id, ep.expert_id, ep.match_id, ep.lottery_type, ep.sub_type, ep.bet_code, ep.handicap, ep.confidence, ep.reasoning, ep.is_correct, e.name, e.avatar_url, e.title
		FROM expert_predictions ep JOIN matches m ON ep.match_id = m.match_id AND ep.lottery_type = m.lottery_type JOIN experts e ON ep.expert_id = e.id
		WHERE ep.expert_id = $1 AND ep.is_correct IS NOT NULL ORDER BY m.match_time_str ASC`, expertID)
	if err != nil {
		return nil, fmt.Errorf("get settled expert predictions: %w", err)
	}
	defer rows.Close()
	var list []ExpertPrediction
	for rows.Next() {
		var ep ExpertPrediction
		if err := rows.Scan(&ep.ID, &ep.ExpertID, &ep.MatchID, &ep.LotteryType, &ep.SubType, &ep.BetCode, &ep.Handicap, &ep.Confidence, &ep.Reasoning, &ep.IsCorrect, &ep.ExpertName, &ep.AvatarURL, &ep.Title); err != nil {
			continue
		}
		list = append(list, ep)
	}
	return list, nil
}

func (db *DB) GetAIPredStats() (total, correct int, err error) {
	err = db.QueryRow(`SELECT COUNT(*) AS total, COALESCE(SUM(CASE WHEN ap.bet_code = gd.betCode THEN 1 ELSE 0 END), 0) AS correct
		FROM ai_predictions ap
		JOIN match_results mr ON ap.match_id = mr.match_id AND mr.is_valid = true,
		LATERAL (SELECT (jsonb_array_elements(mr.game_draw_list)->>'betCode') AS betCode, (jsonb_array_elements(mr.game_draw_list)->>'subType') AS subType) gd
		WHERE gd.subType = ap.sub_type`).Scan(&total, &correct)
	if err != nil {
		return 0, 0, fmt.Errorf("get ai pred stats: %w", err)
	}
	return total, correct, nil
}

func (db *DB) GetSettledAIPreds() ([]Prediction, error) {
	rows, err := db.Query(`SELECT ap.match_id, ap.lottery_type, ap.sub_type, ap.betCode, ap.confidence,
		COALESCE(gd.betCode = ap.betCode, false) AS is_correct
		FROM ai_predictions ap
		LEFT JOIN match_results mr ON ap.match_id = mr.match_id AND mr.is_valid = true,
		LATERAL (SELECT (jsonb_array_elements(mr.game_draw_list)->>'betCode') AS betCode, (jsonb_array_elements(mr.game_draw_list)->>'subType') AS subType) gd
		WHERE gd.subType = ap.sub_type
		ORDER BY mr.match_time_str ASC, ap.match_id`)
	if err != nil {
		return nil, fmt.Errorf("get settled ai predictions: %w", err)
	}
	defer rows.Close()
	var list []Prediction
	for rows.Next() {
		var p Prediction
		var correct bool
		if err := rows.Scan(&p.MatchID, &p.LotteryType, &p.SubType, &p.BetCode, &p.Handicap, &correct); err != nil {
			continue
		}
		if correct {
			p.Status = "won"
		} else {
			p.Status = "lost"
		}
		list = append(list, p)
	}
	return list, nil
}

func (db *DB) GetSettledByOpenID(openID, fromTime, toTime string) ([]Prediction, error) {
	rows, err := db.Query(`SELECT p.id, p.user_id, p.match_id, p.lottery_type, p.match_code, p.sub_type, p.bet_code, p.handicap, p.points, p.status, p.is_correct
		FROM predictions p
		JOIN matches m ON p.match_id = m.match_id AND p.lottery_type = m.lottery_type
		WHERE p.user_id = (SELECT id FROM users WHERE open_id = $1) AND m.match_time_str >= $2 AND m.match_time_str < $3 AND p.status IN ('won','lost') AND p.is_correct IS NOT NULL`, openID, fromTime, toTime)
	if err != nil {
		return nil, fmt.Errorf("get settled predictions by openID: %w", err)
	}
	defer rows.Close()
	var list []Prediction
	for rows.Next() {
		var p Prediction
		if err := rows.Scan(&p.ID, &p.UserID, &p.MatchID, &p.LotteryType, &p.MatchCode, &p.SubType, &p.BetCode, &p.Handicap, &p.Points, &p.Status, &p.IsCorrect); err != nil {
			return nil, fmt.Errorf("scan settled prediction by openID: %w", err)
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate settled predictions by openID: %w", err)
	}
	return list, nil
}

func (db *DB) GetSettledByUserID(userID int64, fromTime, toTime string) ([]Prediction, error) {
	rows, err := db.Query(`SELECT p.id, p.user_id, p.match_id, p.lottery_type, p.match_code, p.sub_type, p.bet_code, p.handicap, p.points, p.status, p.is_correct
		FROM predictions p
		JOIN matches m ON p.match_id = m.match_id AND p.lottery_type = m.lottery_type
		WHERE p.user_id = $1 AND m.match_time_str >= $2 AND m.match_time_str < $3 AND p.status IN ('won','lost') AND p.is_correct IS NOT NULL`, userID, fromTime, toTime)
	if err != nil {
		return nil, fmt.Errorf("get settled predictions by userID: %w", err)
	}
	defer rows.Close()
	var list []Prediction
	for rows.Next() {
		var p Prediction
		if err := rows.Scan(&p.ID, &p.UserID, &p.MatchID, &p.LotteryType, &p.MatchCode, &p.SubType, &p.BetCode, &p.Handicap, &p.Points, &p.Status, &p.IsCorrect); err != nil {
			return nil, fmt.Errorf("scan settled prediction by userID: %w", err)
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate settled predictions by userID: %w", err)
	}
	return list, nil
}

func (db *DB) GetRobotPredictStats(openID string) (exists bool, todayTotal, todayPending int, err error) {
	var userID int64
	err = db.QueryRow(`SELECT id FROM users WHERE open_id = $1`, openID).Scan(&userID)
	if err == sql.ErrNoRows {
		return false, 0, 0, nil
	}
	if err != nil {
		return false, 0, 0, fmt.Errorf("get robot user: %w", err)
	}
	err = db.QueryRow(`SELECT count(*) FROM predictions WHERE user_id = $1 AND created_at::date = CURRENT_DATE`, userID).Scan(&todayTotal)
	if err != nil {
		return true, 0, 0, nil
	}
	db.QueryRow(`SELECT count(*) FROM predictions WHERE user_id = $1 AND created_at::date = CURRENT_DATE AND status = 'pending'`, userID).Scan(&todayPending)
	return true, todayTotal, todayPending, nil
}

type DrawResultData struct {
	MatchID           string
	MatchCode         string
	MatchTimeStr     string
	WeekDay          string
	HomeTeamName     string
	AwayTeamName     string
	LeagueName       string
	HomeScore        string
	AwayScore        string
	NormalTimeScore  string
	HalfTimeScore    string
	IsValid          bool
	GameDrawJSON     string
	LotteryType      string
}
