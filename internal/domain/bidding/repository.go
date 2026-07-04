package bidding

import (
	"context"

	"github.com/opendsp/opendsp/internal/domain/feature"
)

// FeatureRepo retrieves real-time features for scoring.
type FeatureRepo interface {
	GetFeatures(ctx context.Context, userID, mediaID, geoCity string) (feature.FeatureSet, error)
}
