-- Advertiser qualification & finance
ALTER TABLE advertiser ADD COLUMN IF NOT EXISTS qualification_status SMALLINT DEFAULT 0;
ALTER TABLE advertiser ADD COLUMN IF NOT EXISTS qualification_reason VARCHAR(500);
ALTER TABLE advertiser ADD COLUMN IF NOT EXISTS credit_limit DECIMAL(14,2) DEFAULT 0;
ALTER TABLE advertiser ADD COLUMN IF NOT EXISTS address VARCHAR(256);
ALTER TABLE advertiser ADD COLUMN IF NOT EXISTS website VARCHAR(256);
ALTER TABLE advertiser ADD COLUMN IF NOT EXISTS brand_names VARCHAR(512);

CREATE TABLE IF NOT EXISTS proof_material (
    id              BIGSERIAL PRIMARY KEY,
    advertiser_id   BIGINT NOT NULL REFERENCES advertiser(id),
    material_type   SMALLINT NOT NULL,
    file_url        VARCHAR(2048) NOT NULL,
    file_name       VARCHAR(256),
    file_size       INTEGER,
    audit_status    SMALLINT DEFAULT 0,
    audit_reason    VARCHAR(500),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS balance_transaction (
    id              BIGSERIAL PRIMARY KEY,
    advertiser_id   BIGINT NOT NULL REFERENCES advertiser(id),
    amount          DECIMAL(14,2) NOT NULL,
    balance_before  DECIMAL(14,2) NOT NULL,
    balance_after   DECIMAL(14,2) NOT NULL,
    tx_type         SMALLINT NOT NULL,
    description     VARCHAR(500),
    operator_id     BIGINT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_proof_material_advertiser ON proof_material(advertiser_id);
CREATE INDEX IF NOT EXISTS idx_balance_tx_advertiser ON balance_transaction(advertiser_id, created_at DESC);
