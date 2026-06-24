package apifox

import "encoding/json"

type GetMatchBetListReply struct {
	List   []MatchBetInfo   `json:"list"`
	Groups []MatchBetGroup  `json:"groups"`
}

type GetMatchBetInfoReply struct {
	Info MatchBetInfo `json:"info"`
}

type MatchBetGroup struct {
	Issue string         `json:"issue"`
	List  []MatchBetInfo `json:"list"`
}

type MatchBetInfo struct {
	MatchInfo      MatchInfo      `json:"matchInfo"`
	LotteryInfo    LotteryInfo    `json:"lotteryInfo"`
	TournamentInfo TournamentInfo `json:"tournamentInfo"`
	HomeTeamInfo   TeamInfo       `json:"homeTeamInfo"`
	AwayTeamInfo   TeamInfo       `json:"awayTeamInfo"`
}

type MatchInfo struct {
	MatchID       string     `json:"matchId"`
	SportID       string     `json:"sportId"`
	MatchTimeStr  string     `json:"matchTimeStr"`
	Status        string     `json:"status"`
	StatusCode    string     `json:"statusCode"`
	StatusName    string     `json:"statusName"`
	HomeTeamScore MatchScore `json:"homeTeamScore"`
	AwayTeamScore MatchScore `json:"awayTeamScore"`
}

type LotteryInfo struct {
	LotteryType  string    `json:"lotteryType"`
	MatchCode    string    `json:"matchCode"`
	Issue        string    `json:"issue"`
	Round        string    `json:"round"`
	BetEndTimeStr string  `json:"betEndTimeStr"`
	Reverse      bool      `json:"reverse"`
	BetInfos     []BetInfo `json:"betInfos"`
	SpValue      float64   `json:"spValue"`
	IsStopSell   bool      `json:"isStopSell"`
}

type TournamentInfo struct {
	CnName  string `json:"cnName"`
	CnAlias string `json:"cnAlias"`
	Color   string `json:"color"`
	Level   string `json:"level"`
	LogoURL string `json:"logoUrl"`
	ID      string `json:"id"`
}

type TeamInfo struct {
	CnName          string `json:"cnName"`
	CnAlias         string `json:"cnAlias"`
	LogoURL         string `json:"logoUrl"`
	TournamentRank  string `json:"tournamentRank"`
	FifaClubRank    string `json:"fifaClubRank"`
	FifaCountryRank string `json:"fifaCountryRank"`
	FibaCountryRank string `json:"fibaCountryRank"`
	ID              string `json:"id"`
	Rank            string `json:"rank"`
}

type MatchScore struct {
	Score            string `json:"score"`
	NormalTimeScore  string `json:"normalTimeScore"`
	HalfTimeScore    string `json:"halfTimeScore"`
}

type BetInfo struct {
	SubType     string      `json:"subType"`
	SupportType string      `json:"supportType"`
	Handicap    float64     `json:"handicap"`
	Options     []BetOption `json:"options"`
}

type BetOption struct {
	BetCode string  `json:"betCode"`
	Odds    float64 `json:"odds"`
	Change  string  `json:"change"`
}

type GetLotteryDrawHomeListReply struct {
	List []LotteryDrawHomeInfo `json:"list"`
}

type LotteryDrawHomeInfo struct {
	LotteryType  string        `json:"lotteryType"`
	OptionCount  string        `json:"optionCount"`
	LastDrawInfo DrawInfoReply `json:"lastDrawInfo"`
}

type DrawInfoReply struct {
	MatchID        string           `json:"matchId"`
	MatchCode      string           `json:"matchCode"`
	WeekDay        string           `json:"weekDay"`
	Round          string           `json:"round"`
	IsSingle       bool             `json:"isSingle"`
	MatchTimeStr   string           `json:"matchTimeStr"`
	TournamentInfo TournamentInfo   `json:"tournamentInfo"`
	HomeTeamInfo   TeamInfo         `json:"homeTeamInfo"`
	AwayTeamInfo   TeamInfo         `json:"awayTeamInfo"`
	IsValid        bool             `json:"isValid"`
	HomeTeamScore  MatchScore       `json:"homeTeamScore"`
	AwayTeamScore  MatchScore       `json:"awayTeamScore"`
	GameDrawList   []GameDrawInfo   `json:"gameDrawList"`
}

type GameDrawInfo struct {
	SubType  string  `json:"subType"`
	Handicap float64 `json:"handicap"`
	BetCode  string  `json:"betCode"`
	Odds     float64 `json:"odds"`
}

type GetHotMatchListReply struct {
	List  []HotMatchInfo `json:"list"`
	IsNew bool           `json:"isNew"`
}

type HotMatchInfo struct {
	MatchID             string      `json:"matchId"`
	MatchCnName         string      `json:"matchCnName"`
	MatchCnAlias        string      `json:"matchCnAlias"`
	AwayTeamName        string      `json:"awayTeamName"`
	AwayTeamAliasName   string      `json:"awayTeamAliasName"`
	AwayTeamLogoURL     string      `json:"awayTeamLogoUrl"`
	HomeTeamName        string      `json:"homeTeamName"`
	HomeTeamAliasName   string      `json:"homeTeamAliasName"`
	HomeTeamLogoURL     string      `json:"homeTeamLogoUrl"`
	LotteryRound        string      `json:"lotteryRound"`
	LotteryBetEndTimeStr string     `json:"lotteryBetEndTimeStr"`
	SubType             string      `json:"subType"`
	MatchCode           string      `json:"matchCode"`
	Issue               string      `json:"issue"`
	LotteryBetOptions   []BetOption `json:"lotteryBetOptions"`
}

type LoginByWechatRequest struct {
	AppID          string `json:"appid"`
	Code           string `json:"code"`
	State          string `json:"state"`
	MerchantID     string `json:"merchantId"`
	ReferCustomerID string `json:"referCustomerId"`
	ReferShopUserID string `json:"referShopUserId"`
}

type LoginByWechatReply struct {
	Token          string      `json:"token"`
	Shops          []LoginShop `json:"shops"`
	WechatNickname string      `json:"wechatNickname"`
	WechatAvatar   string      `json:"wechatAvatar"`
	MerchantID     string      `json:"merchantId"`
	NeedBind       bool        `json:"needBind"`
	OpenID         string      `json:"openId"`
}

type LoginShop struct {
	MerchantID string `json:"merchantId"`
	ShopName   string `json:"shopName"`
}

func (m *MatchBetInfo) ToJSON() string {
	b, _ := json.Marshal(m)
	return string(b)
}
