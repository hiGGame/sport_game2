package predictor

import (
	"encoding/json"
	"fmt"
	"strings"

	"sport_game2/internal/model"
)

const SystemPrompt = `你是一个专业的竞彩分析师AI助手（AI狗）。
你需要根据赛事信息和赔率数据，对以下玩法给出预测：
- 胜平负（HAD）：预测主队胜(H)、平(D)、客队胜(A)
- 让球胜平负（HHAD）：考虑让球数后预测
- 比分（CRS）：预测具体比分
- 总进球数（TTG）：预测总进球数（0-7+）
- 半全场（HAFU）：预测半场和全场结果组合

请基于以下信息分析：
1. 赔率变化趋势
2. 两队历史排名和实力
3. 主客场因素

请以JSON格式返回预测结果，格式如下：
{
  "predictions": [
    {
      "subType": "6",
      "betCode": "H",
      "confidence": 0.65,
      "reasoning": "分析理由..."
    }
  ]
}`

type LLMRequest struct {
	MatchInfo    string `json:"matchInfo"`
	OddsInfo     string `json:"oddsInfo"`
	TeamInfo     string `json:"teamInfo"`
}

type LLMResponse struct {
	Predictions []model.PredictionResult `json:"predictions"`
}

func BuildPrompt(match *MatchContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("赛事: %s vs %s\n", match.HomeTeam, match.AwayTeam))
	sb.WriteString(fmt.Sprintf("联赛: %s\n", match.LeagueName))
	sb.WriteString(fmt.Sprintf("比赛时间: %s\n\n", match.MatchTime))
	sb.WriteString("赔率信息:\n")
	for _, o := range match.Odds {
		sb.WriteString(fmt.Sprintf("  玩法%s: ", o.SubType))
		for _, opt := range o.Options {
			sb.WriteString(fmt.Sprintf("%s=%.2f ", opt.BetCode, opt.Odds))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n请给出你的预测分析。")
	return sb.String()
}

type MatchContext struct {
	HomeTeam   string
	AwayTeam   string
	LeagueName string
	MatchTime  string
	Odds       []OddsContext
}

type OddsContext struct {
	SubType string
	Options []OptionContext
}

type OptionContext struct {
	BetCode string
	Odds    float64
}

func ParseLLMResponse(content string) (*LLMResponse, error) {
	content = strings.TrimSpace(content)
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no valid JSON found in LLM response")
	}
	jsonStr := content[start : end+1]

	var resp LLMResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}
	return &resp, nil
}
