package bet

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"sport_game2/internal/adapter/apifox"
	"sport_game2/internal/repo"
)

var (
	ErrMatchNotFound    = errors.New("赛事未找到")
	ErrBetLocked        = errors.New("距开赛时间不足,竞猜已锁定")
	ErrMatchStopped     = errors.New("该赛事已停止销售")
	ErrMatchStarted     = errors.New("赛事已经开始")
	ErrMatchFinished    = errors.New("赛事已经结束")
	ErrInvalidSelection = errors.New("无效的投注选项")
	ErrDuplicateBet     = errors.New("该比赛已投注过,不能重复投注")
)

type matchStore interface {
	GetMatchById(matchId string) (*apifox.MatchBetInfo, error)
	GetMatchResult(matchId string) (*repo.DrawResultData, error)
}

type betStore interface {
	CreatePrediction(p *repo.Prediction) (int64, error)
	GetPredictionsByUser(userID int64, limit int) ([]repo.Prediction, error)
	GetAIPrediction(matchID string) ([]repo.AIPrediction, error)
	GetExpertPredictions(matchID string) ([]repo.ExpertPrediction, error)
	CheckDuplicatePrediction(userID int64, matchID, lotteryType, subType string) (bool, error)
	GetPredictionsByOpenIDInRange(openID, fromTime, toTime string) ([]repo.Prediction, error)
	GetPredictionsByUserInRange(userID int64, fromTime, toTime string) ([]repo.Prediction, error)
	GetAllPredictionsInRange(fromTime, toTime string) ([]repo.Prediction, error)
}

type creditsManager interface {
	Deduct(userID int64, amount int, reason string, refID int64) (int, error)
	Add(userID int64, amount int, reason string) (int, error)
	IncrementTotalBets(userID int64) error
}

const (
	defaultLotteryType = "227"
	defaultSubType     = "6"
	defaultPoints      = 2
	shareBonus         = 2

	// PKTimezone 是竞猜时间判定使用的固定时区 (Asia/Shanghai)。
	// 竞彩周期以每日 11:00 为分界线 (前日 11:00 ~ 当日 11:00),
	// Go 端用此时区计算昨天竞彩周期的起止时间,传给 SQL 做字符串范围比较,
	// 与 GetDrawList 的 11:00 周期逻辑保持一致。
	PKTimezone = "Asia/Shanghai"

	// 机器人用户在 users.open_id 中的业务标识。由 ensureRobotUser 保底创建,
	// GetDailyPK 通过这些标识查询机器人昨天的已结算竞猜。
	robotExpertOpenID = "robot_expert"
	robotAIOpenID     = "robot_ai"
)

// winner 取值常量,用于 DailyPKResponse.Winner 字段。
const (
	winnerUser   = "user"
	winnerExpert = "expert"
	winnerAI     = "ai"
	winnerTie    = "tie"
)

type Service struct {
	matchRepo         matchStore
	betRepo           betStore
	creditManager     creditsManager
	lockMinutesBefore int
	pkLocation        *time.Location
}

func NewService(matchRepo matchStore, betRepo betStore, cm creditsManager, lockMinutesBefore int) *Service {
	loc, err := time.LoadLocation(PKTimezone)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return &Service{
		matchRepo:         matchRepo,
		betRepo:           betRepo,
		creditManager:     cm,
		lockMinutesBefore: lockMinutesBefore,
		pkLocation:        loc,
	}
}

type CreateBetRequest struct {
	MatchID string `json:"matchId"`
	BetCode string `json:"betCode"`
	Points  int    `json:"points"`
	SubType string `json:"subType"`
}

type CreateBetResponse struct {
	PredictionID int64 `json:"predictionId"`
	BalanceAfter int   `json:"balanceAfter"`
}

func (s *Service) CreateBet(userID int64, req *CreateBetRequest) (*CreateBetResponse, error) {
	points := req.Points
	if points <= 0 {
		points = defaultPoints
	}
	if req.BetCode == "" || req.MatchID == "" {
		return nil, ErrInvalidSelection
	}
	subType := req.SubType
	if subType == "" {
		subType = defaultSubType
	}

	match, err := s.matchRepo.GetMatchById(req.MatchID)
	if err != nil {
		return nil, fmt.Errorf("get match: %w", err)
	}
	if match == nil {
		return nil, ErrMatchNotFound
	}
	if match.LotteryInfo.IsStopSell {
		return nil, ErrMatchStopped
	}

	if ts := strings.TrimSpace(match.MatchInfo.MatchTimeStr); ts != "" {
		matchTime, err := time.ParseInLocation("2006-01-02 15:04:05", ts, s.pkLocation)
		if err != nil {
			return nil, fmt.Errorf("parse match time %q: %w", match.MatchInfo.MatchTimeStr, err)
		}
		if time.Now().After(matchTime) {
			return nil, ErrMatchStarted
		}
	}

	if s.lockMinutesBefore > 0 {
		betEndStr := strings.TrimSpace(match.LotteryInfo.BetEndTimeStr)
		if betEndStr != "" {
			betEndTime, err := time.ParseInLocation("2006-01-02 15:04:05", betEndStr, s.pkLocation)
			if err != nil {
				return nil, fmt.Errorf("parse bet end time %q: %w", match.LotteryInfo.BetEndTimeStr, err)
			}
			lockTime := betEndTime.Add(-time.Duration(s.lockMinutesBefore) * time.Minute)
			if time.Now().After(lockTime) {
				return nil, ErrBetLocked
			}
		}
	}

	validOption := false
	for _, bi := range match.LotteryInfo.BetInfos {
		if bi.SubType == subType {
			for _, opt := range bi.Options {
				if opt.BetCode == req.BetCode {
					validOption = true
					break
				}
			}
			break
		}
	}
	if !validOption {
		return nil, ErrInvalidSelection
	}

	if dup, err := s.betRepo.CheckDuplicatePrediction(userID, req.MatchID, defaultLotteryType, subType); err != nil {
		return nil, fmt.Errorf("check duplicate: %w", err)
	} else if dup {
		return nil, ErrDuplicateBet
	}

	pred := &repo.Prediction{
		UserID:      userID,
		MatchID:     req.MatchID,
		LotteryType: defaultLotteryType,
		MatchCode:   match.LotteryInfo.MatchCode,
		SubType:     subType,
		BetCode:     req.BetCode,
		Points:      points,
	}

	// Create prediction first; only increment total_bets on success to avoid
	// leaving a phantom count when the insert fails.
	predID, err := s.betRepo.CreatePrediction(pred)
	if err != nil {
		return nil, fmt.Errorf("create prediction: %w", err)
	}

	if err := s.creditManager.IncrementTotalBets(userID); err != nil {
		return nil, fmt.Errorf("increment total bets: %w", err)
	}

	return &CreateBetResponse{
		PredictionID: predID,
		BalanceAfter: 0,
	}, nil
}

func (s *Service) GetUserPredictions(userID int64, limit int) ([]repo.Prediction, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.betRepo.GetPredictionsByUser(userID, limit)
}

type EnrichedPrediction struct {
	ID          int64  `json:"id"`
	MatchID     string `json:"matchId"`
	HomeTeam    string `json:"homeTeam"`
	AwayTeam    string `json:"awayTeam"`
	LeagueName  string `json:"leagueName"`
	MatchTime   string `json:"matchTime"`
	SubType     string `json:"subType"`
	BetCode     string `json:"betCode"`
	Points      int    `json:"points"`
	Status      string `json:"status"`
	IsCorrect   *bool  `json:"isCorrect"`
	Winner      string `json:"winner,omitempty"`
	AIBetCode   string `json:"aiBetCode,omitempty"`
	AICorrect   *bool  `json:"aiCorrect,omitempty"`
	ExpertName  string `json:"expertName,omitempty"`
	ExpertBet   string `json:"expertBetCode,omitempty"`
	ExpertCorrect *bool `json:"expertCorrect,omitempty"`
}

func (s *Service) GetUserPredictionsEnriched(userID int64, limit int) ([]EnrichedPrediction, error) {
	preds, err := s.betRepo.GetPredictionsByUser(userID, limit)
	if err != nil {
		return nil, err
	}

	var result []EnrichedPrediction
	for _, p := range preds {
		ep := EnrichedPrediction{
			ID: p.ID, MatchID: p.MatchID, SubType: p.SubType, BetCode: p.BetCode,
			Points: p.Points, Status: p.Status, IsCorrect: p.IsCorrect,
		}

		match, _ := s.matchRepo.GetMatchById(p.MatchID)
		if match != nil {
			ep.HomeTeam = match.HomeTeamInfo.CnName
			ep.AwayTeam = match.AwayTeamInfo.CnName
			ep.LeagueName = match.TournamentInfo.CnName
			ep.MatchTime = match.MatchInfo.MatchTimeStr
		}

		if p.Status == "won" || p.Status == "lost" {
			ep.Winner = "tie"
			drawResult, _ := s.matchRepo.GetMatchResult(p.MatchID)
			if drawResult != nil && drawResult.IsValid {
				var gameDraws []apifox.GameDrawInfo
				json.Unmarshal([]byte(drawResult.GameDrawJSON), &gameDraws)
				drawMap := make(map[string]string)
				for _, gd := range gameDraws {
					drawMap[gd.SubType] = gd.BetCode
				}

				actualCode := drawMap[p.SubType]

				aiPreds, _ := s.betRepo.GetAIPrediction(p.MatchID)
				for _, ap := range aiPreds {
					if ap.SubType == p.SubType {
						ep.AIBetCode = ap.BetCode
						aiCorrect := ap.BetCode == actualCode
						ep.AICorrect = &aiCorrect
						if aiCorrect {
							ep.Winner = "ai"
						}
					}
				}

				expertPreds, _ := s.betRepo.GetExpertPredictions(p.MatchID)
				for _, xp := range expertPreds {
					if xp.SubType == p.SubType {
						ep.ExpertName = xp.ExpertName
						ep.ExpertBet = xp.BetCode
						expertCorrect := xp.BetCode == actualCode
						ep.ExpertCorrect = &expertCorrect
						if expertCorrect {
							if ep.Winner == "ai" {
								ep.Winner = "tie"
							} else {
								ep.Winner = "expert"
							}
						}
					}
				}

				if ep.IsCorrect != nil && *ep.IsCorrect {
					if ep.Winner == "ai" || ep.Winner == "expert" {
						ep.Winner = "tie"
					} else {
						ep.Winner = "user"
					}
				}
			}
		}

		result = append(result, ep)
	}

	return result, nil
}

type ShareResponse struct {
	BalanceAfter int `json:"balanceAfter"`
}

func (s *Service) ShareSuccess(userID int64) (*ShareResponse, error) {
	balance, err := s.creditManager.Add(userID, shareBonus, "share")
	if err != nil {
		return nil, fmt.Errorf("add share credits: %w", err)
	}
	return &ShareResponse{BalanceAfter: balance}, nil
}

type DailyPKResponse struct {
	UserTotal      int     `json:"userTotal"`
	UserWins       int     `json:"userWins"`
	UserAccuracy   float64 `json:"userAccuracy"`
	ExpertTotal    int     `json:"expertTotal"`
	ExpertWins     int     `json:"expertWins"`
	ExpertAccuracy float64 `json:"expertAccuracy"`
	AITotal        int     `json:"aiTotal"`
	AIWins         int     `json:"aiWins"`
	AIAccuracy     float64 `json:"aiAccuracy"`
	Winner         string  `json:"winner,omitempty"`
}

// GetPK 返回上一个竞彩周期(11:00~11:00)的三方 PK 对比结果。
//
// 规则:
//   - 用户在上一个竞彩周期无下注 → 返回 nil (handler 返回“该周期没有竞猜记录”)
//   - 用户 + 大拿(老委鬼) + AI狗 任一方的预测未全部开奖 → Winner 为 "" (handler 返回“等待开奖”)
//   - 全部已开奖 → 正常返回 PK 对比, Winner 不为空
//
// 竞彩周期: 每日 11:00 为分界线。
//   - 当前 >= 11:00 → 周期 = 昨天 11:00 ~ 今天 11:00
//   - 当前 <  11:00 → 周期 = 前天 11:00 ~ 昨天 11:00
func (s *Service) GetPK(userID int64) (*DailyPKResponse, error) {
	now := time.Now().In(s.pkLocation)
	pkDate := now.AddDate(0, 0, -1)
	if now.Hour() < 11 {
		pkDate = now.AddDate(0, 0, -2)
	}
	fromTime := pkDate.Format("2006-01-02") + " 11:00:00"
	toTime := pkDate.AddDate(0, 0, 1).Format("2006-01-02") + " 11:00:00"

	userPreds, err := s.betRepo.GetPredictionsByUserInRange(userID, fromTime, toTime)
	if err != nil {
		return nil, fmt.Errorf("get user predictions: %w", err)
	}
	if len(userPreds) == 0 {
		return nil, nil
	}

	expertPreds, err := s.betRepo.GetPredictionsByOpenIDInRange(robotExpertOpenID, fromTime, toTime)
	if err != nil {
		return nil, fmt.Errorf("get expert predictions: %w", err)
	}
	aiPreds, err := s.betRepo.GetPredictionsByOpenIDInRange(robotAIOpenID, fromTime, toTime)
	if err != nil {
		return nil, fmt.Errorf("get ai predictions: %w", err)
	}

	if hasPending(userPreds, expertPreds, aiPreds) {
		return &DailyPKResponse{}, nil
	}

	userTotal, userWins := countWins(userPreds)
	expertTotal, expertWins := countWins(expertPreds)
	aiTotal, aiWins := countWins(aiPreds)

	acc := func(wins, total int) float64 {
		if total == 0 {
			return 0
		}
		return float64(wins) / float64(total) * 100
	}

	resp := &DailyPKResponse{
		UserTotal:      userTotal,
		UserWins:       userWins,
		UserAccuracy:   acc(userWins, len(userPreds)),
		ExpertTotal:    expertTotal,
		ExpertWins:     expertWins,
		ExpertAccuracy: acc(expertWins, expertTotal),
		AITotal:        aiTotal,
		AIWins:         aiWins,
		AIAccuracy:     acc(aiWins, aiTotal),
	}
	resp.Winner = decidePKWinner(userWins, expertWins, aiWins)

	return resp, nil
}

// hasPending returns true if any prediction across all groups is still pending.
func hasPending(groups ...[]repo.Prediction) bool {
	for _, group := range groups {
		for _, p := range group {
			if p.Status == "pending" {
				return true
			}
		}
	}
	return false
}

// decidePKWinner 根据三方胜场数返回 PK 胜者标识:
//   - 单独领先者 -> 对应标识 (user/expert/ai)
//   - 有 >=2 方并列领先 -> winnerTie
func decidePKWinner(userWins, expertWins, aiWins int) string {
	maxWins := userWins
	wins := []int{userWins, expertWins, aiWins}
	for _, w := range wins {
		if w > maxWins {
			maxWins = w
		}
	}

	leaders := 0
	for _, w := range wins {
		if w == maxWins {
			leaders++
		}
	}
	if leaders > 1 {
		return winnerTie
	}

	switch maxWins {
	case userWins:
		return winnerUser
	case expertWins:
		return winnerExpert
	default:
		return winnerAI
	}
}

func countWins(preds []repo.Prediction) (total, wins int) {
	for _, p := range preds {
		if p.IsCorrect != nil && *p.IsCorrect {
			wins++
		}
	}
	return len(preds), wins
}
