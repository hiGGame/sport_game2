package v1

import (
	"github.com/gin-gonic/gin"

	"sport_game2/internal/errs"
	"sport_game2/internal/middleware"
	"sport_game2/internal/service/auth"
)

type AuthHandler struct {
	svc *auth.Service
}

func NewAuthHandler(svc *auth.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) LoginByWechat(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.AbortWithError(c, err)
		return
	}

	if req.Code == "" {
		middleware.AbortWithError(c, errs.New(errs.CodeValidation, "code is required", 400))
		return
	}

	resp, err := h.svc.Login(&req)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}

	c.JSON(200, resp)
}

func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	userID := c.GetInt64("userId")
	user, err := h.svc.GetUser(userID)
	if err != nil || user == nil {
		middleware.AbortWithError(c, errs.ErrNotFound)
		return
	}

	c.JSON(200, gin.H{
		"id":        user.ID,
		"openId":    user.OpenID,
		"nickname":  user.Nickname,
		"avatarUrl": user.AvatarURL,
		"credits":   user.Credits,
		"totalBets": user.TotalBets,
		"wins":      user.Wins,
	})
}

func (h *AuthHandler) UpdateUserInfo(c *gin.Context) {
	userID := c.GetInt64("userId")
	var req struct {
		Nickname  string `json:"nickname"`
		AvatarURL string `json:"avatarUrl"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.AbortWithError(c, err)
		return
	}

	if err := h.svc.UpdateProfile(userID, req.Nickname, req.AvatarURL); err != nil {
		middleware.AbortWithError(c, err)
		return
	}

	c.JSON(200, gin.H{"success": true})
}

func (h *AuthHandler) DevLogin(c *gin.Context) {
	var req auth.DevLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.UserNum = 1
	}

	resp, err := h.svc.DevLogin(&req)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}

	c.JSON(200, resp)
}
