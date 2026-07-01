package v1

import (
	"errors"

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
			middleware.AbortWithError(c, errs.New(errs.CodeMatchLocked, err.Error(), 500))
		case errors.Is(err, bet.ErrMatchStopped):
			middleware.AbortWithError(c, errs.New(errs.CodeMatchStopped, err.Error(), 500))
		case errors.Is(err, bet.ErrMatchStarted):
			middleware.AbortWithError(c, errs.New(errs.CodeMatchStarted, err.Error(), 500))
		case errors.Is(err, bet.ErrMatchFinished):
			middleware.AbortWithError(c, errs.New(errs.CodeMatchFinished, err.Error(), 500))
		case errors.Is(err, bet.ErrMatchNotFound):
			middleware.AbortWithError(c, errs.New(errs.CodeNotFound, err.Error(), 500))
		case errors.Is(err, bet.ErrInvalidSelection):
			middleware.AbortWithError(c, errs.New(errs.CodeValidation, err.Error(), 500))
		case errors.Is(err, bet.ErrDuplicateBet):
			middleware.AbortWithError(c, errs.New(errs.CodeDuplicateBet, err.Error(), 500))
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

	resp, err := h.svc.GetDailyPK(userID)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	if resp == nil {
		c.JSON(200, gin.H{"message": "昨天没有竞猜记录,不参与PK"})
		return
	}

	c.JSON(200, resp)
}
