package bidding

import (
	"context"

	"github.com/opendsp/opendsp/internal/domain/bidding"
	"github.com/opendsp/opendsp/internal/domain/feature"
)

// FeatureAssemblyStage enriches candidates with real-time features for scoring.
// This is a POST-MATCH stage.
type FeatureAssemblyStage struct {
	featureRepo feature.FeatureRepo
}

// NewFeatureAssemblyStage creates a feature assembly stage.
func NewFeatureAssemblyStage(repo feature.FeatureRepo) *FeatureAssemblyStage {
	return &FeatureAssemblyStage{featureRepo: repo}
}

// Name returns the stage name.
func (s *FeatureAssemblyStage) Name() string { return "feature_assembly" }

// Process attaches a FeatureSet to each candidate.
func (s *FeatureAssemblyStage) Process(ctx context.Context, req *bidding.BidRequest, candidates []*bidding.Candidate) ([]*bidding.Candidate, error) {
	features, err := s.featureRepo.GetFeatures(ctx, req.UserID, req.MediaID, req.GeoCity)
	if err != nil {
		features = feature.NewFeatureSet()
	}

	for _, c := range candidates {
		c.Features = features
	}

	return candidates, nil
}
