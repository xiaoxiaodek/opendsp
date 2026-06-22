-- name: InsertConversionEvent :exec
INSERT INTO conversion_event
(click_id, event_type, adgroup_id, creative_id, campaign_id, advertiser_id,
 media_id, ad_position_id, price, revenue, device_id, ip, ua, geo_city, extra, event_time)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16);

-- name: UpsertReportConversions :exec
INSERT INTO report_hourly (hour, advertiser_id, campaign_id, ad_group_id, creative_id, media_id, ad_position_id, conversions, revenue)
VALUES (date_trunc('hour', NOW()), $1, $2, $3, $4, $5, $6, 1, $7)
ON CONFLICT (hour, ad_group_id, creative_id, media_id, ad_position_id)
DO UPDATE SET
  conversions = report_hourly.conversions + 1,
  revenue = report_hourly.revenue + EXCLUDED.revenue;

-- name: FindClickID :one
SELECT click_id, event_type, adgroup_id, creative_id, campaign_id, advertiser_id,
       media_id, ad_position_id, price, device_id, ip, ua, geo_city, event_time
FROM stat_event
WHERE click_id = $1 AND event_type = 2
LIMIT 1;

-- name: FindClickIDInWindow :one
SELECT click_id, event_type, adgroup_id, creative_id, campaign_id, advertiser_id,
       media_id, ad_position_id, price, device_id, ip, ua, geo_city, event_time
FROM stat_event
WHERE click_id = $1 AND event_type = 2 AND event_time >= $2
LIMIT 1;

-- name: CountConversionsByAdvertiser :one
SELECT COUNT(*) FROM conversion_event
WHERE advertiser_id = $1 AND event_time >= $2 AND event_time < $3;

-- name: SumRevenueByAdvertiser :one
SELECT COALESCE(SUM(revenue), 0)::float8 FROM conversion_event
WHERE advertiser_id = $1 AND event_time >= $2 AND event_time < $3;
