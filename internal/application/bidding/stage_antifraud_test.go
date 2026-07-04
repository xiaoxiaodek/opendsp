package bidding

import (
	"context"
	"testing"

	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/opendsp/opendsp/internal/domain/fraud"
)

type mockFraudService struct {
	score fraud.RiskScore
}

func (m *mockFraudService) Assess(ctx context.Context, req *bidding.BidRequest) (fraud.RiskScore, error) {
	return m.score, nil
}

func TestAntiFraudStage_Pass(t *testing.T) {
	stage := NewAntiFraudStage(&mockFraudService{score: fraud.RiskScore{Value: 0.1}}, 0.8)
	req := &bidding.BidRequest{RequestID: "test"}
	result, err := stage.Process(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil empty slice (pass), got nil (abort)")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 candidates, got %d", len(result))
	}
}

func TestAntiFraudStage_Block(t *testing.T) {
	stage := NewAntiFraudStage(&mockFraudService{score: fraud.RiskScore{Value: 0.9}}, 0.8)
	req := &bidding.BidRequest{RequestID: "test-fraud"}
	result, err := stage.Process(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result (abort) for fraudulent request")
	}
}

func TestAntiFraudStage_Boundary(t *testing.T) {
	stage := NewAntiFraudStage(&mockFraudService{score: fraud.RiskScore{Value: 0.8}}, 0.8)
	req := &bidding.BidRequest{RequestID: "test-boundary"}
	result, err := stage.Process(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected abort at threshold boundary (>= 0.8)")
	}
}

func TestAntiFraudStage_StageName(t *testing.T) {
	stage := NewAntiFraudStage(&mockFraudService{score: fraud.Clean()}, 0.8)
	if stage.Name() != "antifraud" {
		t.Errorf("expected stage name 'antifraud', got '%s'", stage.Name())
	}
}

func TestAntiFraudStage_WithCandidates(t *testing.T) {
	stage := NewAntiFraudStage(&mockFraudService{score: fraud.Clean()}, 0.8)
	candidates := []*bidding.Candidate{
		bidding.NewCandidate(1, 10, 1.0),
		bidding.NewCandidate(2, 20, 2.0),
	}
	req := &bidding.BidRequest{RequestID: "test-with-candidates"}
	result, err := stage.Process(context.Background(), req, candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 candidates passed through, got %d", len(result))
	}
}

func TestAntiFraudStage_WithReasons(t *testing.T) {
	score := fraud.RiskScore{Value: 0.9, Reasons: []string{fraud.ReasonRequestRateIP}}
	stage := NewAntiFraudStage(&mockFraudService{score: score}, 0.8)
	req := &bidding.BidRequest{RequestID: "test-reasons"}
	result, err := stage.Process(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result (abort) for fraudulent request with reasons")
	}
}
