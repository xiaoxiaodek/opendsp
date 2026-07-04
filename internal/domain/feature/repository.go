package feature

import "context"

// FeatureRepo retrieves real-time feature sets for model inference.
type FeatureRepo interface {
	GetFeatures(ctx context.Context, userID, mediaID, geoCity string) (FeatureSet, error)
}
