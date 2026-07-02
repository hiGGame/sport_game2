package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"sport_game2/internal/config"
	"sport_game2/internal/crawler"
	"sport_game2/internal/crawler/sporttery"
	"sport_game2/internal/repo"
	"sport_game2/internal/service/predictor"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	log.Info("starting spider...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config", zap.Error(err))
	}

	db, err := repo.NewDB(cfg.Database.DSN(), cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns)
	if err != nil {
		log.Fatal("failed to connect db", zap.Error(err))
	}
	defer db.Close()

	if err := tryAcquireLock(db); err != nil {
		log.Fatal("another spider instance is already running, exiting", zap.Error(err))
	}

	client := sporttery.NewClient(cfg.Spider.UserAgent, cfg.Spider.Referer)
	c := sporttery.NewCrawler(client).WithRetry(cfg.Spider.RetryCount, cfg.Spider.RetryDelay)
	predictorSvc := predictor.NewRuleProvider()
	pipeline := crawler.NewPipeline(c, db, log, cfg.Spider.HealthFile, cfg.Spider.HealthMaxAge, predictorSvc)

	runOnce := len(os.Args) > 1 && os.Args[1] == "--once"
	if runOnce {
		log.Info("running single crawl cycle")
		pipeline.RunAllOdds()
		pipeline.RunAllResults()
		log.Info("single crawl done")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Info("received signal, shutting down...", zap.String("signal", sig.String()))
		cancel()
	}()

	log.Info("starting scheduled spider",
		zap.Duration("odds_interval", cfg.Spider.IntervalOdds),
		zap.Duration("result_interval", cfg.Spider.IntervalResult),
		zap.Int("retry_count", cfg.Spider.RetryCount),
		zap.Duration("retry_delay", cfg.Spider.RetryDelay),
	)

	pipeline.Schedule(ctx, cfg.Spider.IntervalOdds, cfg.Spider.IntervalResult)
}

func tryAcquireLock(db *repo.DB) error {
	var locked bool
	err := db.QueryRow("SELECT pg_try_advisory_lock(20260622)").Scan(&locked)
	if err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("advisory lock already held by another instance")
	}
	return nil
}


