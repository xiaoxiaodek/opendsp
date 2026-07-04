package roi

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	domainroi "github.com/opendsp/opendsp/internal/domain/roi"
)

// ConversionRepo implements domain/roi.ConversionRepo using PostgreSQL.
type ConversionRepo struct {
	pool *pgxpool.Pool
}

// NewConversionRepo creates a PostgreSQL-backed conversion repository.
func NewConversionRepo(pool *pgxpool.Pool) *ConversionRepo {
	return &ConversionRepo{pool: pool}
}

// GetCostAndRevenue aggregates cost and revenue from roi_metrics table.
func (r *ConversionRepo) GetCostAndRevenue(ctx context.Context, params domainroi.CostRevenueParams) (*domainroi.CostRevenueResult, error) {
	query := `
		SELECT
			COALESCE(SUM(cost_micros), 0) as cost_micros,
			COALESCE(SUM(revenue_micros), 0) as revenue_micros,
			COALESCE(SUM(conversions), 0) as conversions
		FROM roi_metrics
		WHERE advertiser_id = $1
			AND date >= $2
			AND date <= $3
			AND ($4::bigint IS NULL OR campaign_id = $4)
			AND ($5::bigint IS NULL OR adgroup_id = $5)
	`

	var result domainroi.CostRevenueResult
	err := r.pool.QueryRow(ctx, query,
		params.AdvertiserID,
		params.Start.Format("2006-01-02"),
		params.End.Format("2006-01-02"),
		params.CampaignID,
		params.AdGroupID,
	).Scan(&result.CostMicros, &result.RevenueMicros, &result.Conversions)

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpsertMetrics updates or inserts ROI metrics for a given date.
func (r *ConversionRepo) UpsertMetrics(ctx context.Context, advertiserID, campaignID, adGroupID int64, date time.Time, costMicros, revenueMicros int64, conversions int) error {
	query := `
		INSERT INTO roi_metrics (advertiser_id, campaign_id, adgroup_id, date, cost_micros, revenue_micros, conversions, roas, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 0, now())
		ON CONFLICT (advertiser_id, campaign_id, adgroup_id, date)
		DO UPDATE SET
			cost_micros = roi_metrics.cost_micros + EXCLUDED.cost_micros,
			revenue_micros = roi_metrics.revenue_micros + EXCLUDED.revenue_micros,
			conversions = roi_metrics.conversions + EXCLUDED.conversions,
			roas = CASE
				WHEN roi_metrics.cost_micros + EXCLUDED.cost_micros > 0
				THEN (roi_metrics.revenue_micros + EXCLUDED.revenue_micros)::numeric / (roi_metrics.cost_micros + EXCLUDED.cost_micros)::numeric
				ELSE 0
			END,
			updated_at = now()
	`

	_, err := r.pool.Exec(ctx, query,
		advertiserID, campaignID, adGroupID, date,
		costMicros, revenueMicros, conversions,
	)
	return err
}
