package feature

import (
	"context"
	"fmt"
	"strconv"

	"github.com/opendsp/opendsp/internal/domain/feature"
	"github.com/redis/go-redis/v9"
)

// FeatureRepo retrieves real-time features from Redis for model inference.
// Looks up user-level features (behavior history) and context-level features
// (media/position/geo averages), merging them into a single FeatureSet.
type FeatureRepo struct {
	rdb *redis.Client
}

func NewFeatureRepo(rdb *redis.Client) *FeatureRepo {
	return &FeatureRepo{rdb: rdb}
}

// GetFeatures implements domain/feature.FeatureRepo.
// Retrieves user features and context features from Redis, merging both.
// Gracefully degrades: if Redis is unavailable or keys are missing,
// returns an empty FeatureSet rather than failing the bid.
func (r *FeatureRepo) GetFeatures(ctx context.Context, userID, mediaID, geoCity string) (feature.FeatureSet, error) {
	fs := feature.NewFeatureSet()

	// User-level features: recent behavior, historical CTR/CVR
	if userID != "" {
		userKey := fmt.Sprintf("feature:user:%s", userID)
		userFeatures, err := r.rdb.HGetAll(ctx, userKey).Result()
		if err != nil && err != redis.Nil {
			return fs, fmt.Errorf("feature: get user features: %w", err)
		}
		for k, v := range userFeatures {
			if fv, err := strconv.ParseFloat(v, 64); err == nil {
				fs.Set(k, fv)
			}
		}
	}

	// Context-level features: media/geo averages
	ctxKey := fmt.Sprintf("feature:context:%s:%s", mediaID, geoCity)
	ctxFeatures, err := r.rdb.HGetAll(ctx, ctxKey).Result()
	if err != nil && err != redis.Nil {
		return fs, fmt.Errorf("feature: get context features: %w", err)
	}
	for k, v := range ctxFeatures {
		if fv, err := strconv.ParseFloat(v, 64); err == nil {
			fs.Set(k, fv)
		}
	}

	return fs, nil
}
