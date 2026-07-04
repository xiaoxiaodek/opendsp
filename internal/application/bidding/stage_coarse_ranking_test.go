package bidding

import (
	"context"
	"testing"

	domainBidding "github.com/opendsp/opendsp/internal/domain/bidding"
)

func TestCoarseRankingStage_LRScoring(t *testing.T) {
	model := domainBidding.LRModel{
		Intercept: -2.5,
		Weights: map[string]float64{
			"bid_price": 0.8,
			"hour":      0.3,
		},
	}
	stage := NewCoarseRankingStage(2, model)

	candidates := []*domainBidding.Candidate{
		domainBidding.NewCandidate(1, 10, 50.0),
		domainBidding.NewCandidate(2, 20, 10.0),
		domainBidding.NewCandidate(3, 30, 100.0),
	}

	req := &domainBidding.BidRequest{RequestID: "test"}
	result, err := stage.Process(context.Background(), req, candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 candidates, got %d", len(result))
	}
}

func TestCoarseRankingStage_BelowMax(t *testing.T) {
	model := domainBidding.LRModel{Intercept: 0}
	stage := NewCoarseRankingStage(5, model)

	candidates := []*domainBidding.Candidate{
		domainBidding.NewCandidate(1, 10, 1.0),
		domainBidding.NewCandidate(2, 20, 2.0),
	}

	req := &domainBidding.BidRequest{RequestID: "test"}
	result, err := stage.Process(context.Background(), req, candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected all 2 candidates, got %d", len(result))
	}
}

func TestCoarseRankingStage_Empty(t *testing.T) {
	model := domainBidding.LRModel{Intercept: 0}
	stage := NewCoarseRankingStage(200, model)

	req := &domainBidding.BidRequest{RequestID: "test"}
	result, err := stage.Process(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 candidates, got %d", len(result))
	}
}

func TestCoarseRankingStage_DefaultMax(t *testing.T) {
	model := domainBidding.LRModel{Intercept: 0}
	stage := NewCoarseRankingStage(0, model)
	if stage.maxCandidates != 200 {
		t.Errorf("expected default max 200, got %d", stage.maxCandidates)
	}
}

func TestLRModel_Score(t *testing.T) {
	model := domainBidding.LRModel{
		Intercept: 0,
		Weights:   map[string]float64{"x": 1.0},
	}
	feats := map[string]float64{"x": 0}
	s := model.Score(feats)
	if s != 0.5 {
		t.Errorf("sigmoid(0) = %f, want 0.5", s)
	}

	feats["x"] = 10
	s = model.Score(feats)
	if s <= 0.99 {
		t.Errorf("sigmoid(10) = %f, want > 0.99", s)
	}

	feats["x"] = -10
	s = model.Score(feats)
	if s >= 0.01 {
		t.Errorf("sigmoid(-10) = %f, want < 0.01", s)
	}
}
