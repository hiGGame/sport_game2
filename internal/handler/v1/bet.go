package v1

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"sport_game2/internal/errs"
	"sport_game2/internal/middleware"
	"sport_game2/internal/service/bet"
)

type BetHandler struct {
	svc *bet.Service
}

func NewBetHandler(svc *bet.Service) *BetHandler {
	return &BetHandler{svc: svc}
}

func (h *BetHandler) CreateBet(c *gin.Context) {
	userID := c.GetInt64("userId")

	var req bet.CreateBetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.AbortWithError(c, err)
		return
	}

	resp, err := h.svc.CreateBet(userID, &req)
	if err != nil {
		switch {
		case errors.Is(err, bet.ErrBetLocked):
			middleware.AbortWithError(c, errs.New(errs.CodeMatchLocked, err.Error(), http.StatusForbidden))
		case errors.Is(err, bet.ErrMatchStopped):
			middleware.AbortWithError(c, errs.New(errs.CodeMatchStopped, err.Error(), http.StatusForbidden))
		case errors.Is(err, bet.ErrMatchStarted):
			middleware.AbortWithError(c, errs.New(errs.CodeMatchStarted, err.Error(), http.StatusForbidden))
		case errors.Is(err, bet.ErrMatchFinished):
			middleware.AbortWithError(c, errs.New(errs.CodeMatchFinished, err.Error(), http.StatusForbidden))
		case errors.Is(err, bet.ErrMatchNotFound):
			middleware.AbortWithError(c, errs.New(errs.CodeNotFound, err.Error(), http.StatusNotFound))
		case errors.Is(err, bet.ErrInvalidSelection):
			middleware.AbortWithError(c, errs.New(errs.CodeValidation, err.Error(), http.StatusBadRequest))
		case errors.Is(err, bet.ErrDuplicateBet):
			middleware.AbortWithError(c, errs.New(errs.CodeDuplicateBet, err.Error(), http.StatusConflict))
		default:
			middleware.AbortWithError(c, err)
		}
		return
	}

	c.JSON(200, resp)
}

func (h *BetHandler) GetUserPredictions(c *gin.Context) {
	userID := c.GetInt64("userId")

	preds, err := h.svc.GetUserPredictionsEnriched(userID, 20)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}

	c.JSON(200, gin.H{"list": preds})
}

func (h *BetHandler) ShareSuccess(c *gin.Context) {
	userID := c.GetInt64("userId")

	resp, err := h.svc.ShareSuccess(userID)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}

	c.JSON(200, resp)
}

func (h *BetHandler) DailyPK(c *gin.Context) {
	userID := c.GetInt64("userId")

	resp, err := h.svc.GetPK(userID)
	if err != nil {
		log.Printf("[PK] user=%d ERROR: %v", userID, err)
		middleware.AbortWithError(c, err)
		return
	}
	if resp == nil {
		log.Printf("[PK] user=%d NO_BETS", userID)
		c.JSON(200, gin.H{"settled": false, "message": "该周期没有竞猜记录"})
		return
	}
	if resp.Winner == "" {
		log.Printf("[PK] user=%d WAITING", userID)
		c.JSON(200, gin.H{"settled": false, "message": "等待开奖结果"})
		return
	}

	log.Printf("[PK] user=%d READY winner=%s user=%d/%d expert=%d/%d ai=%d/%d",
		userID, resp.Winner, resp.UserWins, resp.UserTotal, resp.ExpertWins, resp.ExpertTotal, resp.AIWins, resp.AITotal)
	c.JSON(200, resp)
}
