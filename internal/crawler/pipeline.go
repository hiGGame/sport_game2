package crawler

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"go.uber.org/zap"

	"sport_game2/internal/adapter/apifox"
	"sport_game2/internal/crawler/sporttery"
	"sport_game2/internal/repo"
	"sport_game2/internal/service/predictor"
)

type matchWriter interface {
	UpsertMatch(m apifox.MatchBetInfo) error
	UpsertMatchResult(r apifox.DrawInfoReply) error
	LogSpiderJob(jobType string, status string, count int, errMsg string)
	GetExperts() ([]repo.Expert, error)
	UpsertExpertPrediction(ep *repo.ExpertPrediction) error
	UpsertAIPrediction(a *repo.AIPrediction) error
	UpsertTeam(teamID, name, alias, logoURL string) error
	UpsertLeague(leagueID, name, alias, color, level string) error
	BackfillMatchLogos() error
	SyncMatchLogos() error
	SettleAllForMatch(matchID string) (int, error)
}

type Pipeline struct {
	crawler    *sporttery.Crawler
	writer     matchWriter
	log        *zap.Logger
	healthFile string
	healthAge  time.Duration
	aiProvider predictor.Provider
}

func NewPipeline(c *sporttery.Crawler, writer matchWriter, log *zap.Logger, healthFile string, healthAge time.Duration, aiProvider predictor.Provider) *Pipeline {
	return &Pipeline{
		crawler:    c,
		writer:     writer,
		log:        log,
		healthFile: healthFile,
		healthAge:  healthAge,
		aiProvider: aiProvider,
	}
}

func (p *Pipeline) RunFootballOdds() error {
	start := time.Now()
	p.log.Info("crawling football HAD odds...")

	matches, err := p.crawler.CrawlFootballHAD()
	if err != nil {
		p.writer.LogSpiderJob("football_had", "failed", 0, err.Error())
		return fmt.Errorf("crawl football HAD: %w", err)
	}

	experts, _ := p.writer.GetExperts()

	successCount, failCount := 0, 0
	for _, m := range matches {
		if err := p.writer.UpsertMatch(m); err != nil {
			p.log.Warn("upsert match failed", zap.String("matchId", m.MatchInfo.MatchID), zap.Error(err))
			failCount++
			continue
		}
		successCount++

		p.cacheTeam(&m)
		p.cacheLeague(&m)

		p.generateAIPredictions(&m)
		p.generateExpertPredictions(&m, experts)
	}

	p.enrichTeamLogos(matches)
	p.writer.BackfillMatchLogos()
	p.writer.SyncMatchLogos()

	status := "success"
	if failCount > 0 && successCount == 0 {
		status = "failed"
	} else if failCount > 0 {
		status = "partial"
	}
	p.writer.LogSpiderJob("football_had", status, successCount, "")
	p.log.Info("football HAD done",
		zap.Int("success", successCount),
		zap.Int("failed", failCount),
		zap.Duration("elapsed", time.Since(start)))
	return nil
}

func (p *Pipeline) cacheTeam(m *apifox.MatchBetInfo) {
	if m.HomeTeamInfo.ID != "" {
		p.writer.UpsertTeam(m.HomeTeamInfo.ID, m.HomeTeamInfo.CnName, m.HomeTeamInfo.CnAlias, m.HomeTeamInfo.LogoURL)
	}
	if m.AwayTeamInfo.ID != "" {
		p.writer.UpsertTeam(m.AwayTeamInfo.ID, m.AwayTeamInfo.CnName, m.AwayTeamInfo.CnAlias, m.AwayTeamInfo.LogoURL)
	}
}

func (p *Pipeline) cacheLeague(m *apifox.MatchBetInfo) {
	if m.TournamentInfo.ID != "" {
		p.writer.UpsertLeague(m.TournamentInfo.ID, m.TournamentInfo.CnName, m.TournamentInfo.CnAlias,
			m.TournamentInfo.Color, m.TournamentInfo.Level)
	}
}

func (p *Pipeline) enrichTeamLogos(matches []apifox.MatchBetInfo) {
	seen := make(map[string]bool)
	for _, m := range matches {
		for _, t := range []struct{ id, name string }{
			{m.HomeTeamInfo.ID, m.HomeTeamInfo.CnName},
			{m.AwayTeamInfo.ID, m.AwayTeamInfo.CnName},
		} {
			if t.id == "" || seen[t.id] {
				continue
			}
			seen[t.id] = true
			logo := p.crawler.FetchTeamLogo(t.id)
			if logo != "" {
				p.writer.UpsertTeam(t.id, t.name, "", logo)
			}
		}
	}
}

func (p *Pipeline) generateAIPredictions(match *apifox.MatchBetInfo) {
	if p.aiProvider == nil {
		return
	}
	results, err := p.aiProvider.Predict(match)
	if err != nil {
		p.log.Warn("ai prediction failed", zap.String("matchId", match.MatchInfo.MatchID), zap.Error(err))
		return
	}
	for _, r := range results {
		if err := p.writer.UpsertAIPrediction(&repo.AIPrediction{
			MatchID: r.MatchID, LotteryType: r.LotteryType, SubType: r.SubType,
			BetCode: r.BetCode, Confidence: r.Confidence, Reasoning: r.Reasoning, ModelName: r.ModelName,
		}); err != nil {
			p.log.Warn("upsert ai prediction failed", zap.String("matchId", r.MatchID), zap.Error(err))
		}
	}
}

func (p *Pipeline) generateExpertPredictions(match *apifox.MatchBetInfo, experts []repo.Expert) {
	if len(experts) == 0 {
		return
	}
	for _, bi := range match.LotteryInfo.BetInfos {
		if len(bi.Options) == 0 {
			continue
		}
		for _, expert := range experts {
			idx := rand.Intn(len(bi.Options))
			opt := bi.Options[idx]
			conf := 0.5 + rand.Float64()*0.3
			if err := p.writer.UpsertExpertPrediction(&repo.ExpertPrediction{
				ExpertID: expert.ID, MatchID: match.MatchInfo.MatchID,
				LotteryType: match.LotteryInfo.LotteryType, SubType: bi.SubType,
				BetCode: opt.BetCode, Confidence: conf,
				Reasoning: fmt.Sprintf("%s推荐：基于数据分析，看好此选项。", expert.Name),
			}); err != nil {
				p.log.Warn("upsert expert prediction failed", zap.String("matchId", match.MatchInfo.MatchID), zap.Error(err))
			}
		}
	}
}

func (p *Pipeline) RunFootballResults() error {
	start := time.Now()
	p.log.Info("crawling football results...")

	beginDate := sporttery.DateOffset(-7)
	endDate := sporttery.Today()

	results, err := p.crawler.CrawlFootballResults(beginDate, endDate)
	if err != nil {
		p.writer.LogSpiderJob("football_results", "failed", 0, err.Error())
		return fmt.Errorf("crawl football results: %w", err)
	}

	successCount, failCount, settleCount := 0, 0, 0
	for _, r := range results {
		if err := p.writer.UpsertMatchResult(r); err != nil {
			p.log.Warn("upsert result failed", zap.String("matchId", r.MatchID), zap.Error(err))
			failCount++
			continue
		}
		successCount++

		if r.IsValid {
			if n, err := p.writer.SettleAllForMatch(r.MatchID); err != nil {
				p.log.Warn("settle failed", zap.String("matchId", r.MatchID), zap.Error(err))
			} else {
				settleCount += n
			}
		}
	}

	status := "success"
	if failCount > 0 && successCount == 0 {
		status = "failed"
	} else if failCount > 0 {
		status = "partial"
	}
	p.writer.LogSpiderJob("football_results", status, successCount, "")
	p.log.Info("football results done",
		zap.Int("success", successCount),
		zap.Int("failed", failCount),
		zap.Int("settled", settleCount),
		zap.Duration("elapsed", time.Since(start)))
	return nil
}

func (p *Pipeline) RunAllOdds() {
	defer p.recoverPanic("RunAllOdds")

	if err := p.RunFootballOdds(); err != nil {
		p.log.Error("football odds failed", zap.Error(err))
	}
}

func (p *Pipeline) RunAllResults() {
	defer p.recoverPanic("RunAllResults")

	if err := p.RunFootballResults(); err != nil {
		p.log.Error("football results failed", zap.Error(err))
	}
}

func (p *Pipeline) recoverPanic(job string) {
	if r := recover(); r != nil {
		p.log.Error("panic recovered",
			zap.String("job", job),
			zap.Any("panic", r),
			zap.Stack("stack"))
	}
}

// Schedule runs the spider on intervals with graceful shutdown and health check support.
func (p *Pipeline) Schedule(ctx context.Context, oddsInterval, resultInterval time.Duration) {
	oddsTicker := time.NewTicker(oddsInterval)
	resultTicker := time.NewTicker(resultInterval)
	defer oddsTicker.Stop()
	defer resultTicker.Stop()

	heartbeatInterval := min(oddsInterval, resultInterval) / 2
	if heartbeatInterval < 30*time.Second {
		heartbeatInterval = 30 * time.Second
	}
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	p.writeHealth()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeatTicker.C:
				p.writeHealth()
			}
		}
	}()

	p.RunAllOdds()
	p.RunAllResults()

	for {
		select {
		case <-ctx.Done():
			p.log.Info("spider scheduler shutting down gracefully")
			p.clearHealth()
			return
		case <-oddsTicker.C:
			p.RunAllOdds()
			p.writeHealth()
		case <-resultTicker.C:
			p.RunAllResults()
			p.writeHealth()
		}
	}
}

func (p *Pipeline) writeHealth() {
	if p.healthFile == "" {
		return
	}
	data := []byte(time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(p.healthFile, data, 0644); err != nil {
		p.log.Warn("failed to write health file", zap.Error(err))
	}
}

func (p *Pipeline) clearHealth() {
	if p.healthFile != "" {
		os.Remove(p.healthFile)
	}
}
