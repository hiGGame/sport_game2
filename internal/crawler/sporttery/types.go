package sporttery

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// --- Odds Calculator ---

type CalculatorResponse struct {
	ErrorCode string  `json:"errorCode"`
	Value    CalcValue `json:"value"`
}

type CalcValue struct {
	MatchInfoList []MatchInfoDate `json:"matchInfoList"`
}

type MatchInfoDate struct {
	BusinessDate string        `json:"businessDate"`
	SubMatchList []RawMatch    `json:"subMatchList"`
}

type RawMatch struct {
	MatchID         json.Number `json:"matchId"`
	MatchDate       string      `json:"matchDate"`
	MatchTime       string      `json:"matchTime"`
	MatchNumStr     string      `json:"matchNumStr"`
	MatchNum        json.Number `json:"matchNum"`
	MatchWeek       string      `json:"matchWeek"`
	MatchStatus     string      `json:"matchStatus"`
	LeagueID        json.Number `json:"leagueId"`
	LeagueAbbName   string      `json:"leagueAbbName"`
	LeagueAllName   string      `json:"leagueAllName"`
	LeagueCode      string      `json:"leagueCode"`
	HomeTeamAbbName string      `json:"homeTeamAbbName"`
	HomeTeamAllName string      `json:"homeTeamAllName"`
	HomeTeamCode    string      `json:"homeTeamCode"`
	HomeTeamID      json.Number `json:"homeTeamId"`
	HomeRank        string      `json:"homeRank"`
	AwayTeamAbbName string      `json:"awayTeamAbbName"`
	AwayTeamAllName string      `json:"awayTeamAllName"`
	AwayTeamCode    string      `json:"awayTeamCode"`
	AwayTeamID      json.Number `json:"awayTeamId"`
	AwayRank        string      `json:"awayRank"`
	IsHot           int         `json:"isHot"`
	IsHide          int         `json:"isHide"`

	Had  PoolOdds `json:"had"`
	Hhad PoolOdds `json:"hhad"`
	Crs  PoolOdds `json:"crs"`
	Ttg  PoolOdds `json:"ttg"`
	Hafu PoolOdds `json:"hafu"`

	Mnl  PoolOdds `json:"mnl"`
	Hdc  PoolOdds `json:"hdc"`
	Wsf  PoolOdds `json:"wsf"`
	Hhu  PoolOdds `json:"hhu"`

	PoolList []PoolInfo `json:"poolList"`
}

type PoolOdds struct {
	H string `json:"h"`
	D string `json:"d"`
	A string `json:"a"`

	Hf string `json:"hf"`
	Df string `json:"df"`
	Af string `json:"af"`

	GoalLine      string `json:"goalLine"`
	GoalLineValue string `json:"goalLineValue"`
	UpdateDate    string `json:"updateDate"`
	UpdateTime    string `json:"updateTime"`
}

type PoolInfo struct {
	PoolCode      string `json:"poolCode"`
	PoolStatus    string `json:"poolStatus"`
	Single        int    `json:"single"`
	AllUp         int    `json:"allUp"`
	PoolCloseDate string `json:"poolCloseDate"`
	PoolCloseTime string `json:"poolCloseTime"`
}

// --- Match Result (getUniformMatchResultV1) ---

type MatchResultResponse struct {
	ErrorCode string      `json:"errorCode"`
	Value    ResultValue  `json:"value"`
}

type ResultValue struct {
	MatchResult []RawResult `json:"matchResult"`
}

type RawResult struct {
	MatchID          json.Number `json:"matchId"`
	MatchNumStr      string      `json:"matchNumStr"`
	MatchDate        string      `json:"matchDate"`
	HomeTeam         string      `json:"homeTeam"`
	AllHomeTeam      string      `json:"allHomeTeam"`
	HomeTeamID       json.Number `json:"homeTeamId"`
	AwayTeam         string      `json:"awayTeam"`
	AllAwayTeam      string      `json:"allAwayTeam"`
	AwayTeamID       json.Number `json:"awayTeamId"`
	LeagueID         json.Number `json:"leagueId"`
	LeagueName       string      `json:"leagueName"`
	LeagueNameAbbr   string      `json:"leagueNameAbbr"`
	LeagueBackColor  string      `json:"leagueBackColor"`
	MatchResultStatus string     `json:"matchResultStatus"`
	PoolStatus       string      `json:"poolStatus"`
	WinFlag          string      `json:"winFlag"`
	GoalLine         string      `json:"goalLine"`
	SectionsNo1      string      `json:"sectionsNo1"`
	SectionsNo999    string      `json:"sectionsNo999"`
	H                string      `json:"h"`
	D                string      `json:"d"`
	A                string      `json:"a"`
	BettingSingle    int         `json:"bettingSingle"`
}

// --- Match Head (getMatchHeadV1) — contains team logos ---

type MatchHeadResponse struct {
	ErrorCode string    `json:"errorCode"`
	Value    MatchHead `json:"value"`
}

type MatchHead struct {
	HomeTeamLogoPath  string `json:"homeTeamLogoPath"`
	AwayTeamLogoPath  string `json:"awayTeamLogoPath"`
	HomeTeamShortName string `json:"homeTeamShortName"`
	AwayTeamShortName string `json:"awayTeamShortName"`
}

type TeamInfoResponse struct {
	ErrorCode string   `json:"errorCode"`
	Value    TeamInfo `json:"value"`
}

type TeamInfo struct {
	LogoUrl string `json:"logoUrl"`
}

// --- Parsing ---

func ParseCalculator(data []byte) (*CalculatorResponse, error) {
	var resp CalculatorResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse calculator: %w", err)
	}
	if resp.ErrorCode != "0" {
		return nil, fmt.Errorf("sporttery error code: %s", resp.ErrorCode)
	}
	return &resp, nil
}

func ParseMatchResult(data []byte) (*MatchResultResponse, error) {
	var resp MatchResultResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse match result: %w", err)
	}
	if resp.ErrorCode != "0" {
		return nil, fmt.Errorf("sporttery error code: %s", resp.ErrorCode)
	}
	return &resp, nil
}

func ParseMatchHead(data []byte) (*MatchHeadResponse, error) {
	var resp MatchHeadResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse match head: %w", err)
	}
	if resp.ErrorCode != "0" {
		return nil, fmt.Errorf("sporttery error code: %s", resp.ErrorCode)
	}
	return &resp, nil
}

func ParseTeamInfo(data []byte) (*TeamInfoResponse, error) {
	var resp TeamInfoResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse team info: %w", err)
	}
	if resp.ErrorCode != "0" {
		return nil, fmt.Errorf("sporttery error code: %s", resp.ErrorCode)
	}
	return &resp, nil
}

func Today() string {
	return time.Now().Format("2006-01-02")
}

func DateOffset(days int) string {
	return time.Now().AddDate(0, 0, days).Format("2006-01-02")
}

func parseScore(raw string) (string, string) {
	if raw == "" {
		return "", ""
	}
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return raw, ""
}

// mapBetCode converts sporttery H/D/A to 3/1/0.
func mapBetCode(code string) string {
	switch code {
	case "H":
		return "3"
	case "D":
		return "1"
	case "A":
		return "0"
	}
	return code
}

// normalizeLogoURL ensures the logo URL has https: prefix.
func normalizeLogoURL(u string) string {
	if u != "" && len(u) >= 2 && u[:2] == "//" {
		return "https:" + u
	}
	return u
}
