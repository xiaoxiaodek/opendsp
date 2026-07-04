package fraud

import (
	"context"

	"github.com/opendsp/opendsp/internal/domain/bidding"
)

// FraudDetectionService assesses whether a bid request is fraudulent.
type FraudDetectionService interface {
	Assess(ctx context.Context, req *bidding.BidRequest) (RiskScore, error)
}

// ImpressionEvent carries data needed for post-bid fraud checks on impressions.
type ImpressionEvent struct {
	RequestID  string
	MediaID    string
	PositionID string
	IP         string
	DeviceID   string
	UserAgent  string
}

// ClickEvent carries data needed for post-bid fraud checks on clicks.
type ClickEvent struct {
	RequestID  string
	MediaID    string
	PositionID string
	IP         string
	DeviceID   string
	UserAgent  string
}

// PostBidChecker runs fraud checks after impression/click events.
type PostBidChecker interface {
	CheckImpression(ctx context.Context, event ImpressionEvent) []string
	CheckClick(ctx context.Context, event ClickEvent) []string
}
