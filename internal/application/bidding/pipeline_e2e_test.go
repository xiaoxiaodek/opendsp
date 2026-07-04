package bidding

import (
	"context"
	"testing"

	"github.com/opendsp/opendsp/internal/domain/abtest"
	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/opendsp/opendsp/internal/domain/fraud"
)

func TestFullPipeline_PreMatchToPostMatch(t *testing.T) {
	// Pre-match stages
	preMatch := []Stage{
		NewABTestStage(&mockABTestService{}),
		NewAntiFraudStage(&mockFraudService{score: fraud.Clean()}, 0.8),
	}

	// Post-match stages
	postMatch := []Stage{
		NewCoarseRankingStage(3, bidding.LRModel{Intercept: 0}),
		NewStage("mock_feature", func(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
			return candidates, nil
		}),
		NewStage("mock_scoring", func(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
			for _, c := range candidates {
				c.PredCTR = 0.01
				c.PredCVR = 0.005
			}
			return candidates, nil
		}),
		NewStage("mock_dpa", func(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
			for _, c := range candidates {
				if c.HasPredictions() {
					c.FinalScore *= 1.2
				}
			}
			return candidates, nil
		}),
		NewPricingStage(bidding.PricingStrategyECPM, 0, nil),
		NewStage("mock_pacing", func(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
			return candidates, nil
		}),
		NewStage("mock_budget_guard", func(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
			return candidates, nil
		}),
	}

	pipeline := NewPipeline(preMatch, postMatch)

	req := &bidding.BidRequest{RequestID: "e2e-test", MediaID: "iqiyi"}

	// Pre-match should pass
	if !pipeline.RunPreMatch(context.Background(), req) {
		t.Fatal("pre-match should pass for clean request")
	}

	// Simulate index matching producing candidates
	candidates := []*bidding.Candidate{
		bidding.NewCandidate(1, 10, 2.0),
		bidding.NewCandidate(2, 20, 1.0),
		bidding.NewCandidate(3, 30, 5.0),
		bidding.NewCandidate(4, 40, 3.0),
		bidding.NewCandidate(5, 50, 1.5),
	}

	result, aborted := pipeline.RunPostMatch(context.Background(), req, candidates)
	if aborted {
		t.Fatal("post-match should not abort")
	}
	if len(result) == 0 {
		t.Fatal("post-match should produce at least 1 candidate")
	}
	if len(result) > 3 {
		t.Errorf("coarse ranking should limit to 3, got %d", len(result))
	}

	// Verify scoring and pricing were applied
	for _, c := range result {
		if c.PredCTR == 0 {
			t.Error("candidate should have pCTR after scoring stage")
		}
		if c.PredCVR == 0 {
			t.Error("candidate should have pCVR after scoring stage")
		}
		if c.ECPM == 0 {
			t.Error("candidate should have eCPM after pricing stage")
		}
		if c.FinalScore == 0 {
			t.Error("candidate should have FinalScore")
		}
	}
}

func TestFullPipeline_AntiFraudBlock(t *testing.T) {
	preMatch := []Stage{
		NewAntiFraudStage(&mockFraudService{score: fraud.RiskScore{Value: 1.0, Reasons: []string{fraud.ReasonRequestRateIP}}}, 0.8),
	}

	pipeline := NewPipeline(preMatch, nil)
	req := &bidding.BidRequest{RequestID: "e2e-block", MediaID: "iqiyi", IP: "10.0.0.1"}

	if pipeline.RunPreMatch(context.Background(), req) {
		t.Fatal("pre-match should abort for fraudulent request")
	}
}

func TestFullPipeline_PricingStrategy(t *testing.T) {
	// Test ECPM strategy
	ecpmStage := NewPricingStage(bidding.PricingStrategyECPM, 0, nil)
	c := bidding.NewCandidate(1, 10, 2.0)
	c.PredCTR = 0.01
	c.PredCVR = 0.005

	result, err := ecpmStage.Process(context.Background(), &bidding.BidRequest{}, []*bidding.Candidate{c})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatal("expected 1 candidate")
	}
	if result[0].ECPM == 0 {
		t.Error("expected non-zero eCPM with ECPM strategy")
	}
	if result[0].FinalScore == 0 {
		t.Error("expected FinalScore set from eCPM")
	}

	// Test OXBI strategy
	oxbiStage := NewPricingStage(bidding.PricingStrategyOXBI, 2.0, nil)
	c2 := bidding.NewCandidate(2, 20, 2.0)
	c2.PredCTR = 0.01
	c2.PredCVR = 0.01

	result2, err := oxbiStage.Process(context.Background(), &bidding.BidRequest{}, []*bidding.Candidate{c2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2[0].ECPM == 0 {
		t.Error("expected non-zero eCPM with OXBI strategy")
	}
}

type mockABTestService struct{}

func (m *mockABTestService) Assign(ctx context.Context, req *bidding.BidRequest) (*abtest.Assignment, error) {
	return nil, nil
}
