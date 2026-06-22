-- Advertiser queries
-- name: CreateAdvertiser :one
INSERT INTO advertiser (name, industry, contact_name, contact_email, address, website, brand_names)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id;

-- name: GetAdvertiser :one
SELECT id, name, industry, contact_name, contact_email, balance, status,
       COALESCE(qualification_status, 0) as qualification_status,
       qualification_reason, COALESCE(credit_limit, 0) as credit_limit,
       address, website, brand_names, created_at, updated_at
FROM advertiser WHERE id = $1;

-- name: CountAdvertisers :one
SELECT COUNT(*) FROM advertiser
WHERE (sqlc.narg('status')::smallint IS NULL OR status = sqlc.narg('status')::smallint)
  AND (sqlc.narg('qualification_status')::smallint IS NULL OR COALESCE(qualification_status, 0) = sqlc.narg('qualification_status')::smallint);

-- name: ListAdvertisers :many
SELECT id, name, industry, contact_name, contact_email, balance, status,
       COALESCE(qualification_status, 0) as qualification_status,
       qualification_reason, COALESCE(credit_limit, 0) as credit_limit,
       address, website, brand_names, created_at, updated_at
FROM advertiser
WHERE (sqlc.narg('status')::smallint IS NULL OR status = sqlc.narg('status')::smallint)
  AND (sqlc.narg('qualification_status')::smallint IS NULL OR COALESCE(qualification_status, 0) = sqlc.narg('qualification_status')::smallint)
ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: UpdateAdvertiser :exec
UPDATE advertiser SET
  name = COALESCE($2, name),
  industry = COALESCE($3, industry),
  contact_name = COALESCE($4, contact_name),
  contact_email = COALESCE($5, contact_email),
  address = COALESCE($6, address),
  website = COALESCE($7, website),
  brand_names = COALESCE($8, brand_names),
  updated_at = NOW()
WHERE id = $1;

-- name: UpdateAdvertiserQualification :exec
UPDATE advertiser SET qualification_status = $2, qualification_reason = $3, updated_at = NOW() WHERE id = $1;

-- name: DeleteAdvertiser :exec
DELETE FROM advertiser WHERE id = $1;

-- Proof material queries
-- name: CreateProofMaterial :exec
INSERT INTO proof_material (advertiser_id, material_type, file_url, file_name, file_size)
VALUES ($1, $2, $3, $4, $5);

-- name: ListProofMaterials :many
SELECT id, advertiser_id, material_type, file_url, file_name, file_size, audit_status, audit_reason, created_at
FROM proof_material WHERE advertiser_id = $1 ORDER BY created_at DESC;

-- Balance queries
-- name: GetAdvertiserBalance :one
SELECT COALESCE(balance, 0) as balance, COALESCE(credit_limit, 0) as credit_limit FROM advertiser WHERE id = $1;

-- name: RechargeAdvertiser :one
UPDATE advertiser SET balance = balance + $2, updated_at = NOW() WHERE id = $1
RETURNING balance;

-- name: CreateBalanceTransaction :exec
INSERT INTO balance_transaction (advertiser_id, amount, balance_before, balance_after, tx_type, description, operator_id)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: CountBalanceTransactions :one
SELECT COUNT(*) FROM balance_transaction WHERE advertiser_id = $1;

-- name: ListBalanceTransactions :many
SELECT id, advertiser_id, amount, balance_before, balance_after, tx_type, description, operator_id, created_at
FROM balance_transaction WHERE advertiser_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- Media queries
-- name: CreateMedia :one
INSERT INTO media (name, code, domain) VALUES ($1, $2, $3) RETURNING id;

-- name: UpdateMedia :exec
UPDATE media SET name = COALESCE($2, name), domain = COALESCE($3, domain) WHERE id = $1;

-- name: UpdateMediaStatus :exec
UPDATE media SET status = $2 WHERE id = $1;

-- Ad position queries
-- name: CreateAdPosition :one
INSERT INTO ad_position (media_id, name, position_type, ad_format, width, height, max_size, duration_min, duration_max, mime_types)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id;

-- name: UpdateAdPosition :exec
UPDATE ad_position SET
  name = COALESCE($2, name),
  width = COALESCE($3, width),
  height = COALESCE($4, height),
  max_size = COALESCE($5, max_size),
  duration_min = COALESCE($6, duration_min),
  duration_max = COALESCE($7, duration_max)
WHERE id = $1;

-- Admin queries
-- name: CountUsers :one
SELECT COUNT(*) FROM users
WHERE (sqlc.narg('role')::varchar IS NULL OR role = sqlc.narg('role')::varchar);

-- name: ListUsers :many
SELECT id, email, name, advertiser_id, role, created_at FROM users
WHERE (sqlc.narg('role')::varchar IS NULL OR role = sqlc.narg('role')::varchar)
ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: UpdateUserRole :exec
UPDATE users SET role = $2 WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (email, password_hash, name, advertiser_id, role)
VALUES ($1, $2, $3, $4, $5) RETURNING id;

-- name: ListAllMedia :many
SELECT id, name, code, domain, status FROM media ORDER BY id;

-- name: ListAdPositionsByMedia :many
SELECT id, media_id, name, position_type, ad_format, width, height, max_size, duration_min, duration_max, mime_types, status
FROM ad_position WHERE media_id = $1 ORDER BY id;

-- name: CountPendingAudits :one
SELECT COUNT(*) FROM (
  SELECT id FROM creative WHERE audit_status = 0
  UNION ALL
  SELECT id FROM advertiser WHERE COALESCE(qualification_status, 0) = 0
) t;

-- name: ListPendingCreativeAudits :many
SELECT c.id, 1 as audit_type, c.name, ca.advertiser_id,
       COALESCE(a.name, '') as advertiser_name,
       c.audit_status as status, c.audit_reason as reason, c.created_at
FROM creative c
JOIN ad_group ag ON c.ad_group_id = ag.id
JOIN campaign ca ON ag.campaign_id = ca.id
JOIN advertiser a ON ca.advertiser_id = a.id
WHERE c.audit_status = 0
ORDER BY c.created_at ASC LIMIT $1 OFFSET $2;

-- name: ListPendingAdvertiserAudits :many
SELECT id, 2 as audit_type, name, id as advertiser_id, name as advertiser_name,
       COALESCE(qualification_status, 0) as status, qualification_reason as reason, created_at
FROM advertiser
WHERE COALESCE(qualification_status, 0) = 0
ORDER BY created_at ASC LIMIT $1 OFFSET $2;
