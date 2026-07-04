package dpa

import (
	"context"
	"time"
)

// ProductRepo manages the product catalog for DPA campaigns.
type ProductRepo interface {
	// GetByUserBehaviors returns products that match a user's recent behaviors.
	GetByUserBehaviors(ctx context.Context, advertiserID int64, behaviors []UserBehavior) ([]Product, error)

	// UpsertProducts bulk-inserts or updates products from a feed sync.
	UpsertProducts(ctx context.Context, products []Product) error
}

// BehaviorRepo tracks user behavior events on advertiser sites.
type BehaviorRepo interface {
	// GetRecentBehaviors returns recent user behaviors within the lookback window.
	GetRecentBehaviors(ctx context.Context, userID string, advertiserID int64, since time.Time) ([]UserBehavior, error)

	// RecordBehavior stores a user behavior event.
	RecordBehavior(ctx context.Context, behavior UserBehavior) error
}
