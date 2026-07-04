package bidding

import (
	"context"

	"github.com/opendsp/opendsp/internal/domain/feature"
)

// ScoringService predicts CTR and CVR for candidates given feature context.
type ScoringService interface {
	PredictCTR(ctx context.Context, features feature.FeatureSet, candidate *Candidate) (CTR, error)
	PredictCVR(ctx context.Context, features feature.FeatureSet, candidate *Candidate) (CVR, error)
}

// PricingService calculates eCPM from predictions and bid strategy.
type PricingService interface {
	Calculate(ctx context.Context, candidate *Candidate, strategy PricingStrategy) (ECPM, error)
}
