-- ROI metrics queries

-- name: UpsertROIMetrics :exec
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
    updated_at = now();

-- name: GetCostAndRevenue :one
SELECT
    COALESCE(SUM(cost_micros), 0)::bigint as cost_micros,
    COALESCE(SUM(revenue_micros), 0)::bigint as revenue_micros,
    COALESCE(SUM(conversions), 0)::bigint as conversions
FROM roi_metrics
WHERE advertiser_id = $1
    AND date >= $2
    AND date <= $3
    AND (sqlc.narg('campaign_id')::bigint IS NULL OR campaign_id = sqlc.narg('campaign_id')::bigint)
    AND (sqlc.narg('adgroup_id')::bigint IS NULL OR adgroup_id = sqlc.narg('adgroup_id')::bigint);

-- name: ListROIMetricsByAdvertiser :many
SELECT advertiser_id, COALESCE(campaign_id, 0)::bigint as campaign_id,
    SUM(cost_micros)::bigint as cost_micros, SUM(revenue_micros)::bigint as revenue_micros
FROM roi_metrics
WHERE date >= $1
GROUP BY advertiser_id, COALESCE(campaign_id, 0)
HAVING SUM(cost_micros) > 0;
