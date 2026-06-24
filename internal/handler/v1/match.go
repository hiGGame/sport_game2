package v1

import (
	"github.com/gin-gonic/gin"

	"sport_game2/internal/errs"
	"sport_game2/internal/middleware"
	"sport_game2/internal/service/match"
)

type MatchHandler struct {
	svc *match.Service
}

func NewMatchHandler(svc *match.Service) *MatchHandler {
	return &MatchHandler{svc: svc}
}

func (h *MatchHandler) GetMatchBetList(c *gin.Context) {
	lotteryType := c.Query("lotteryType")
	subType := c.Query("subType")
	sortType := c.DefaultQuery("sortType", "0")

	resp, err := h.svc.GetMatchBetList(lotteryType, subType, sortType)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}

	c.JSON(200, resp)
}

func (h *MatchHandler) GetMatchBetInfo(c *gin.Context) {
	lotteryType := c.Query("lotteryType")
	matchCode := c.Query("matchCode")

	if lotteryType == "" || matchCode == "" {
		middleware.AbortWithError(c, errs.New(errs.CodeValidation, "lotteryType and matchCode are required", 400))
		return
	}

	resp, err := h.svc.GetMatchBetInfo(lotteryType, matchCode)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}

	c.JSON(200, resp)
}

func (h *MatchHandler) GetHotMatchList(c *gin.Context) {
	resp, err := h.svc.GetHotMatchList()
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}

	c.JSON(200, resp)
}

func (h *MatchHandler) GetSpiderStatus(c *gin.Context) {
	t, err := h.svc.GetLastSpiderJobTime()
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	c.JSON(200, gin.H{"lastCrawlTime": t})
}
