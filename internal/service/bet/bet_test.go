package bet

import (
	"errors"
	"testing"

	"sport_game2/internal/adapter/apifox"
	"sport_game2/internal/repo"
)

type fakeMatchRepo struct {
	matchInfo *apifox.MatchBetInfo
	err       error
}

func (f *fakeMatchRepo) GetMatchById(matchId string) (*apifox.MatchBetInfo, error) {
	return f.matchInfo, f.err
}

func (f *fakeMatchRepo) GetMatchResult(matchId string) (*repo.DrawResultData, error) {
	return nil, nil
}

type fakeBetRepo struct {
	predID          int64
	predErr         error
	predList        []repo.Prediction
	listErr         error
	aiPreds         []repo.AIPrediction
	expertPreds     []repo.ExpertPrediction
	settledByUser   []repo.Prediction
	settledByUserErr error
	settledByOpen   map[string][]repo.Prediction
	settledByOpenErr error
}

func (f *fakeBetRepo) CreatePrediction(p *repo.Prediction) (int64, error) {
	return f.predID, f.predErr
}

func (f *fakeBetRepo) GetPredictionsByUser(userID int64, limit int) ([]repo.Prediction, error) {
	return f.predList, f.listErr
}

func (f *fakeBetRepo) GetAIPrediction(matchID string) ([]repo.AIPrediction, error) {
	return f.aiPreds, nil
}

func (f *fakeBetRepo) GetExpertPredictions(matchID string) ([]repo.ExpertPrediction, error) {
	return f.expertPreds, nil
}

func (f *fakeBetRepo) CheckDuplicatePrediction(userID int64, matchID, lotteryType, subType string) (bool, error) {
	return false, nil
}

func (f *fakeBetRepo) GetSettledByOpenID(openID, fromTime, toTime string) ([]repo.Prediction, error) {
	if f.settledByOpenErr != nil {
		return nil, f.settledByOpenErr
	}
	if f.settledByOpen != nil {
		if preds, ok := f.settledByOpen[openID]; ok {
			return preds, nil
		}
	}
	return nil, nil
}

func (f *fakeBetRepo) GetSettledByUserID(userID int64, fromTime, toTime string) ([]repo.Prediction, error) {
	return f.settledByUser, f.settledByUserErr
}

type fakeCreditsManager struct {
	balance int
	err     error
}

func (f *fakeCreditsManager) Deduct(userID int64, amount int, reason string, refID int64) (int, error) {
	return f.balance, f.err
}

func (f *fakeCreditsManager) Add(userID int64, amount int, reason string) (int, error) {
	return f.balance + amount, f.err
}

func (f *fakeCreditsManager) IncrementTotalBets(userID int64) error {
	return f.err
}

func TestCreateBetValid(t *testing.T) {
	matchRepo := &fakeMatchRepo{
		matchInfo: &apifox.MatchBetInfo{
			LotteryInfo: apifox.LotteryInfo{
				IsStopSell: false,
				MatchCode:  "001",
				BetEndTimeStr: "2099-12-31 23:59:59",
				BetInfos: []apifox.BetInfo{
					{
						SubType: "6",
						Options: []apifox.BetOption{
							{BetCode: "3", Odds: 1.5},
							{BetCode: "1", Odds: 3.5},
						},
					},
				},
			},
		},
	}
	betRepo := &fakeBetRepo{predID: 100}
	creditMgr := &fakeCreditsManager{balance: 900}

	svc := NewService(matchRepo, betRepo, creditMgr, 60)

	resp, err := svc.CreateBet(1, &CreateBetRequest{
		MatchID: "match_123",
		BetCode: "3",
		Points:  100,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.PredictionID != 100 {
		t.Errorf("expected predID 100, got %d", resp.PredictionID)
	}
	if resp.BalanceAfter != 0 {
		t.Errorf("expected balance 0 (no deduction), got %d", resp.BalanceAfter)
	}
}

func TestCreateBetDefaultPoints(t *testing.T) {
	matchRepo := &fakeMatchRepo{
		matchInfo: &apifox.MatchBetInfo{
			LotteryInfo: apifox.LotteryInfo{
				IsStopSell:    false,
				MatchCode:     "001",
				BetEndTimeStr: "2099-12-31 23:59:59",
				BetInfos: []apifox.BetInfo{
					{SubType: "6", Options: []apifox.BetOption{{BetCode: "3", Odds: 1.5}}},
				},
			},
		},
	}
	betRepo := &fakeBetRepo{predID: 100}
	creditMgr := &fakeCreditsManager{balance: 998}

	svc := NewService(matchRepo, betRepo, creditMgr, 60)

	resp, err := svc.CreateBet(1, &CreateBetRequest{MatchID: "m", BetCode: "3", Points: 0})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.PredictionID != 100 {
		t.Errorf("expected predID 100, got %d", resp.PredictionID)
	}

	resp2, err := svc.CreateBet(1, &CreateBetRequest{MatchID: "m", BetCode: "3", Points: -5})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp2.PredictionID != 100 {
		t.Errorf("expected predID 100, got %d", resp2.PredictionID)
	}
}

func TestCreateBetMatchNotFound(t *testing.T) {
	matchRepo := &fakeMatchRepo{matchInfo: nil, err: nil}
	betRepo := &fakeBetRepo{}
	creditMgr := &fakeCreditsManager{}

	svc := NewService(matchRepo, betRepo, creditMgr, 60)

	_, err := svc.CreateBet(1, &CreateBetRequest{MatchID: "unknown", BetCode: "3", Points: 100})
	if !errors.Is(err, ErrMatchNotFound) {
		t.Errorf("expected ErrMatchNotFound, got %v", err)
	}
}

func TestCreateBetMatchStopped(t *testing.T) {
	matchRepo := &fakeMatchRepo{
		matchInfo: &apifox.MatchBetInfo{
			LotteryInfo: apifox.LotteryInfo{IsStopSell: true},
		},
	}
	betRepo := &fakeBetRepo{}
	creditMgr := &fakeCreditsManager{}

	svc := NewService(matchRepo, betRepo, creditMgr, 60)

	_, err := svc.CreateBet(1, &CreateBetRequest{MatchID: "stopped_match", BetCode: "3", Points: 100})
	if !errors.Is(err, ErrMatchStopped) {
		t.Errorf("expected ErrMatchStopped, got %v", err)
	}
}

func TestCreateBetInvalidSelection(t *testing.T) {
	matchRepo := &fakeMatchRepo{
		matchInfo: &apifox.MatchBetInfo{
			LotteryInfo: apifox.LotteryInfo{
				IsStopSell: false,
				MatchCode:  "001",
				BetEndTimeStr: "2099-12-31 23:59:59",
				BetInfos: []apifox.BetInfo{
					{
						SubType: "6",
						Options: []apifox.BetOption{
							{BetCode: "3", Odds: 1.5},
						},
					},
				},
			},
		},
	}
	betRepo := &fakeBetRepo{}
	creditMgr := &fakeCreditsManager{}

	svc := NewService(matchRepo, betRepo, creditMgr, 60)

	_, err := svc.CreateBet(1, &CreateBetRequest{MatchID: "match_123", BetCode: "999", Points: 100})
	if !errors.Is(err, ErrInvalidSelection) {
		t.Errorf("expected ErrInvalidSelection, got %v", err)
	}
}

func TestGetUserPredictionsEnriched(t *testing.T) {
	betRepo := &fakeBetRepo{
		predList: []repo.Prediction{
			{ID: 1, MatchID: "m1", SubType: "6", BetCode: "3", Status: "pending", Points: 2},
			{ID: 2, MatchID: "m2", SubType: "6", BetCode: "1", Status: "won", Points: 2, IsCorrect: boolPtr(true)},
		},
	}
	matchRepo := &fakeMatchRepo{
		matchInfo: &apifox.MatchBetInfo{
			MatchInfo: apifox.MatchInfo{MatchTimeStr: "2026-01-01 20:00"},
			HomeTeamInfo: apifox.TeamInfo{CnName: "主队"},
			AwayTeamInfo: apifox.TeamInfo{CnName: "客队"},
			TournamentInfo: apifox.TournamentInfo{CnName: "测试联赛"},
		},
	}
	creditMgr := &fakeCreditsManager{}

	svc := NewService(matchRepo, betRepo, creditMgr, 60)

	preds, err := svc.GetUserPredictionsEnriched(1, 20)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(preds) != 2 {
		t.Errorf("expected 2 predictions, got %d", len(preds))
	}
	if preds[0].HomeTeam != "主队" {
		t.Errorf("expected 主队, got %s", preds[0].HomeTeam)
	}
	if preds[1].Status != "won" {
		t.Errorf("expected won, got %s", preds[1].Status)
	}
}

func boolPtr(b bool) *bool { return &b }

func makePreds(n int, correct int) []repo.Prediction {
	preds := make([]repo.Prediction, n)
	for i := 0; i < n; i++ {
		c := false
		if i < correct {
			c = true
		}
		preds[i] = repo.Prediction{ID: int64(i + 1), IsCorrect: boolPtr(c)}
	}
	return preds
}

func TestDecidePKWinner(t *testing.T) {
	cases := []struct {
		name         string
		user, exp, ai int
		want          string
	}{
		{"user_solo", 5, 3, 1, winnerUser},
		{"expert_solo", 1, 5, 3, winnerExpert},
		{"ai_solo", 1, 3, 7, winnerAI},
		{"user_expert_tie", 5, 5, 1, winnerTie},
		{"user_ai_tie", 5, 1, 5, winnerTie},
		{"expert_ai_tie", 1, 5, 5, winnerTie},
		{"all_tie", 4, 4, 4, winnerTie},
		{"all_zero", 0, 0, 0, winnerTie},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := decidePKWinner(tc.user, tc.exp, tc.ai)
			if got != tc.want {
				t.Errorf("decidePKWinner(%d,%d,%d) = %q, want %q", tc.user, tc.exp, tc.ai, got, tc.want)
			}
		})
	}
}

func TestGetDailyPK(t *testing.T) {
	t.Run("no_user_predictions_returns_nil", func(t *testing.T) {
		betRepo := &fakeBetRepo{settledByUser: nil}
		svc := NewService(&fakeMatchRepo{}, betRepo, &fakeCreditsManager{}, 60)
		resp, err := svc.GetDailyPK(1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp != nil {
			t.Errorf("expected nil resp, got %+v", resp)
		}
	})
	t.Run("user_db_error_propagates", func(t *testing.T) {
		betRepo := &fakeBetRepo{settledByUserErr: errors.New("db down")}
		svc := NewService(&fakeMatchRepo{}, betRepo, &fakeCreditsManager{}, 60)
		_, err := svc.GetDailyPK(1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
	t.Run("full_response_and_winner", func(t *testing.T) {
		betRepo := &fakeBetRepo{
			settledByUser: makePreds(4, 3),
			settledByOpen: map[string][]repo.Prediction{
				robotExpertOpenID: makePreds(5, 5),
				robotAIOpenID:     makePreds(3, 2),
			},
		}
		svc := NewService(&fakeMatchRepo{}, betRepo, &fakeCreditsManager{}, 60)
		resp, err := svc.GetDailyPK(1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.UserTotal != 4 || resp.UserWins != 3 {
			t.Errorf("user stats wrong: total=%d wins=%d", resp.UserTotal, resp.UserWins)
		}
		if resp.ExpertTotal != 5 || resp.ExpertWins != 5 {
			t.Errorf("expert stats wrong: total=%d wins=%d", resp.ExpertTotal, resp.ExpertWins)
		}
		if resp.AITotal != 3 || resp.AIWins != 2 {
			t.Errorf("ai stats wrong: total=%d wins=%d", resp.AITotal, resp.AIWins)
		}
		if resp.UserAccuracy != 75 {
			t.Errorf("user accuracy = %v, want 75", resp.UserAccuracy)
		}
		if resp.ExpertAccuracy != 100 {
			t.Errorf("expert accuracy = %v, want 100", resp.ExpertAccuracy)
		}
		if resp.Winner != winnerExpert {
			t.Errorf("winner = %q, want %q", resp.Winner, winnerExpert)
		}
	})
	t.Run("ai_solo_winner_bug_regression", func(t *testing.T) {
		betRepo := &fakeBetRepo{
			settledByUser: makePreds(3, 3),
			settledByOpen: map[string][]repo.Prediction{
				robotExpertOpenID: makePreds(5, 5),
				robotAIOpenID:     makePreds(7, 7),
			},
		}
		svc := NewService(&fakeMatchRepo{}, betRepo, &fakeCreditsManager{}, 60)
		resp, err := svc.GetDailyPK(1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.Winner != winnerAI {
			t.Errorf("winner = %q, want %q (regression: maxWins was not updated for ai)", resp.Winner, winnerAI)
		}
	})
}
