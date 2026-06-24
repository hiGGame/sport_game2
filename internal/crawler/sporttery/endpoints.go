package sporttery

import "fmt"

const (
	baseAPI = "https://webapi.sporttery.cn/gateway/uniform"
)

// FootballCalculatorURL returns the odds API URL for a football pool code.
func FootballCalculatorURL(poolCode string) string {
	return fmt.Sprintf("%s/football/getMatchCalculatorV1.qry?channel=c&poolCode=%s", baseAPI, poolCode)
}

// FootballResultURL returns the draw result API URL for football.
func FootballResultURL(beginDate, endDate string, pageNo, pageSize int) string {
	return fmt.Sprintf("%s/football/getUniformMatchResultV1.qry?matchBeginDate=%s&matchEndDate=%s&leagueId=&pageSize=%d&pageNo=%d&isFix=0&matchPage=1&pcOrWap=1",
		baseAPI, beginDate, endDate, pageSize, pageNo)
}

// MatchHeadURL returns the match head info API URL (contains team logos).
func MatchHeadURL(matchID string) string {
	return fmt.Sprintf("%s/football/getMatchHeadV1.qry?source=web&sportteryMatchId=%s", baseAPI, matchID)
}

// TeamInfoURL returns the team detail API URL (contains logoUrl via gmTeamId).
func TeamInfoURL(teamID string) string {
	return fmt.Sprintf("%s/football/team/getTeamInfoV1.qry?gmTeamId=%s", baseAPI, teamID)
}
