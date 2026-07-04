-- name: CreateAdGroup :one
INSERT INTO ad_group (campaign_id, name, bid_type, bid_price, daily_budget, freq_cap, freq_period, targeting, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, version, created_at, updated_at;

-- name: GetAdGroup :one
SELECT id, campaign_id, name, bid_type, bid_price, daily_budget, freq_cap, freq_period, targeting, status, version, created_at, updated_at
FROM ad_group WHERE id = $1;

-- name: UpdateAdGroup :one
UPDATE ad_group SET name=$1, bid_price=$2, daily_budget=$3, freq_cap=$4, targeting=$5, version=version+1, updated_at=$6
WHERE id=$7 AND version=$8
RETURNING version;

-- name: UpdateAdGroupStatus :exec
UPDATE ad_group SET status=$1, version=version+1, updated_at=NOW() WHERE id=$2;

-- name: CountAdGroups :one
SELECT COUNT(*) FROM ad_group WHERE (sqlc.narg('campaign_id')::bigint IS NULL OR sqlc.narg('campaign_id')::bigint = 0 OR campaign_id = sqlc.narg('campaign_id')::bigint)
  AND (sqlc.narg('status')::smallint IS NULL OR status = sqlc.narg('status')::smallint);

-- name: ListAdGroups :many
SELECT id, campaign_id, name, bid_type, bid_price, daily_budget, freq_cap, freq_period, targeting, status, version, created_at, updated_at
FROM ad_group WHERE (sqlc.narg('campaign_id')::bigint IS NULL OR sqlc.narg('campaign_id')::bigint = 0 OR campaign_id = sqlc.narg('campaign_id')::bigint)
  AND (sqlc.narg('status')::smallint IS NULL OR status = sqlc.narg('status')::smallint)
ORDER BY created_at DESC LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListActiveAdGroups :many
SELECT ag.id, ag.campaign_id, ag.name, ag.bid_type, ag.bid_price, ag.daily_budget,
       ag.freq_cap, ag.freq_period, ag.targeting, ag.status, ag.version,
       c.advertiser_id AS advertiser_id, c.start_time AS campaign_start_time, c.end_time AS campaign_end_time
FROM ad_group ag
INNER JOIN campaign c ON c.id = ag.campaign_id
WHERE ag.status = $1 AND c.status = $2;
