package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"sport_game2/internal/middleware"
	"sport_game2/internal/service/leaderboard"
)

type LeaderboardHandler struct {
	svc *leaderboard.Service
}

func NewLeaderboardHandler(svc *leaderboard.Service) *LeaderboardHandler {
	return &LeaderboardHandler{svc: svc}
}

func (h *LeaderboardHandler) GetLeaderboard(c *gin.Context) {
	userID := c.GetInt64("userId")

	entries, err := h.svc.GetLeaderboard(userID)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": entries})
}

func (h *LeaderboardHandler) GetAvatars(c *gin.Context) {
	c.JSON(http.StatusOK, AvatarCacheInstance.Get())
}
