package bidding

import (
	"context"

	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/opendsp/opendsp/internal/domain/budget"
)

// PacingStage performs probabilistic budget pacing to smooth spend across the day.
// This is a POST-MATCH stage.
type PacingStage struct {
	service budget.PacingService
}

// NewPacingStage creates a pacing pipeline stage.
func NewPacingStage(service budget.PacingService) *PacingStage {
	return &PacingStage{service: service}
}

// Name returns the stage name.
func (s *PacingStage) Name() string { return "pacing" }

// Process probabilistically drops candidates based on budget consumption rate.
func (s *PacingStage) Process(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
	if s.service == nil {
		return candidates, nil
	}

	// Filter candidates through pacing. Currently uses ad group bid price
	// as a proxy for spend. Future: track actual clearing price per bid.
	var kept []*bidding.Candidate
	for _, c := range candidates {
		shouldBid, err := s.service.ShouldBid(ctx, int64(c.AdGroupID), 0, 0)
		if err != nil {
			// On error, keep the candidate (fail open).
			kept = append(kept, c)
			continue
		}
		if shouldBid {
			kept = append(kept, c)
		}
	}

	return kept, nil
}
