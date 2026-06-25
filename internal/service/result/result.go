package result

import (
	"encoding/json"
	"fmt"
	"strings"

	"sport_game2/internal/adapter/apifox"
	"sport_game2/internal/model"
	"sport_game2/internal/repo"
	"sport_game2/internal/service/predictor"
)

type matchStore interface {
	GetMatchBetInfo(lotteryType, matchCode string) (*apifox.MatchBetInfo, error)
	GetMatchById(matchId string) (*apifox.MatchBetInfo, error)
	GetDrawList() ([]apifox.LotteryDrawHomeInfo, error)
	GetMatchResult(matchID string) (*repo.DrawResultData, error)
}

type predictionStore interface {
	GetPredictionsByMatch(matchID string) ([]repo.Prediction, error)
	GetAIPrediction(matchID string) ([]repo.AIPrediction, error)
	UpsertAIPrediction(a *repo.AIPrediction) error
	GetExpertPredictions(matchID string) ([]repo.ExpertPrediction, error)
	SettleAllForMatch(matchID string) (int, error)
}

type userStore interface {
	GetUserByID(id int64) (*repo.User, error)
}

type creditsManager interface {
	Deduct(userID int64, amount int, reason string, refID int64) (int, error)
	Refund(userID int64, amount int, reason string, refID int64) (int, error)
	Award(userID int64, amount int, reason string, refID int64) (int, error)
}

type Service struct {
	matchRepo      matchStore
	predRepo       predictionStore
	userRepo       userStore
	creditManager  creditsManager
	aiProvider     predictor.Provider
}

func NewService(matchRepo matchStore, predRepo predictionStore, userRepo userStore, cm creditsManager, aiProvider predictor.Provider) *Service {
	return &Service{
		matchRepo:     matchRepo,
		predRepo:      predRepo,
		userRepo:      userRepo,
		creditManager: cm,
		aiProvider:    aiProvider,
	}
}

func (s *Service) GetDrawHomeList() (*apifox.GetLotteryDrawHomeListReply, error) {
	list, err := s.matchRepo.GetDrawList()
	if err != nil {
		return nil, err
	}
	return &apifox.GetLotteryDrawHomeListReply{List: list}, nil
}

type MatchPredictionView struct {
	MatchID           string                       `json:"matchId"`
	MatchInfo         apifox.MatchBetInfo          `json:"matchInfo"`
	DrawInfo          *apifox.DrawInfoReply        `json:"drawInfo,omitempty"`
	AIPredictions     []model.PredictionResult     `json:"aiPredictions"`
	ExpertPredictions []model.ExpertPredictionView `json:"expertPredictions"`
	UserPredictions   []model.UserPredictionView   `json:"userPredictions"`
}

func (s *Service) GetMatchPrediction(matchID string) (*MatchPredictionView, error) {
	aiPreds, _ := s.predRepo.GetAIPrediction(matchID)

	var aiResults []model.PredictionResult
	for _, a := range aiPreds {
		aiResults = append(aiResults, model.PredictionResult{
			MatchID:    a.MatchID,
			LotteryType: a.LotteryType,
			SubType:    a.SubType,
			BetCode:    a.BetCode,
			Confidence: a.Confidence,
			Reasoning:  a.Reasoning,
			ModelName:  a.ModelName,
		})
	}

	if len(aiResults) == 0 && s.aiProvider != nil {
		if err := s.generateAIPrediction(matchID); err == nil {
			aiPreds, _ = s.predRepo.GetAIPrediction(matchID)
			for _, a := range aiPreds {
				aiResults = append(aiResults, model.PredictionResult{
					MatchID:    a.MatchID,
					LotteryType: a.LotteryType,
					SubType:    a.SubType,
					BetCode:    a.BetCode,
					Confidence: a.Confidence,
					Reasoning:  a.Reasoning,
					ModelName:  a.ModelName,
				})
			}
		}
	}

	expertPreds, _ := s.predRepo.GetExpertPredictions(matchID)
	var expertViews []model.ExpertPredictionView
	for _, ep := range expertPreds {
		expertViews = append(expertViews, model.ExpertPredictionView{
			ExpertName: ep.ExpertName,
			AvatarURL:  ep.AvatarURL,
			Title:      ep.Title,
			SubType:    ep.SubType,
			BetCode:    ep.BetCode,
			Confidence: ep.Confidence,
			Reasoning:  ep.Reasoning,
		})
	}

	userPreds, _ := s.predRepo.GetPredictionsByMatch(matchID)
	var userViews []model.UserPredictionView
	for _, up := range userPreds {
		user, _ := s.userRepo.GetUserByID(up.UserID)
		nickname := ""
		avatar := ""
		if user != nil {
			nickname = user.Nickname
			avatar = user.AvatarURL
		}
		userViews = append(userViews, model.UserPredictionView{
			UserID:    up.UserID,
			Nickname:  nickname,
			AvatarURL: avatar,
			SubType:   up.SubType,
			BetCode:   up.BetCode,
			Points:     up.Points,
		})
	}

	var drawInfo *apifox.DrawInfoReply
	resultData, _ := s.matchRepo.GetMatchResult(matchID)
	matchData, _ := s.matchRepo.GetMatchById(matchID)
	if resultData != nil && resultData.IsValid {
		homeLogo, awayLogo := "", ""
		homeTeamAlias, awayTeamAlias := "", ""
		homeTeamID, awayTeamID := "", ""
		if matchData != nil {
			homeLogo = matchData.HomeTeamInfo.LogoURL
			awayLogo = matchData.AwayTeamInfo.LogoURL
			homeTeamAlias = matchData.HomeTeamInfo.CnAlias
			awayTeamAlias = matchData.AwayTeamInfo.CnAlias
			homeTeamID = matchData.HomeTeamInfo.ID
			awayTeamID = matchData.AwayTeamInfo.ID
		}

		homeNT, awayNT := splitScore(resultData.NormalTimeScore)
		homeHT, awayHT := splitScore(resultData.HalfTimeScore)

		drawInfo = &apifox.DrawInfoReply{
			MatchID:      resultData.MatchID,
			MatchCode:    resultData.MatchCode,
			MatchTimeStr: resultData.MatchTimeStr,
			WeekDay:      resultData.WeekDay,
			HomeTeamInfo: apifox.TeamInfo{
				CnName:  resultData.HomeTeamName,
				CnAlias: homeTeamAlias,
				LogoURL: homeLogo,
				ID:      homeTeamID,
			},
			AwayTeamInfo: apifox.TeamInfo{
				CnName:  resultData.AwayTeamName,
				CnAlias: awayTeamAlias,
				LogoURL: awayLogo,
				ID:      awayTeamID,
			},
			IsValid: resultData.IsValid,
			HomeTeamScore: apifox.MatchScore{
				Score:           resultData.HomeScore,
				NormalTimeScore: homeNT,
				HalfTimeScore:   homeHT,
			},
			AwayTeamScore: apifox.MatchScore{
				Score:           resultData.AwayScore,
				NormalTimeScore: awayNT,
				HalfTimeScore:   awayHT,
			},
		}
		var gameDraws []apifox.GameDrawInfo
		if json.Unmarshal([]byte(resultData.GameDrawJSON), &gameDraws) == nil {
			drawInfo.GameDrawList = gameDraws
		}
	}

	return &MatchPredictionView{
		MatchID:           matchID,
		DrawInfo:          drawInfo,
		AIPredictions:     aiResults,
		ExpertPredictions: expertViews,
		UserPredictions:   userViews,
	}, nil
}

func (s *Service) generateAIPrediction(matchID string) error {
	match, err := s.matchRepo.GetMatchById(matchID)
	if err != nil || match == nil {
		return fmt.Errorf("match not found: %s", matchID)
	}
	results, err := s.aiProvider.Predict(match)
	if err != nil {
		return fmt.Errorf("ai predict: %w", err)
	}
	for _, r := range results {
		if err := s.predRepo.UpsertAIPrediction(&repo.AIPrediction{
			MatchID: r.MatchID, LotteryType: r.LotteryType, SubType: r.SubType,
			BetCode: r.BetCode, Confidence: r.Confidence, Reasoning: r.Reasoning, ModelName: r.ModelName,
		}); err != nil {
			return fmt.Errorf("upsert ai prediction: %w", err)
		}
	}
	return nil
}

type SettlementResult struct {
	MatchID     string
	Settled     int
	Refunded    int
}

func (s *Service) SettleMatch(matchID string) (*SettlementResult, error) {
	n, err := s.predRepo.SettleAllForMatch(matchID)
	if err != nil {
		return nil, err
	}
	return &SettlementResult{MatchID: matchID, Settled: n}, nil
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
