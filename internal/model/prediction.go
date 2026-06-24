package model

type PredictionResult struct {
	MatchID     string  `json:"matchId"`
	LotteryType string  `json:"lotteryType"`
	SubType     string  `json:"subType"`
	BetCode     string  `json:"betCode"`
	Confidence  float64 `json:"confidence"`
	Reasoning   string  `json:"reasoning"`
	ModelName   string  `json:"modelName"`
}

type ExpertPredictionView struct {
	ExpertName string  `json:"expertName"`
	AvatarURL  string  `json:"avatarUrl"`
	Title      string  `json:"title"`
	SubType    string  `json:"subType"`
	BetCode    string  `json:"betCode"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

type UserPredictionView struct {
	UserID    int64  `json:"userId"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
	SubType   string `json:"subType"`
	BetCode   string `json:"betCode"`
	Points    int    `json:"points"`
}
