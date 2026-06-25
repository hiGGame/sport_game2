package leaderboard

import "sport_game2/internal/repo"

type store interface {
	GetUserByOpenID(openID string) (*repo.User, error)
	GetTopUsers(limit int) ([]repo.User, error)
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
	BestStreak int     `json:"bestStreak"`
}

const (
	robotExpertOpenID = "robot_expert"
	expertDisplayName = "老委鬼"
	expertAvatar      = "/web/avatar/expert/weiguitx.jpg"
)

func (s *Service) GetLeaderboard() ([]LeaderboardEntry, error) {
	var entries []LeaderboardEntry

	expertUser, _ := s.repo.GetUserByOpenID(robotExpertOpenID)
	if expertUser != nil {
		preds, _ := s.repo.GetSettledUserPreds(expertUser.ID)
		wins := 0
		for _, p := range preds {
			if p.IsCorrect != nil && *p.IsCorrect {
				wins++
			}
		}
		accuracy := 0.0
		if len(preds) > 0 {
			accuracy = float64(wins) / float64(len(preds)) * 100
		}
		entries = append(entries, LeaderboardEntry{
			Name:       expertDisplayName,
			Role:       "expert",
			Avatar:     expertAvatar,
			Accuracy:   accuracy,
			Wins:       wins,
			BestStreak: calcStreak(preds),
		})
	}

	totalA, correctA, _ := s.repo.GetAIPredStats()
	aiPreds, _ := s.repo.GetSettledAIPreds()
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
		BestStreak: calcStreakAI(aiPreds),
	})

	users, _ := s.repo.GetTopUsers(10)
	for _, u := range users {
		userPreds, _ := s.repo.GetSettledUserPreds(u.ID)
		wins := 0
		for _, p := range userPreds {
			if p.IsCorrect != nil && *p.IsCorrect {
				wins++
			}
		}
		accuracy := 0.0
		totalU := len(userPreds)
		if totalU > 0 {
			accuracy = float64(wins) / float64(totalU) * 100
		}
		name := u.Nickname
		if name == "" {
			name = "用户" + string(rune(u.ID+48))
		}
		avatar := u.AvatarURL
		entries = append(entries, LeaderboardEntry{
			Name:       name,
			Role:       "user",
			Avatar:     avatar,
			Accuracy:   accuracy,
			Wins:       wins,
			BestStreak: calcStreak(userPreds),
		})
	}

	return entries, nil
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
