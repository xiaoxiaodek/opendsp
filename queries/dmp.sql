-- name: CreateTag :one
INSERT INTO dmp_tag (advertiser_id, name, tag_type, source, source_config, status)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING id;

-- name: UpdateTagDeviceCount :exec
UPDATE dmp_tag SET device_count = $2, status = $3, updated_at = NOW() WHERE id = $1;

-- name: GetTag :one
SELECT id, advertiser_id, name, tag_type, device_count, source, source_config, status, created_at
FROM dmp_tag WHERE id = $1;

-- name: ListTags :many
SELECT id, advertiser_id, name, tag_type, device_count, source, status, created_at
FROM dmp_tag WHERE advertiser_id = $1 AND ($2::smallint IS NULL OR tag_type = $2)
ORDER BY created_at DESC;

-- name: DeleteTag :exec
DELETE FROM dmp_tag WHERE id = $1;

-- name: CreateAudience :one
INSERT INTO dmp_audience (advertiser_id, name, audience_type, rules, status)
VALUES ($1, $2, $3, $4, $5) RETURNING id;

-- name: UpdateAudienceDeviceCount :exec
UPDATE dmp_audience SET device_count = $2, status = $3, updated_at = NOW() WHERE id = $1;

-- name: GetAudience :one
SELECT id, advertiser_id, name, audience_type, rules, device_count, status, created_at
FROM dmp_audience WHERE id = $1;

-- name: ListAudiences :many
SELECT id, advertiser_id, name, audience_type, rules, device_count, status, created_at
FROM dmp_audience WHERE advertiser_id = $1 AND ($2::smallint IS NULL OR audience_type = $2)
ORDER BY created_at DESC;

-- name: DeleteAudience :exec
DELETE FROM dmp_audience WHERE id = $1;

-- name: UpsertDevice :exec
INSERT INTO dmp_device (device_id, device_type, tag_ids, last_seen)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (device_id, device_type) DO UPDATE SET
    tag_ids = dmp_device.tag_ids || EXCLUDED.tag_ids,
    last_seen = NOW();

-- name: GetDeviceTags :one
SELECT tag_ids FROM dmp_device WHERE device_id = $1 AND device_type = $2;
