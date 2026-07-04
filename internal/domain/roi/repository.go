package roi

import (
	"context"
	"time"
)

// ConversionRepo aggregates conversion data for ROI calculation.
type ConversionRepo interface {
	GetCostAndRevenue(ctx context.Context, params CostRevenueParams) (*CostRevenueResult, error)
}

// CostRevenueParams defines the scope for cost/revenue aggregation.
type CostRevenueParams struct {
	AdvertiserID int64
	CampaignID   *int64
	AdGroupID    *int64
	Start        time.Time
	End          time.Time
}

// CostRevenueResult holds aggregated cost and revenue.
type CostRevenueResult struct {
	CostMicros    int64
	RevenueMicros int64
	Conversions   int64
}
