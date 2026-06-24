package predictor

import (
	"fmt"
	"math"

	"sport_game2/internal/adapter/apifox"
	"sport_game2/internal/model"
	pkgSporttery "sport_game2/pkg/sporttery"
)

type RuleProvider struct{}

func NewRuleProvider() *RuleProvider {
	return &RuleProvider{}
}

func (r *RuleProvider) Name() string {
	return "rule"
}

func (r *RuleProvider) Predict(match *apifox.MatchBetInfo) ([]model.PredictionResult, error) {
	var results []model.PredictionResult

	for _, bi := range match.LotteryInfo.BetInfos {
		pred := predictSubType(match.LotteryInfo.LotteryType, bi)
		if pred.BetCode != "" {
			pred.MatchID = match.MatchInfo.MatchID
			pred.LotteryType = match.LotteryInfo.LotteryType
			pred.ModelName = r.Name()
			results = append(results, pred)
		}
	}

	return results, nil
}

func predictSubType(lotteryType string, bi apifox.BetInfo) model.PredictionResult {
	if len(bi.Options) == 0 {
		return model.PredictionResult{SubType: bi.SubType}
	}

	probs := calcProbabilities(bi.Options)

	bestIdx := 0
	for i, p := range probs {
		if p > probs[bestIdx] {
			bestIdx = i
		}
	}

	reasoning := fmt.Sprintf("基于赔率反推概率，%s概率最高(%.1f%%)。",
		pkgSporttery.SubTypeLabel(lotteryType, bi.SubType), probs[bestIdx]*100)
	if len(bi.Options) > 1 {
		secondIdx := -1
		for i, p := range probs {
			if i == bestIdx {
				continue
			}
			if secondIdx == -1 || p > probs[secondIdx] {
				secondIdx = i
			}
		}
		reasoning += fmt.Sprintf(" 次选概率%.1f%%。", probs[secondIdx]*100)
	}

	return model.PredictionResult{
		SubType:    bi.SubType,
		BetCode:    bi.Options[bestIdx].BetCode,
		Confidence: probs[bestIdx],
		Reasoning:  reasoning,
	}
}

func calcProbabilities(options []apifox.BetOption) []float64 {
	var probs []float64
	var totalWeight float64

	for _, opt := range options {
		w := 1.0 / opt.Odds
		probs = append(probs, w)
		totalWeight += w
	}

	for i := range probs {
		probs[i] = probs[i] / totalWeight
	}

	return probs
}

var _ = math.Round
