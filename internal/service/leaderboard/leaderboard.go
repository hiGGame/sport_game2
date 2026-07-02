package leaderboard

import (
	"fmt"

	"sport_game2/internal/repo"
)

type store interface {
	GetUserByOpenID(openID string) (*repo.User, error)
	GetUserByID(id int64) (*repo.User, error)
	GetSettledUserPreds(userID int64) ([]repo.Prediction, error)
	GetAIPredStats() (total, correct int, err error)
	GetSettledAIPreds() ([]repo.Prediction, error)
}

type Service struct {
	repo store
}

func NewService(repo store) *Service {
	return &Service{repo: repo}
}

type LeaderboardEntry struct {
	Name       string  `json:"name"`
	Role       string  `json:"role"`
	Avatar     string  `json:"avatar"`
	Accuracy   float64 `json:"accuracy"`
	Wins       int     `json:"wins"`
	Total      int     `json:"total"`
	BestStreak int     `json:"bestStreak"`
}

const (
	robotExpertOpenID = "robot_expert"
	expertDisplayName = "老委鬼"
	expertAvatar      = "/web/avatar/expert/weiguitx.jpg"
)

func (s *Service) GetLeaderboard(userID int64) ([]LeaderboardEntry, error) {
	var entries []LeaderboardEntry

	// Expert (老委鬼) — backed by robot_expert user predictions.
	expertUser, err := s.repo.GetUserByOpenID(robotExpertOpenID)
	if err != nil {
		return nil, fmt.Errorf("get expert user: %w", err)
	}
	if expertUser != nil {
		ep, err := s.buildUserEntry(expertUser.ID, expertDisplayName, "expert", expertAvatar)
		if err != nil {
			return nil, fmt.Errorf("build expert entry: %w", err)
		}
		entries = append(entries, ep)
	}

	// AI 狗
	totalA, correctA, err := s.repo.GetAIPredStats()
	if err != nil {
		return nil, fmt.Errorf("get ai pred stats: %w", err)
	}
	aiPreds, err := s.repo.GetSettledAIPreds()
	if err != nil {
		return nil, fmt.Errorf("get settled ai preds: %w", err)
	}
	aiAccuracy := 0.0
	if totalA > 0 {
		aiAccuracy = float64(correctA) / float64(totalA) * 100
	}
	entries = append(entries, LeaderboardEntry{
		Name:       "AI狗",
		Role:       "ai",
		Avatar:     "/static/avatars/aiDog/dogtx.png",
		Accuracy:   aiAccuracy,
		Wins:       correctA,
		Total:      totalA,
		BestStreak: calcStreakAI(aiPreds),
	})

	// Current user.
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}
	if user != nil {
		name := user.Nickname
		if name == "" {
			name = fmt.Sprintf("用户%d", user.ID)
		}
		avatar := user.AvatarURL
		ue, err := s.buildUserEntry(userID, name, "user", avatar)
		if err != nil {
			return nil, fmt.Errorf("build user entry: %w", err)
		}
		entries = append(entries, ue)
	}

	return entries, nil
}

func (s *Service) buildUserEntry(userID int64, name, role, avatar string) (LeaderboardEntry, error) {
	preds, err := s.repo.GetSettledUserPreds(userID)
	if err != nil {
		return LeaderboardEntry{}, fmt.Errorf("get settled preds: %w", err)
	}
	wins := 0
	for _, p := range preds {
		if p.IsCorrect != nil && *p.IsCorrect {
			wins++
		}
	}
	accuracy := 0.0
	totalU := len(preds)
	if totalU > 0 {
		accuracy = float64(wins) / float64(totalU) * 100
	}
	return LeaderboardEntry{
		Name:       name,
		Role:       role,
		Avatar:     avatar,
		Accuracy:   accuracy,
		Wins:       wins,
		Total:      totalU,
		BestStreak: calcStreak(preds),
	}, nil
}

func calcStreak(preds []repo.Prediction) int {
	maxStreak, cur := 0, 0
	for _, p := range preds {
		if p.IsCorrect != nil && *p.IsCorrect {
			cur++
			if cur > maxStreak {
				maxStreak = cur
			}
		} else {
			cur = 0
		}
	}
	return maxStreak
}

func calcStreakAI(preds []repo.Prediction) int {
	maxStreak, cur := 0, 0
	for _, p := range preds {
		if p.Status == "won" {
			cur++
			if cur > maxStreak {
				maxStreak = cur
			}
		} else {
			cur = 0
		}
	}
	return maxStreak
}
