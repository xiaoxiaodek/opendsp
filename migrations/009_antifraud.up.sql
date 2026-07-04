CREATE TABLE IF NOT EXISTS fraud_blacklist (
    id         BIGSERIAL PRIMARY KEY,
    rule_type  VARCHAR(32)  NOT NULL,
    rule_value VARCHAR(512) NOT NULL,
    reason     VARCHAR(256),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(rule_type, rule_value)
);

CREATE TABLE IF NOT EXISTS fraud_events (
    id          BIGSERIAL PRIMARY KEY,
    request_id  VARCHAR(64)  NOT NULL,
    rule_type   VARCHAR(32)  NOT NULL,
    rule_value  VARCHAR(512) NOT NULL,
    risk_score  NUMERIC(5,4),
    action      VARCHAR(16)  NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_fraud_events_created ON fraud_events(created_at);
CREATE INDEX IF NOT EXISTS idx_fraud_events_request ON fraud_events(request_id);
