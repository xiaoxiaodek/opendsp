package abtest

import (
	"context"

	"github.com/opendsp/opendsp/internal/domain/bidding"
)

// AssignmentService decides which experiment variant a bid request should use.
// Uses hash-based traffic splitting for deterministic assignment.
type AssignmentService interface {
	// Assign deterministically maps a bid request to an experiment variant.
	// Uses hash(requestID) % 100 to split traffic.
	// Returns nil if the request is not part of any running experiment.
	Assign(ctx context.Context, req *bidding.BidRequest) (*Assignment, error)
}

// AnalysisService compares variants to determine statistical significance.
type AnalysisService interface {
	// CompareVariants checks if any variant is statistically better than control.
	CompareVariants(ctx context.Context, experimentID int64) (*ComparisonResult, error)
}

// ComparisonResult holds the statistical analysis of variant performance.
type ComparisonResult struct {
	ExperimentID   int64
	ControlVariant string
	WinnerVariant  string // empty if no clear winner
	Confidence     float64
	LiftCTR        float64
	LiftCVR        float64
	LiftROAS       float64
}
