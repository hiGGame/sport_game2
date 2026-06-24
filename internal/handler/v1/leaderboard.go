package v1

import (
	"github.com/gin-gonic/gin"

	"sport_game2/internal/service/leaderboard"
)

type LeaderboardHandler struct {
	svc *leaderboard.Service
}

func NewLeaderboardHandler(svc *leaderboard.Service) *LeaderboardHandler {
	return &LeaderboardHandler{svc: svc}
}

func (h *LeaderboardHandler) GetLeaderboard(c *gin.Context) {
	entries, err := h.svc.GetLeaderboard()
	if err != nil {
		c.JSON(500, gin.H{"code": 50000, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"list": entries})
}

func (h *LeaderboardHandler) GetAvatars(c *gin.Context) {
	c.JSON(200, AvatarCacheInstance.Get())
}
