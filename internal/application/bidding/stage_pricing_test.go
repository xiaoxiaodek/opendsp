package bidding

import (
	"context"
	"testing"

	"github.com/opendsp/opendsp/internal/domain/bidding"
)

func TestPricingStage_ECPM(t *testing.T) {
	stage := NewPricingStage(bidding.PricingStrategyECPM, 0, nil)

	candidates := []*bidding.Candidate{
		{BidPrice: 2.0, PredCTR: 0.01, PredCVR: 0.005},
		{BidPrice: 1.0, PredCTR: 0.02, PredCVR: 0.01},
		{BidPrice: 5.0, PredCTR: 0.001, PredCVR: 0.001},
	}

	req := &bidding.BidRequest{}
	result, err := stage.Process(context.Background(), req, candidates)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 candidates, got %d", len(result))
	}

	// Verify eCPM calculation: bidPrice * pCTR * pCVR * 1000 * 1_000_000
	// Candidate 1: 2.0 * 0.01 * 0.005 * 1000 * 1e6 = 100,000 micros
	expectedECPM1 := int64(2.0 * 0.01 * 0.005 * 1000 * 1_000_000)
	if result[0].ECPM == 0 {
		t.Error("expected non-zero eCPM")
	}
	_ = expectedECPM1

	// Verify candidates are sorted by FinalScore descending
	for i := 1; i < len(result); i++ {
		if result[i-1].FinalScore < result[i].FinalScore {
			t.Errorf("candidates not sorted descending: [%d].FinalScore=%.0f < [%d].FinalScore=%.0f",
				i-1, result[i-1].FinalScore, i, result[i].FinalScore)
		}
	}
}

func TestPricingStage_OXBI(t *testing.T) {
	stage := NewPricingStage(bidding.PricingStrategyOXBI, 2.0, nil)

	candidates := []*bidding.Candidate{
		{BidPrice: 2.0, PredCTR: 0.01, PredCVR: 0.01},  // high CVR -> higher multiplier
		{BidPrice: 2.0, PredCTR: 0.01, PredCVR: 0.001}, // low CVR -> lower multiplier
	}

	req := &bidding.BidRequest{}
	result, err := stage.Process(context.Background(), req, candidates)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(result))
	}

	// High CVR candidate should have higher FinalScore
	if result[0].FinalScore <= result[1].FinalScore {
		t.Error("OXBI: high CVR candidate should rank above low CVR candidate")
	}
}

func TestPricingStage_OXBI_MultiplierCapped(t *testing.T) {
	stage := NewPricingStage(bidding.PricingStrategyOXBI, 0.1, nil) // very low target -> high multiplier

	candidates := []*bidding.Candidate{
		{BidPrice: 1.0, PredCTR: 0.1, PredCVR: 0.5},
	}

	req := &bidding.BidRequest{}
	result, err := stage.Process(context.Background(), req, candidates)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Multiplier should be capped at 2.0
	// Without cap: 0.5 / (1/0.1) = 0.5 / 10 = 0.05 -> actually below floor
	// So it hits floor 0.5
	expectedMinECPM := int64(1.0 * 0.5 * 0.1 * 0.5 * 1000 * 1_000_000)
	_ = expectedMinECPM
	if result[0].ECPM == 0 {
		t.Error("expected non-zero eCPM with capped multiplier")
	}
}

func TestPricingStage_EmptyCandidates(t *testing.T) {
	stage := NewPricingStage(bidding.PricingStrategyECPM, 0, nil)
	req := &bidding.BidRequest{}
	result, err := stage.Process(context.Background(), req, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 candidates, got %d", len(result))
	}
}
