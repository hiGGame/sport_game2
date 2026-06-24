package predictor

import (
	"sport_game2/internal/adapter/apifox"
	"sport_game2/internal/model"
)

type Provider interface {
	Predict(match *apifox.MatchBetInfo) ([]model.PredictionResult, error)
	Name() string
}
