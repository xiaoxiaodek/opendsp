-- Fraud events queries

-- name: InsertFraudEvent :exec
INSERT INTO fraud_events (request_id, rule_type, rule_value, risk_score, action, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: CountFraudEvents :one
SELECT COUNT(*) FROM fraud_events
WHERE created_at >= $1 AND created_at <= $2;

-- name: ListFraudEvents :many
SELECT id, request_id, rule_type, rule_value, COALESCE(risk_score, 0) as risk_score, action, created_at
FROM fraud_events
WHERE created_at >= $1 AND created_at <= $2
ORDER BY created_at DESC LIMIT $3 OFFSET $4;

-- name: GetFraudEventStats :one
SELECT
    COUNT(*)::bigint as total,
    COUNT(*) FILTER (WHERE action = 'blocked')::bigint as blocked,
    COUNT(*) FILTER (WHERE action = 'flagged')::bigint as flagged
FROM fraud_events
WHERE created_at >= $1 AND created_at <= $2;
