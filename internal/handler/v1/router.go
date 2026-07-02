package v1

import (
	"github.com/gin-gonic/gin"

	"sport_game2/internal/middleware"
	"sport_game2/pkg/jwt"
)

type Router struct {
	authHandler        *AuthHandler
	matchHandler       *MatchHandler
	betHandler         *BetHandler
	resultHandler      *ResultHandler
	leaderboardHandler *LeaderboardHandler
	jwtManager         *jwt.Manager
}

func NewRouter(auth *AuthHandler, match *MatchHandler, bet *BetHandler, result *ResultHandler, leaderboard *LeaderboardHandler, jwtManager *jwt.Manager) *Router {
	return &Router{
		authHandler:        auth,
		matchHandler:       match,
		betHandler:         bet,
		resultHandler:      result,
		leaderboardHandler: leaderboard,
		jwtManager:         jwtManager,
	}
}

func (r *Router) Register(rg *gin.RouterGroup) {
	rg.Use(middleware.ErrorHandler())

	rg.GET("/lottery/match/bet-list", r.matchHandler.GetMatchBetList)
	rg.GET("/lottery/match/bet-info", r.matchHandler.GetMatchBetInfo)
	rg.GET("/lottery/match/draw/list", r.resultHandler.GetDrawHomeList)
	rg.GET("/matches/:matchId/predict", r.resultHandler.GetMatchPrediction)
	rg.GET("/lottery/avatars", r.leaderboardHandler.GetAvatars)
	rg.GET("/system/spider-status", r.matchHandler.GetSpiderStatus)

	rg.POST("/customer/login/wechat", r.authHandler.LoginByWechat)
	rg.POST("/customer/login/dev", r.authHandler.DevLogin)
	rg.GET("/customer/login/official/redirect", r.authHandler.OfficialLoginRedirect)
	rg.HEAD("/customer/login/official/redirect", r.authHandler.OfficialLoginRedirect)
	rg.GET("/customer/login/official/callback", r.authHandler.OfficialLoginCallback)
	rg.HEAD("/customer/login/official/callback", r.authHandler.OfficialLoginCallback)

	authed := rg.Group("/")
	authed.Use(middleware.JWTAuth(r.jwtManager))
	{
		authed.GET("/user/info", r.authHandler.GetUserInfo)
		authed.PUT("/user/info", r.authHandler.UpdateUserInfo)
		authed.GET("/lottery/leaderboard", r.leaderboardHandler.GetLeaderboard)
		authed.POST("/lottery/bet/create", r.betHandler.CreateBet)
		authed.GET("/lottery/bet/mine", r.betHandler.GetUserPredictions)
		authed.POST("/lottery/bet/share", r.betHandler.ShareSuccess)
		authed.GET("/lottery/pk", r.betHandler.DailyPK)
		authed.POST("/matches/:matchId/settle", r.resultHandler.SettleMatch)
	}
}
