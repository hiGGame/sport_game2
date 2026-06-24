package v1

import (
	"github.com/gin-gonic/gin"

	"sport_game2/internal/errs"
	"sport_game2/internal/middleware"
	"sport_game2/internal/service/result"
)

type ResultHandler struct {
	svc *result.Service
}

func NewResultHandler(svc *result.Service) *ResultHandler {
	return &ResultHandler{svc: svc}
}

func (h *ResultHandler) GetDrawHomeList(c *gin.Context) {
	resp, err := h.svc.GetDrawHomeList()
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	c.JSON(200, resp)
}

func (h *ResultHandler) GetMatchPrediction(c *gin.Context) {
	matchID := c.Param("matchId")
	if matchID == "" {
		middleware.AbortWithError(c, errs.New(errs.CodeValidation, "matchId is required", 400))
		return
	}

	resp, err := h.svc.GetMatchPrediction(matchID)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}

	c.JSON(200, resp)
}

func (h *ResultHandler) SettleMatch(c *gin.Context) {
	matchID := c.Param("matchId")

	resp, err := h.svc.SettleMatch(matchID)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}

	c.JSON(200, resp)
}
