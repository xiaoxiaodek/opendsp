package bidding

import (
	"context"

	"github.com/opendsp/opendsp/internal/domain/bidding"
)

// ScoringStage runs pCTR and pCVR prediction on each candidate.
// This is a POST-MATCH stage.
type ScoringStage struct {
	service bidding.ScoringService
}

// NewScoringStage creates a scoring pipeline stage.
func NewScoringStage(service bidding.ScoringService) *ScoringStage {
	return &ScoringStage{service: service}
}

// Name returns the stage name.
func (s *ScoringStage) Name() string { return "scoring" }

// Process predicts CTR and CVR for each candidate.
func (s *ScoringStage) Process(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
	for _, c := range candidates {
		ctr, err := s.service.PredictCTR(ctx, c.Features, c)
		if err != nil {
			continue
		}
		cvr, err := s.service.PredictCVR(ctx, c.Features, c)
		if err != nil {
			continue
		}
		c.PredCTR = ctr.Value
		c.PredCVR = cvr.Value
	}
	return candidates, nil
}
