-- name: UpsertCreativeSync :one
INSERT INTO creative_sync (creative_id, platform, status, external_id, external_tvid, reason, raw_response, synced_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (creative_id, platform)
DO UPDATE SET
    status = EXCLUDED.status,
    external_id = COALESCE(EXCLUDED.external_id, creative_sync.external_id),
    external_tvid = COALESCE(EXCLUDED.external_tvid, creative_sync.external_tvid),
    reason = EXCLUDED.reason,
    raw_response = EXCLUDED.raw_response,
    synced_at = EXCLUDED.synced_at,
    updated_at = NOW()
RETURNING id;

-- name: GetCreativeSync :one
SELECT id, creative_id, platform, status, external_id, external_tvid, reason, raw_response, synced_at, created_at, updated_at
FROM creative_sync WHERE creative_id = $1 AND platform = $2;

-- name: ListPendingCreativeSync :many
SELECT cs.id, cs.creative_id, cs.platform, cs.status, cs.external_id, cs.external_tvid, cs.reason,
       c.asset_url, c.asset_mime, c.asset_width, c.asset_height, c.asset_duration,
       c.title, c.description, c.landing_url, c.deeplink_url, c.imp_tracker, c.click_tracker,
       c.ad_group_id, aig.campaign_id, cam.advertiser_id
FROM creative_sync cs
JOIN creative c ON cs.creative_id = c.id
JOIN ad_group aig ON c.ad_group_id = aig.id
JOIN campaign cam ON aig.campaign_id = cam.id
WHERE cs.platform = $1 AND cs.status IN (1, 2);

-- name: ListApprovedCreativeSync :many
SELECT cs.creative_id, cs.platform, cs.external_tvid
FROM creative_sync cs
WHERE cs.status = 3;

-- name: UpsertAdvertiserSync :one
INSERT INTO advertiser_sync (advertiser_id, platform, status, external_ad_id, reason, raw_response, synced_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (advertiser_id, platform)
DO UPDATE SET
    status = EXCLUDED.status,
    external_ad_id = COALESCE(EXCLUDED.external_ad_id, advertiser_sync.external_ad_id),
    reason = EXCLUDED.reason,
    raw_response = EXCLUDED.raw_response,
    synced_at = EXCLUDED.synced_at,
    updated_at = NOW()
RETURNING id;

-- name: GetAdvertiserSync :one
SELECT id, advertiser_id, platform, status, external_ad_id, reason, raw_response, synced_at, created_at, updated_at
FROM advertiser_sync WHERE advertiser_id = $1 AND platform = $2;

-- name: ListPendingAdvertiserSync :many
SELECT ads.id, ads.advertiser_id, ads.platform, ads.status, ads.external_ad_id, ads.reason,
       adv.name, adv.industry
FROM advertiser_sync ads
JOIN advertiser adv ON ads.advertiser_id = adv.id
WHERE ads.platform = $1 AND ads.status IN (1, 2);
