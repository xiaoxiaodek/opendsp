// Package dpa defines the Dynamic Product Ads domain model.
// DPA tracks user behavior on advertiser sites and retargets them
// with dynamically generated product creatives on other media.
package dpa

import "time"

// Product represents a product from an advertiser's product feed.
type Product struct {
	ID          string
	AdvertiserID int64
	Title       string
	Description string
	ImageURL    string
	LandingURL  string
	Price       float64
	Category    string
	Brand       string
	InStock     bool
	UpdatedAt   time.Time
}

// ProductFeed represents a collection of products for DPA campaigns.
type ProductFeed struct {
	ID           int64
	AdvertiserID int64
	Name         string
	FeedURL      string
	ProductCount int64
	LastSyncedAt time.Time
}

// UserBehavior tracks a user's interaction with products on an advertiser site.
type UserBehavior struct {
	UserID    string
	ProductID string
	Action    string // "view", "add_to_cart", "purchase"
	Timestamp time.Time
	Value     float64
}
