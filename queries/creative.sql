-- name: CreateCreative :one
INSERT INTO creative (ad_group_id, name, creative_type, asset_url, asset_size, asset_duration, asset_width, asset_height, asset_mime,
  title, description, cta_text, brand_name, brand_logo, landing_url, deeplink_url, imp_tracker, click_tracker, third_party_trackers)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
RETURNING id, version, created_at, updated_at;

-- name: CountCreativesByAdGroup :one
SELECT COUNT(*) FROM creative WHERE (sqlc.narg('ad_group_id')::bigint IS NULL OR ad_group_id = sqlc.narg('ad_group_id')::bigint) AND is_valid = 1;

-- name: ListCreativesByAdGroup :many
SELECT id, ad_group_id, name, creative_type, asset_url, asset_size, asset_duration, asset_width, asset_height, asset_mime,
  title, description, cta_text, brand_name, brand_logo, landing_url, deeplink_url, imp_tracker, click_tracker, third_party_trackers,
  audit_status, audit_reason, version, created_at, updated_at
FROM creative WHERE (sqlc.narg('ad_group_id')::bigint IS NULL OR ad_group_id = sqlc.narg('ad_group_id')::bigint) AND is_valid = 1
ORDER BY created_at DESC LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListApprovedCreativesByAdGroup :many
SELECT id, ad_group_id, name, creative_type, asset_url, asset_size, asset_duration, asset_width, asset_height, asset_mime,
  title, description, cta_text, brand_name, brand_logo, landing_url, deeplink_url, imp_tracker, click_tracker, third_party_trackers,
  audit_status
FROM creative WHERE ad_group_id = $1 AND audit_status = $2 AND is_valid = 1;

-- name: SubmitCreativeAudit :exec
UPDATE creative SET audit_status = $1, version = version + 1, updated_at = NOW() WHERE id = $2;

-- name: UpdateCreativeAuditStatus :exec
UPDATE creative SET audit_status = $1, audit_reason = $2, version = version + 1, updated_at = NOW() WHERE id = $3;

-- name: UpdateCreative :exec
UPDATE creative SET name=$1, asset_url=$2, asset_width=$3, asset_height=$4, asset_duration=$5,
  title=$6, description=$7, landing_url=$8, imp_tracker=$9, click_tracker=$10,
  version=version+1, updated_at=NOW()
WHERE id=$11 AND version=$12;
