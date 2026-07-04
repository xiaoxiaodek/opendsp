package bidding

import (
	"context"

	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/opendsp/opendsp/internal/domain/fraud"
)

// AntiFraudStage wraps FraudDetectionService as a pipeline Stage.
// This is a PRE-MATCH stage: it assesses the request before index matching.
// If risk score exceeds threshold, it returns nil candidates to abort the bid.
type AntiFraudStage struct {
	service   fraud.FraudDetectionService
	threshold float64
}

// NewAntiFraudStage creates an anti-fraud pipeline stage.
func NewAntiFraudStage(service fraud.FraudDetectionService, threshold float64) *AntiFraudStage {
	return &AntiFraudStage{service: service, threshold: threshold}
}

// Name returns the stage name.
func (s *AntiFraudStage) Name() string { return "antifraud" }

// Process assesses fraud risk. Returns nil candidates if fraudulent.
// Returns empty non-nil slice on pass (pre-match has no candidates yet).
func (s *AntiFraudStage) Process(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
	score, err := s.service.Assess(ctx, req)
	if err != nil {
		return candidates, err
	}
	if score.IsFraudulent(s.threshold) {
		return nil, nil // nil = abort bid
	}
	// Return empty non-nil slice: pre-match passes with no candidates yet
	if candidates == nil {
		return []*bidding.Candidate{}, nil
	}
	return candidates, nil
}
