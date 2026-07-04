CREATE TABLE IF NOT EXISTS ledger_accounts (
    id             BIGSERIAL PRIMARY KEY,
    advertiser_id  BIGINT NOT NULL,
    balance_micros BIGINT NOT NULL DEFAULT 0,
    frozen_micros  BIGINT NOT NULL DEFAULT 0,
    spent_micros   BIGINT NOT NULL DEFAULT 0,
    version        INT NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ledger_accounts_advertiser ON ledger_accounts(advertiser_id);

CREATE TABLE IF NOT EXISTS ledger_transactions (
    id             BIGSERIAL PRIMARY KEY,
    account_id     BIGINT NOT NULL REFERENCES ledger_accounts(id),
    type           VARCHAR(16) NOT NULL,
    amount_micros  BIGINT NOT NULL,
    balance_after  BIGINT NOT NULL,
    reference_id   VARCHAR(128),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ledger_txns_account ON ledger_transactions(account_id, created_at);
CREATE INDEX IF NOT EXISTS idx_ledger_txns_ref ON ledger_transactions(reference_id);
