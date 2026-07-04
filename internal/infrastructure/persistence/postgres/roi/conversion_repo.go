package roi

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
	domainroi "github.com/opendsp/opendsp/internal/domain/roi"
)

// ConversionRepo implements domain/roi.ConversionRepo using PostgreSQL.
type ConversionRepo struct {
	pool    *pgxpool.Pool
	queries *dbsqlc.Queries
}

// NewConversionRepo creates a PostgreSQL-backed conversion repository.
func NewConversionRepo(pool *pgxpool.Pool) *ConversionRepo {
	return &ConversionRepo{pool: pool, queries: dbsqlc.New(pool)}
}

// GetCostAndRevenue aggregates cost and revenue from roi_metrics table.
func (r *ConversionRepo) GetCostAndRevenue(ctx context.Context, params domainroi.CostRevenueParams) (*domainroi.CostRevenueResult, error) {
	var startDate, endDate pgtype.Date
	_ = startDate.Scan(params.Start.Format("2006-01-02"))
	_ = endDate.Scan(params.End.Format("2006-01-02"))

	row, err := r.queries.GetCostAndRevenue(ctx, &dbsqlc.GetCostAndRevenueParams{
		AdvertiserID: params.AdvertiserID,
		Date:         startDate,
		Date_2:       endDate,
		CampaignID:   params.CampaignID,
		AdgroupID:    params.AdGroupID,
	})
	if err != nil {
		return nil, err
	}
	return &domainroi.CostRevenueResult{
		CostMicros:    row.CostMicros,
		RevenueMicros: row.RevenueMicros,
		Conversions:   row.Conversions,
	}, nil
}

// UpsertMetrics updates or inserts ROI metrics for a given date.
func (r *ConversionRepo) UpsertMetrics(ctx context.Context, advertiserID, campaignID, adGroupID int64, date time.Time, costMicros, revenueMicros int64, conversions int) error {
	var pgDate pgtype.Date
	_ = pgDate.Scan(date.Format("2006-01-02"))

	return r.queries.UpsertROIMetrics(ctx, &dbsqlc.UpsertROIMetricsParams{
		AdvertiserID:  advertiserID,
		CampaignID:    &campaignID,
		AdgroupID:     &adGroupID,
		Date:          pgDate,
		CostMicros:    costMicros,
		RevenueMicros: revenueMicros,
		Conversions:   int32(conversions),
	})
}
