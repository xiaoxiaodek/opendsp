package bidding

import (
	"context"

	"github.com/opendsp/opendsp/internal/domain/abtest"
	"github.com/opendsp/opendsp/internal/domain/bidding"
)

// ABTestStage handles A/B test experiment assignment.
// This is a PRE-MATCH stage that assigns the request to an experiment variant.
type ABTestStage struct {
	service abtest.AssignmentService
}

func NewABTestStage(service abtest.AssignmentService) *ABTestStage {
	return &ABTestStage{service: service}
}

func (s *ABTestStage) Name() string { return "abtest" }

// Process assigns the bid request to an experiment variant.
// Returns empty non-nil slice on pass (pre-match has no candidates).
func (s *ABTestStage) Process(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
	if s.service != nil {
		assignment, _ := s.service.Assign(ctx, req)
		_ = assignment
	}
	if candidates == nil {
		return []*bidding.Candidate{}, nil
	}
	return candidates, nil
}
