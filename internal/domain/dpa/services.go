package dpa

import (
	"context"

	"github.com/opendsp/opendsp/internal/domain/bidding"
)

// RetargetingService decides whether to retarget a user with DPA products
// and which products to show based on their behavior history.
type RetargetingService interface {
	// SelectProducts chooses the best products to show a user based on behavior.
	// Returns nil if no suitable products found (user won't be retargeted).
	SelectProducts(ctx context.Context, req *bidding.BidRequest, behaviors []UserBehavior) ([]Product, error)
}
