package bidding

import (
	"context"

	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/opendsp/opendsp/internal/domain/dpa"
)

// DPAStage handles Dynamic Product Ad retargeting.
// This is a POST-MATCH stage that boosts candidates whose products match
// the user's recent browsing/purchase behavior on advertiser sites.
type DPAStage struct {
	service dpa.RetargetingService
}

// NewDPAStage creates a DPA retargeting pipeline stage.
func NewDPAStage(service dpa.RetargetingService) *DPAStage {
	return &DPAStage{service: service}
}

// Name returns the stage name.
func (s *DPAStage) Name() string { return "dpa" }

// Process boosts candidates that match the user's product interests.
// Products matching recent user behaviors get a score multiplier.
// If no behavior data is available, passes all candidates through unchanged.
func (s *DPAStage) Process(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
	if s.service == nil || len(candidates) == 0 {
		return candidates, nil
	}

	// Try to get retargeting products for this user.
	// If no behaviors exist, SelectProducts returns nil and we pass through.
	products, err := s.service.SelectProducts(ctx, req, nil)
	if err != nil || len(products) == 0 {
		return candidates, nil
	}

	// Build a set of product IDs the user is interested in
	interestSet := make(map[string]bool, len(products))
	for _, p := range products {
		interestSet[p.ID] = true
	}

	// Boost candidates that match user's product interests
	for _, c := range candidates {
		// Check if this candidate's creative is associated with a product the user viewed.
		// The product ID is embedded in the creative metadata (future: creative→product mapping).
		// For now, boost all candidates slightly if user has any product interest.
		if c.HasPredictions() {
			c.FinalScore *= 1.2 // 20% boost for DPA-eligible candidates
		}
	}

	return candidates, nil
}
