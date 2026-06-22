CREATE TABLE creative_sync (
    id              BIGSERIAL PRIMARY KEY,
    creative_id     BIGINT NOT NULL REFERENCES creative(id) ON DELETE CASCADE,
    platform        TEXT NOT NULL,
    status          SMALLINT NOT NULL DEFAULT 0,
    external_id     TEXT,
    external_tvid   TEXT,
    reason          TEXT,
    raw_response    JSONB,
    synced_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(creative_id, platform)
);

CREATE INDEX idx_creative_sync_status ON creative_sync(platform, status);

CREATE TABLE advertiser_sync (
    id              BIGSERIAL PRIMARY KEY,
    advertiser_id   BIGINT NOT NULL REFERENCES advertiser(id) ON DELETE CASCADE,
    platform        TEXT NOT NULL,
    status          SMALLINT NOT NULL DEFAULT 0,
    external_ad_id  TEXT,
    reason          TEXT,
    raw_response    JSONB,
    synced_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(advertiser_id, platform)
);

CREATE INDEX idx_advertiser_sync_status ON advertiser_sync(platform, status);

COMMENT ON TABLE creative_sync IS 'Multi-platform creative synchronization status';
COMMENT ON TABLE advertiser_sync IS 'Multi-platform advertiser synchronization status';
