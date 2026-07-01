package v1

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"sport_game2/internal/errs"
	"sport_game2/internal/middleware"
	"sport_game2/internal/service/auth"
)

type AuthHandler struct {
	svc           *auth.Service
	officialAppID string
	frontendURL   string
}

func NewAuthHandler(svc *auth.Service, officialAppID, frontendURL string) *AuthHandler {
	return &AuthHandler{svc: svc, officialAppID: officialAppID, frontendURL: frontendURL}
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

func (h *AuthHandler) OfficialLoginRedirect(c *gin.Context) {
	frontendURL := c.Query("redirect_uri")
	if frontendURL == "" {
		frontendURL = h.frontendURL
	}
	if _, err := url.Parse(frontendURL); err != nil {
		frontendURL = h.frontendURL
	}

	state := generateRandomState()
	c.SetCookie("oauth_state", state, 300, "/", "", false, true)
	c.SetCookie("oauth_redirect_uri", frontendURL, 300, "/", "", false, true)

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	callbackURL := fmt.Sprintf("%s://%s/v1/customer/login/official/callback", scheme, c.Request.Host)

	oauthURL := fmt.Sprintf(
		"https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_userinfo&state=%s#wechat_redirect",
		h.officialAppID,
		url.QueryEscape(callbackURL),
		state,
	)
	c.Redirect(http.StatusFound, oauthURL)
}

func (h *AuthHandler) OfficialLoginCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		redirectWithError(c, h.resolveFrontendURL(c), "missing code or state")
		return
	}

	cookieState, err := c.Cookie("oauth_state")
	if err != nil || cookieState != state {
		redirectWithError(c, h.resolveFrontendURL(c), "invalid state")
		return
	}
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	frontendURL := h.resolveFrontendURL(c)
	c.SetCookie("oauth_redirect_uri", "", -1, "/", "", false, true)

	resp, err := h.svc.LoginByOfficial(code)
	if err != nil {
		redirectWithError(c, frontendURL, "login failed")
		return
	}

	params := url.Values{}
	params.Set("token", resp.Token)
	params.Set("openId", resp.OpenID)
	params.Set("wechatNickname", resp.WechatNickname)
	params.Set("wechatAvatar", resp.WechatAvatar)
	params.Set("needProfile", fmt.Sprintf("%t", resp.NeedProfile))

	redirectURL := fmt.Sprintf("%s?%s", frontendURL, params.Encode())
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *AuthHandler) resolveFrontendURL(c *gin.Context) string {
	if u, err := c.Cookie("oauth_redirect_uri"); err == nil && u != "" {
		return u
	}
	return h.frontendURL
}

func redirectWithError(c *gin.Context, frontendURL, message string) {
	redirectURL := fmt.Sprintf("%s?error=%s", frontendURL, url.QueryEscape(message))
	c.Redirect(http.StatusFound, redirectURL)
}

func generateRandomState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
