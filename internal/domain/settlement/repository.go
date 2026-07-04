package settlement

import (
	"context"
	"time"
)

// SettlementRepo handles reconciliation between DSP and ADX billing.
type SettlementRepo interface {
	Reconcile(ctx context.Context, advertiserID int64, start, end time.Time) ([]Discrepancy, error)
	RecordSettlement(ctx context.Context, event SettlementEvent) error
}

// SettlementEvent is a single ADX settlement record.
type SettlementEvent struct {
	EventTime    time.Time
	DSPBidID     string
	ADXBidID     string
	MediaID      string
	AdvertiserID int64
	AdGroupID    int64
	DSPCost      int64
	ADXCost      int64
	Currency     string
}
