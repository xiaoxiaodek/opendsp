-- name: CreateCampaign :one
INSERT INTO campaign (advertiser_id, name, budget, daily_budget, start_time, end_time, pacing, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, version, created_at, updated_at;

-- name: GetCampaign :one
SELECT id, advertiser_id, name, budget, daily_budget, start_time, end_time, pacing, status, version, created_at, updated_at
FROM campaign WHERE id = $1;

-- name: UpdateCampaign :one
UPDATE campaign SET name=$1, budget=$2, daily_budget=$3, start_time=$4, end_time=$5, pacing=$6, version=version+1, updated_at=$7
WHERE id=$8 AND version=$9
RETURNING version;

-- name: UpdateCampaignStatus :exec
UPDATE campaign SET status=$1, version=version+1, updated_at=NOW() WHERE id=$2;

-- name: CountCampaigns :one
SELECT COUNT(*) FROM campaign WHERE advertiser_id = $1
  AND (sqlc.narg('status')::smallint IS NULL OR status = sqlc.narg('status')::smallint);

-- name: ListCampaigns :many
SELECT id, advertiser_id, name, budget, daily_budget, start_time, end_time, pacing, status, version, created_at, updated_at
FROM campaign WHERE advertiser_id = $1
  AND (sqlc.narg('status')::smallint IS NULL OR status = sqlc.narg('status')::smallint)
ORDER BY created_at DESC LIMIT $2 OFFSET $3;
