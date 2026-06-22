-- name: InsertStatEvent :exec
INSERT INTO stat_event (event_type, adgroup_id, creative_id, campaign_id, advertiser_id, media_id, ad_position_id, price, charge_type, device_id, ip, ua, geo_city, freq_result, click_id, event_time)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16);

-- name: AggregateReportHourly :exec
INSERT INTO report_hourly (hour, advertiser_id, campaign_id, ad_group_id, creative_id, media_id, ad_position_id, impressions, clicks, cost, win_count, bid_count)
SELECT date_trunc('hour', event_time) AS hour,
  advertiser_id, campaign_id, adgroup_id, creative_id, media_id, ad_position_id,
  COUNT(*) FILTER (WHERE event_type = 1 AND freq_result = 'ok') AS impressions,
  COUNT(*) FILTER (WHERE event_type = 2) AS clicks,
  COALESCE(SUM(price) FILTER (WHERE event_type = 1 AND freq_result = 'ok'), 0) AS cost,
  COUNT(*) FILTER (WHERE event_type = 1 AND freq_result = 'ok') AS win_count,
  COUNT(*) FILTER (WHERE event_type = 1) AS bid_count
FROM stat_event
WHERE event_time >= $1 AND event_time < $2
GROUP BY hour, advertiser_id, campaign_id, adgroup_id, creative_id, media_id, ad_position_id
ON CONFLICT (hour, ad_group_id, creative_id, media_id, ad_position_id)
DO UPDATE SET
  impressions = report_hourly.impressions + EXCLUDED.impressions,
  clicks = report_hourly.clicks + EXCLUDED.clicks,
  cost = report_hourly.cost + EXCLUDED.cost,
  win_count = report_hourly.win_count + EXCLUDED.win_count,
  bid_count = report_hourly.bid_count + EXCLUDED.bid_count;

-- name: QueryReport :many
SELECT hour, advertiser_id, campaign_id, ad_group_id, creative_id, media_id, ad_position_id, impressions, clicks, conversions, cost, revenue, win_count, bid_count
FROM report_hourly WHERE advertiser_id = $1 AND hour >= $2 AND hour < $3
  AND (sqlc.narg('campaign_id')::bigint IS NULL OR campaign_id = sqlc.narg('campaign_id')::bigint)
  AND (sqlc.narg('ad_group_id')::bigint IS NULL OR ad_group_id = sqlc.narg('ad_group_id')::bigint)
ORDER BY hour DESC;

-- name: QueryDashboardBreakdown :many
SELECT
  CASE WHEN sqlc.arg('dimension')::text = 'campaign' THEN c.id
       WHEN sqlc.arg('dimension')::text = 'adgroup' THEN ag.id
       WHEN sqlc.arg('dimension')::text = 'creative' THEN cr.id
  END AS id,
  CASE WHEN sqlc.arg('dimension')::text = 'campaign' THEN c.name
       WHEN sqlc.arg('dimension')::text = 'adgroup' THEN ag.name
       WHEN sqlc.arg('dimension')::text = 'creative' THEN cr.name
  END AS name,
  COALESCE(SUM(rh.impressions), 0)::bigint AS impressions,
  COALESCE(SUM(rh.clicks), 0)::bigint AS clicks,
  COALESCE(SUM(rh.cost), 0)::float8 AS cost
FROM report_hourly rh
LEFT JOIN campaign c ON c.id = rh.campaign_id
LEFT JOIN ad_group ag ON ag.id = rh.ad_group_id
LEFT JOIN creative cr ON cr.id = rh.creative_id
WHERE rh.advertiser_id = $1 AND rh.hour >= $2 AND rh.hour < $3
GROUP BY 1, 2
ORDER BY cost DESC
LIMIT $4;

-- name: QueryEntityReport :many
SELECT hour, SUM(impressions)::bigint AS impressions, SUM(clicks)::bigint AS clicks, SUM(cost)::float8 AS cost
FROM report_hourly
WHERE advertiser_id = $1
  AND (sqlc.arg('dimension')::text = 'campaign' AND campaign_id = $2
    OR sqlc.arg('dimension')::text = 'adgroup' AND ad_group_id = $2
    OR sqlc.arg('dimension')::text = 'creative' AND creative_id = $2)
  AND hour >= $3 AND hour < $4
GROUP BY hour
ORDER BY hour ASC;

-- name: QueryEntitySubItems :many
SELECT
  CASE WHEN sqlc.arg('dimension')::text = 'campaign' THEN ag.id
       WHEN sqlc.arg('dimension')::text = 'adgroup' THEN cr.id
  END AS id,
  CASE WHEN sqlc.arg('dimension')::text = 'campaign' THEN ag.name
       WHEN sqlc.arg('dimension')::text = 'adgroup' THEN cr.name
  END AS name,
  COALESCE(SUM(rh.impressions), 0)::bigint AS impressions,
  COALESCE(SUM(rh.clicks), 0)::bigint AS clicks,
  COALESCE(SUM(rh.cost), 0)::float8 AS cost
FROM report_hourly rh
LEFT JOIN ad_group ag ON ag.id = rh.ad_group_id
LEFT JOIN creative cr ON cr.id = rh.creative_id
WHERE rh.advertiser_id = $1
  AND (sqlc.arg('dimension')::text = 'campaign' AND rh.campaign_id = $2
    OR sqlc.arg('dimension')::text = 'adgroup' AND rh.ad_group_id = $2)
  AND rh.hour >= $3 AND rh.hour < $4
GROUP BY 1, 2
ORDER BY cost DESC
LIMIT 10;
