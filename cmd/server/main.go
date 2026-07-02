package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"math/rand/v2"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"sport_game2/internal/adapter/apifox"
	"sport_game2/internal/config"
	v1handler "sport_game2/internal/handler/v1"
	"sport_game2/internal/middleware"
	"sport_game2/internal/repo"
	"sport_game2/internal/service/auth"
	"sport_game2/internal/service/bet"
	"sport_game2/internal/service/leaderboard"
	"sport_game2/internal/service/match"
	"sport_game2/internal/service/predictor"
	"sport_game2/internal/service/result"
	"sport_game2/pkg/credits"
	"sport_game2/pkg/jwt"
	"sport_game2/pkg/wechat"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config", err)
	}

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := repo.NewDB(cfg.Database.DSN(), cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns)
	if err != nil {
		log.Fatal("failed to connect db", err)
	}
	defer db.Close()

	if err := runMigrations(db, logger); err != nil {
		log.Fatal("failed to run migrations", err)
	}
	logger.Info("migrations applied")

	wechatClient := wechat.NewClient(cfg.Wechat.MiniApp.AppID, cfg.Wechat.MiniApp.Secret)
	officialClient := wechat.NewOfficialClient(cfg.Wechat.Official.AppID, cfg.Wechat.Official.Secret)
	jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.ExpireHours)
	creditManager := credits.NewManager(db)

	authSvc := auth.NewService(db, wechatClient, officialClient, jwtManager, cfg.Bet.InitialCredits)
	matchSvc := match.NewService(db)
	betSvc := bet.NewService(db, db, creditManager, cfg.Bet.LockMinutesBefore)
	predictorSvc := predictor.NewRuleProvider()
	resultSvc := result.NewService(db, db, db, creditManager, predictorSvc)
	leaderboardSvc := leaderboard.NewService(db)

	authHandler := v1handler.NewAuthHandler(authSvc, cfg.Wechat.Official.AppID, cfg.Wechat.FrontendURL)
	matchHandler := v1handler.NewMatchHandler(matchSvc)
	betHandler := v1handler.NewBetHandler(betSvc)
	resultHandler := v1handler.NewResultHandler(resultSvc)
	leaderboardHandler := v1handler.NewLeaderboardHandler(leaderboardSvc)

	router := v1handler.NewRouter(authHandler, matchHandler, betHandler, resultHandler, leaderboardHandler, jwtManager)

	v1handler.InitAvatarCache("./web/avatars", "/static/avatars")

	g := gin.New()
	g.Use(gin.Recovery())
	g.Use(middleware.CORS())

	api := g.Group("/v1")
	router.Register(api)

	api.GET("/robot/status", func(c *gin.Context) {
		type bot struct {
			Name    string `json:"name"`
			Exists  bool   `json:"exists"`
			Total   int    `json:"todayTotal"`
			Pending int    `json:"todayPending"`
		}
		var bots []bot
		for _, o := range []struct{ openID, name string }{{"robot_expert", "老委鬼"}, {"robot_ai", "AI狗"}} {
			e, t, p, _ := db.GetRobotPredictStats(o.openID)
			bots = append(bots, bot{Name: o.name, Exists: e, Total: t, Pending: p})
		}
		c.JSON(200, gin.H{"robots": bots})
	})

	g.GET("/", serveWebPage)
	g.GET("/web", serveWebPage)
	g.GET("/web/index.html", serveWebPage)
	g.Static("/static", "./web")

	// 每日0点重置用户骨头数到初始值。
	// 骨头规则：
	//   - 每日0点重置用户骨头数为2个
	//   - 每次竞猜消耗2个骨头
	//   - 每次分享成功增加2个骨头
	//   - 竞猜中奖不返还骨头（骨头是消耗品，仅用于参与竞猜）
	go scheduleDailyReset(db, cfg.Bet.InitialCredits, logger)

	// 每日11点大拿/AI狗自动竞猜。受配置 bet.robot_auto_predict 开关控制。
	if cfg.Bet.RobotAutoPredict {
		go scheduleRobotPredict(db, logger)
	} else {
		logger.Info("robot auto predict disabled by config (bet.robot_auto_predict=false)")
	}

	// PK 就绪检查: 每隔10分钟检查上一个竞彩周期三方预测是否全部已开奖。
	go schedulePKReadyChecker(db, logger)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	logger.Info("server starting", zap.String("addr", addr))
	if err := g.Run(addr); err != nil {
		log.Fatal("server failed", err)
	}
}

func runMigrations(db *repo.DB, logger *zap.Logger) error {
	entries, err := os.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".sql" {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)

	for _, name := range files {
		path := filepath.Join("migrations", name)
		sql, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := db.Exec(string(sql)); err != nil {
			return fmt.Errorf("exec migration %s: %w", name, err)
		}
		logger.Info("migration applied", zap.String("file", name))
	}
	return nil
}

func serveWebPage(c *gin.Context) {
	data, err := os.ReadFile("./web/index.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "page not found")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

// scheduleDailyReset 每日0点将所有用户的骨头数重置到初始值。
//
// 骨头规则：
//   - 骨头是消耗品，仅用于参与竞猜
//   - 每日0点所有用户重置为初始骨头数（2个）
//   - 每次竞猜消耗2个骨头
//   - 每次分享成功增加2个骨头
//   - 竞猜中奖不返还骨头
func scheduleDailyReset(db *repo.DB, defaultCredits int, logger *zap.Logger) {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		wait := next.Sub(now)
		logger.Info("next daily credit reset scheduled",
			zap.Time("at", next),
			zap.Duration("in", wait),
			zap.Int("credits", defaultCredits))
		time.Sleep(wait)

		n, err := db.ResetAllCredits(defaultCredits)
		if err != nil {
			logger.Error("daily credits reset failed", zap.Error(err))
		} else {
			logger.Info("daily credits reset done", zap.Int64("affected", n), zap.Int("credits", defaultCredits))
		}
	}
}

func scheduleRobotPredict(db *repo.DB, logger *zap.Logger) {
	// 服务启动时如果已过今天 11:00，立即执行一次
	now := time.Now()
	today11 := time.Date(now.Year(), now.Month(), now.Day(), 11, 0, 0, 0, now.Location())
	if now.After(today11) {
		if config.Get().Bet.RobotAutoPredict {
			logger.Info("robot predict: missed today's 11:00 slot, running now")
			n, err := runRobotPredict(db, logger)
			if err != nil {
				logger.Error("robot predict failed", zap.Error(err))
			} else {
				logger.Info("robot predict done (catch-up)", zap.Int("created", n))
			}
		} else {
			logger.Info("robot predict: catch-up skipped, disabled by config")
		}
	}

	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 11, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		wait := next.Sub(now)
		logger.Info("next robot predict scheduled", zap.Time("at", next), zap.Duration("in", wait))
		time.Sleep(wait)

		if !config.Get().Bet.RobotAutoPredict {
			logger.Info("robot predict skipped, disabled by config (bet.robot_auto_predict=false)")
			continue
		}
		n, err := runRobotPredict(db, logger)
		if err != nil {
			logger.Error("robot predict failed", zap.Error(err))
		} else {
			logger.Info("robot predict done", zap.Int("created", n))
		}
	}
}

func runRobotPredict(db *repo.DB, logger *zap.Logger) (int, error) {
	expertUser, err := ensureRobotUser(db, "robot_expert")
	if err != nil {
		return 0, err
	}
	aiUser, err := ensureRobotUser(db, "robot_ai")
	if err != nil {
		return 0, err
	}

	matches, err := db.GetMatchBetList("227", "")
	if err != nil {
		return 0, fmt.Errorf("get matches: %w", err)
	}

	count := 0
	for _, m := range matches {
		if m.LotteryInfo.IsStopSell {
			continue
		}
		bi := pickBetInfo(m)
		if bi == nil || len(bi.Options) == 0 {
			continue
		}

		for _, uid := range []int64{expertUser.ID, aiUser.ID} {
			idx := rand.IntN(len(bi.Options))
			betCode := bi.Options[idx].BetCode
			id, err := db.RobotCreatePrediction(uid, m.MatchInfo.MatchID, m.LotteryInfo.LotteryType, m.LotteryInfo.MatchCode, bi.SubType, betCode)
			if err != nil {
				logger.Warn("robot create prediction failed",
					zap.Int64("userID", uid), zap.String("matchID", m.MatchInfo.MatchID), zap.Error(err))
				continue
			}
			if id > 0 {
				count++
			}
		}
	}
	return count, nil
}

func pickBetInfo(m apifox.MatchBetInfo) *apifox.BetInfo {
	for i := range m.LotteryInfo.BetInfos {
		if m.LotteryInfo.BetInfos[i].SubType == "6" {
			return &m.LotteryInfo.BetInfos[i]
		}
	}
	for i := range m.LotteryInfo.BetInfos {
		if m.LotteryInfo.BetInfos[i].SubType == "1" {
			return &m.LotteryInfo.BetInfos[i]
		}
	}
	return nil
}

func ensureRobotUser(db *repo.DB, openID string) (*repo.User, error) {
	u, err := db.GetUserByOpenID(openID)
	if err != nil {
		return nil, err
	}
	if u != nil {
		return u, nil
	}
	return db.CreateUser(openID, "", 1000)
}

// schedulePKReadyChecker 每隔10分钟检查上一个竞彩周期的三方预测是否已全部开奖。
func schedulePKReadyChecker(db *repo.DB, logger *zap.Logger) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}

	for {
		now := time.Now().In(loc)
		pkDate := now.AddDate(0, 0, -1)
		if now.Hour() < 11 {
			pkDate = now.AddDate(0, 0, -2)
		}
		fromTime := pkDate.Format("2006-01-02") + " 11:00:00"
		toTime := pkDate.AddDate(0, 0, 1).Format("2006-01-02") + " 11:00:00"

		preds, err := db.GetAllPredictionsInRange(fromTime, toTime)
		if err != nil {
			logger.Error("pk ready check failed", zap.Error(err))
		} else {
			pending := 0
			for _, p := range preds {
				if p.Status == "pending" {
					pending++
				}
			}
			if pending > 0 {
				logger.Info("PK waiting for settlement",
					zap.String("cycle", fromTime+" ~ "+toTime),
					zap.Int("pending", pending),
					zap.Int("total", len(preds)))
			} else if len(preds) > 0 {
				logger.Info("PK cycle ready",
					zap.String("cycle", fromTime+" ~ "+toTime),
					zap.Int("total_preds", len(preds)))
			}
		}

		next := time.Now().Add(10 * time.Minute)
		logger.Info("next pk ready check", zap.Time("at", next))
		time.Sleep(10 * time.Minute)
	}
}
