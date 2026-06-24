package sporttery

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"sport_game2/internal/adapter/apifox"
	pkgSporttery "sport_game2/pkg/sporttery"
)

type Crawler struct {
	client     *Client
	retryCount int
	retryDelay time.Duration
}

func NewCrawler(client *Client) *Crawler {
	return &Crawler{
		client:     client,
		retryCount: 3,
		retryDelay: 2 * time.Second,
	}
}

func (c *Crawler) WithRetry(count int, delay time.Duration) *Crawler {
	c.retryCount = count
	c.retryDelay = delay
	return c
}

func (c *Crawler) fetchWithRetry(urlStr string) ([]byte, error) {
	var lastErr error
	for i := 0; i <= c.retryCount; i++ {
		data, err := c.client.Get(urlStr)
		if err == nil {
			return data, nil
		}

		var wafErr *ErrWAFBlocked
		if errors.As(err, &wafErr) {
			return nil, fmt.Errorf("waf blocked, skipping retries: %w", err)
		}

		lastErr = err
		if i < c.retryCount {
			backoff := c.retryDelay * time.Duration(1<<uint(i))
			time.Sleep(backoff)
		}
	}
	return nil, lastErr
}

// CrawlFootballHAD crawls HAD first (works from cloud IPs), then HHAD.
// If HHAD fails due to WAF, HAD data is still updated.
func (c *Crawler) CrawlFootballHAD() ([]apifox.MatchBetInfo, error) {
	// HAD first — more likely to succeed from cloud servers
	hadMatches, err := c.crawlCalculator(pkgSporttery.PoolHAD)
	if err != nil {
		return nil, err
	}

	// Delay before HHAD to avoid WAF rate-limiting
	time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)

	// HHAD may fail on cloud IPs — not fatal, HAD data is still valid
	hhadMatches, err := c.crawlCalculator(pkgSporttery.PoolHHAD)
	if err != nil {
		// HHAD failed, return HAD-only data with logos
		for i := range hadMatches {
			logoHome, logoAway, logoErr := c.fetchTeamLogos(hadMatches[i].MatchInfo.MatchID)
			if logoErr == nil {
				hadMatches[i].HomeTeamInfo.LogoURL = logoHome
				hadMatches[i].AwayTeamInfo.LogoURL = logoAway
			}
			if i < len(hadMatches)-1 {
				time.Sleep(time.Duration(300+rand.Intn(300)) * time.Millisecond)
			}
		}
		return hadMatches, nil
	}

	// Both succeeded — merge
	hadMap := make(map[string]*apifox.MatchBetInfo, len(hadMatches))
	for i := range hadMatches {
		m := &hadMatches[i]
		hadMap[m.MatchInfo.MatchID] = m
	}

	var result []apifox.MatchBetInfo
	for i := range hhadMatches {
		m := &hhadMatches[i]
		if h, ok := hadMap[m.MatchInfo.MatchID]; ok {
			m.LotteryInfo.BetInfos = append(h.LotteryInfo.BetInfos, m.LotteryInfo.BetInfos...)
			if m.LotteryInfo.BetEndTimeStr == "" {
				m.LotteryInfo.BetEndTimeStr = h.LotteryInfo.BetEndTimeStr
			}
		}
		logoHome, logoAway, err := c.fetchTeamLogos(m.MatchInfo.MatchID)
		if err == nil {
			m.HomeTeamInfo.LogoURL = logoHome
			m.AwayTeamInfo.LogoURL = logoAway
		}
		result = append(result, *m)
		if i < len(hhadMatches)-1 {
			time.Sleep(time.Duration(300+rand.Intn(300)) * time.Millisecond)
		}
	}

	return result, nil
}

func (c *Crawler) crawlCalculator(poolCode string) ([]apifox.MatchBetInfo, error) {
	urlStr := FootballCalculatorURL(poolCode)

	data, err := c.fetchWithRetry(urlStr)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", poolCode, err)
	}

	resp, err := ParseCalculator(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", poolCode, err)
	}

	subType := pkgSporttery.PoolCodeToSubType(pkgSporttery.LotteryTypeFoot, poolCode)

	var result []apifox.MatchBetInfo
	for _, dateGroup := range resp.Value.MatchInfoList {
		for i := range dateGroup.SubMatchList {
			rm := &dateGroup.SubMatchList[i]
			mb := rawToMatchBetInfo(subType, poolCode, rm)
			if mb.LotteryInfo.MatchCode != "" {
				result = append(result, mb)
			}
		}
	}

	return result, nil
}

// fetchTeamLogos calls getMatchHeadV1 to retrieve team logo URLs for a match.
func (c *Crawler) fetchTeamLogos(matchID string) (string, string, error) {
	if matchID == "" {
		return "", "", fmt.Errorf("empty matchId")
	}

	urlStr := MatchHeadURL(matchID)
	data, err := c.fetchWithRetry(urlStr)
	if err != nil {
		return "", "", fmt.Errorf("fetch match head: %w", err)
	}

	resp, err := ParseMatchHead(data)
	if err != nil {
		return "", "", fmt.Errorf("parse match head: %w", err)
	}

	home := normalizeLogoURL(resp.Value.HomeTeamLogoPath)
	away := normalizeLogoURL(resp.Value.AwayTeamLogoPath)
	return home, away, nil
}

// FetchTeamLogo gets team logo URL from the team detail API using gmTeamId.
// Returns the logo URL, including the default placeholder (teamdef_zq.png) if no custom logo exists.
func (c *Crawler) FetchTeamLogo(teamID string) string {
	urlStr := TeamInfoURL(teamID)
	data, err := c.fetchWithRetry(urlStr)
	if err != nil {
		return ""
	}
	resp, err := ParseTeamInfo(data)
	if err != nil || resp.Value.LogoUrl == "" {
		return ""
	}
	return normalizeLogoURL(resp.Value.LogoUrl)
}

// CrawlFootballResults crawls football draw results for a date range.
func (c *Crawler) CrawlFootballResults(beginDate, endDate string) ([]apifox.DrawInfoReply, error) {
	pageNo := 1
	pageSize := 100
	var allResults []apifox.DrawInfoReply
	var lastErr error

	for {
		urlStr := FootballResultURL(beginDate, endDate, pageNo, pageSize)

		data, err := c.fetchWithRetry(urlStr)
		if err != nil {
			lastErr = fmt.Errorf("fetch results page %d: %w", pageNo, err)
			break
		}

		resp, err := ParseMatchResult(data)
		if err != nil {
			lastErr = fmt.Errorf("parse results page %d: %w", pageNo, err)
			break
		}

		if len(resp.Value.MatchResult) == 0 {
			break
		}

		for i := range resp.Value.MatchResult {
			r := rawToDrawInfo(&resp.Value.MatchResult[i])
			allResults = append(allResults, r)
		}

		if len(resp.Value.MatchResult) < pageSize {
			break
		}
		pageNo++
		time.Sleep(time.Duration(500+rand.Intn(500)) * time.Millisecond)
	}

	if lastErr != nil && len(allResults) == 0 {
		return nil, lastErr
	}
	return allResults, nil
}

func rawToMatchBetInfo(subType, poolCode string, rm *RawMatch) apifox.MatchBetInfo {
	pool := getPoolOdds(rm, poolCode)
	var options []apifox.BetOption
	if pool != nil {
		if pool.H != "" {
			options = append(options, apifox.BetOption{BetCode: "3", Odds: parseFloat64(pool.H), Change: pool.Hf})
		}
		if pool.D != "" {
			options = append(options, apifox.BetOption{BetCode: "1", Odds: parseFloat64(pool.D), Change: pool.Df})
		}
		if pool.A != "" {
			options = append(options, apifox.BetOption{BetCode: "0", Odds: parseFloat64(pool.A), Change: pool.Af})
		}
	}

	supportType := "2"
	if rm.MatchStatus != "Selling" {
		supportType = "3"
	}

	betEndTimeStr := ""
	for _, p := range rm.PoolList {
		if p.PoolCode == upperPoolCode(poolCode) {
			betEndTimeStr = p.PoolCloseDate + " " + p.PoolCloseTime
			break
		}
	}

	handicap := 0.0
	if pool != nil {
		handicap = parseFloat64(pool.GoalLine)
	}

	matchID := rm.MatchID.String()
	if matchID == "" {
		matchID = rm.MatchNum.String()
	}

	return apifox.MatchBetInfo{
		MatchInfo: apifox.MatchInfo{
			MatchID:     matchID,
			SportID:     pkgSporttery.SportFootball,
			MatchTimeStr: rm.MatchDate + " " + rm.MatchTime,
			Status:      rm.MatchStatus,
			StatusCode:  rm.MatchStatus,
			StatusName:  rm.MatchStatus,
		},
		LotteryInfo: apifox.LotteryInfo{
			LotteryType:   pkgSporttery.LotteryTypeFoot,
			MatchCode:     rm.MatchNumStr,
			Issue:         rm.MatchDate,
			Round:         rm.MatchWeek,
			BetEndTimeStr: betEndTimeStr,
			BetInfos: []apifox.BetInfo{
				{
					SubType:     subType,
					SupportType: supportType,
					Handicap:    handicap,
					Options:     options,
				},
			},
			IsStopSell: rm.MatchStatus != "Selling",
		},
		TournamentInfo: apifox.TournamentInfo{
			CnName:  rm.LeagueAbbName,
			CnAlias: rm.LeagueAllName,
			ID:      rm.LeagueID.String(),
		},
		HomeTeamInfo: apifox.TeamInfo{
			CnName:  rm.HomeTeamAbbName,
			CnAlias: rm.HomeTeamAllName,
			ID:      rm.HomeTeamID.String(),
			Rank:    rm.HomeRank,
		},
		AwayTeamInfo: apifox.TeamInfo{
			CnName:  rm.AwayTeamAbbName,
			CnAlias: rm.AwayTeamAllName,
			ID:      rm.AwayTeamID.String(),
			Rank:    rm.AwayRank,
		},
	}
}

func rawToDrawInfo(rr *RawResult) apifox.DrawInfoReply {
	var gameDrawList []apifox.GameDrawInfo

	hadResult := mapBetCode(rr.WinFlag)
	hadOdds := 0.0
	if rr.H != "" {
		hadOdds = parseFloat64(rr.H)
	}
	if hadResult != "" {
		gameDrawList = append(gameDrawList, apifox.GameDrawInfo{
			SubType: pkgSporttery.FootSubHAD,
			BetCode: hadResult,
			Odds:    hadOdds,
		})
	}

	return apifox.DrawInfoReply{
		MatchID:      rr.MatchID.String(),
		MatchCode:    rr.MatchNumStr,
		MatchTimeStr: rr.MatchDate,
		WeekDay:      "",
		IsSingle:     false,
		TournamentInfo: apifox.TournamentInfo{
			CnName:  rr.LeagueNameAbbr,
			CnAlias: rr.LeagueName,
		},
		HomeTeamInfo: apifox.TeamInfo{
			CnName:  rr.HomeTeam,
			CnAlias: rr.AllHomeTeam,
			ID:      rr.HomeTeamID.String(),
		},
		AwayTeamInfo: apifox.TeamInfo{
			CnName:  rr.AwayTeam,
			CnAlias: rr.AllAwayTeam,
			ID:      rr.AwayTeamID.String(),
		},
		IsValid: rr.MatchResultStatus == "2",
		HomeTeamScore: apifox.MatchScore{
			Score:           rr.SectionsNo999,
			NormalTimeScore: rr.SectionsNo999,
			HalfTimeScore:   rr.SectionsNo1,
		},
		GameDrawList: gameDrawList,
	}
}

func getPoolOdds(rm *RawMatch, poolCode string) *PoolOdds {
	switch poolCode {
	case pkgSporttery.PoolHAD:
		return &rm.Had
	case pkgSporttery.PoolHHAD:
		return &rm.Hhad
	case pkgSporttery.PoolCRS:
		return &rm.Crs
	case pkgSporttery.PoolTTG:
		return &rm.Ttg
	case pkgSporttery.PoolHAFU:
		return &rm.Hafu
	}
	return nil
}

func upperPoolCode(pc string) string {
	switch pc {
	case pkgSporttery.PoolHAD:
		return "HAD"
	case pkgSporttery.PoolHHAD:
		return "HHAD"
	case pkgSporttery.PoolCRS:
		return "CRS"
	case pkgSporttery.PoolTTG:
		return "TTG"
	case pkgSporttery.PoolHAFU:
		return "HAFU"
	}
	return ""
}

func parseFloat64(s string) float64 {
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

var _ json.Number
