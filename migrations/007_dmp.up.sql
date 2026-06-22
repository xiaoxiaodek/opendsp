CREATE TABLE dmp_tag (
    id              BIGSERIAL PRIMARY KEY,
    advertiser_id   BIGINT NOT NULL REFERENCES advertiser(id),
    name            VARCHAR(128) NOT NULL,
    tag_type        SMALLINT NOT NULL DEFAULT 1,
    device_count    BIGINT DEFAULT 0,
    source          VARCHAR(64),
    source_config   JSONB DEFAULT '{}',
    status          SMALLINT DEFAULT 1,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_dmp_tag_advertiser ON dmp_tag(advertiser_id, tag_type);

ALTER TABLE dmp_audience ADD COLUMN IF NOT EXISTS status SMALLINT DEFAULT 1;
ALTER TABLE dmp_audience ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_dmp_audience_advertiser ON dmp_audience(advertiser_id, status);

CREATE TABLE dmp_device (
    device_id       VARCHAR(128) NOT NULL,
    device_type     VARCHAR(16) NOT NULL,
    tag_ids         BIGINT[] DEFAULT '{}',
    first_seen      TIMESTAMPTZ DEFAULT NOW(),
    last_seen       TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (device_id, device_type)
);
