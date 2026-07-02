package leaderboard

import (
	"fmt"

	"sport_game2/internal/repo"
)

type store interface {
	GetUserByOpenID(openID string) (*repo.User, error)
	GetUserByID(id int64) (*repo.User, error)
	GetSettledUserPreds(userID int64) ([]repo.Prediction, error)
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

	robotAIOpenID = "robot_ai"
	aiDisplayName = "AI狗"
	aiAvatar      = "/static/avatars/aiDog/dogtx.png"
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

	// AI 狗 — backed by robot_ai user predictions.
	aiUser, err := s.repo.GetUserByOpenID(robotAIOpenID)
	if err != nil {
		return nil, fmt.Errorf("get ai user: %w", err)
	}
	if aiUser != nil {
		ap, err := s.buildUserEntry(aiUser.ID, aiDisplayName, "ai", aiAvatar)
		if err != nil {
			return nil, fmt.Errorf("build ai entry: %w", err)
		}
		entries = append(entries, ap)
	}

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
