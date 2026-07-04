-- ============================================================================
-- OpenDSP: Merged initial schema
-- Combines migrations 001-011 into a single init script
-- ============================================================================

-- ---------------------------------------------------------------------------
-- 1. Core tables (original 001_init)
-- ---------------------------------------------------------------------------
CREATE TABLE advertiser (
    id                     BIGSERIAL PRIMARY KEY,
    name                   VARCHAR(128) NOT NULL,
    industry               VARCHAR(64),
    contact_name           VARCHAR(64),
    contact_email          VARCHAR(128),
    balance                DECIMAL(14,2) DEFAULT 0,
    status                 SMALLINT DEFAULT 1,
    -- 002_advertiser additions
    qualification_status   SMALLINT DEFAULT 0,
    qualification_reason   VARCHAR(500),
    credit_limit           DECIMAL(14,2) DEFAULT 0,
    address                VARCHAR(256),
    website                VARCHAR(256),
    brand_names            VARCHAR(512),
    created_at             TIMESTAMPTZ DEFAULT NOW(),
    updated_at             TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    email           VARCHAR(128) NOT NULL UNIQUE,
    password_hash   VARCHAR(256) NOT NULL,
    name            VARCHAR(80),
    advertiser_id   BIGINT REFERENCES advertiser(id),
    role            VARCHAR(20) DEFAULT 'viewer',
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE campaign (
    id              BIGSERIAL PRIMARY KEY,
    advertiser_id   BIGINT NOT NULL REFERENCES advertiser(id),
    name            VARCHAR(200) NOT NULL,
    budget          DECIMAL(14,2),
    daily_budget    DECIMAL(14,2),
    start_time      TIMESTAMPTZ,
    end_time        TIMESTAMPTZ,
    pacing          SMALLINT DEFAULT 1,
    status          SMALLINT DEFAULT 1,
    version         BIGINT DEFAULT 1,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE ad_group (
    id              BIGSERIAL PRIMARY KEY,
    campaign_id     BIGINT NOT NULL REFERENCES campaign(id),
    name            VARCHAR(200) NOT NULL,
    bid_type        SMALLINT NOT NULL,
    bid_price       DECIMAL(10,4) NOT NULL,
    daily_budget    DECIMAL(14,2),
    freq_cap        INTEGER,
    freq_period     INTEGER DEFAULT 24,
    targeting       JSONB NOT NULL DEFAULT '{}',
    status          SMALLINT DEFAULT 1,
    version         BIGINT DEFAULT 1,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE creative (
    id              BIGSERIAL PRIMARY KEY,
    ad_group_id     BIGINT NOT NULL REFERENCES ad_group(id),
    name            VARCHAR(200) NOT NULL,
    creative_type   SMALLINT NOT NULL,
    asset_url       VARCHAR(2048) NOT NULL,
    asset_size      INTEGER,
    asset_duration  INTEGER DEFAULT 0,
    asset_width     INTEGER,
    asset_height    INTEGER,
    asset_mime      VARCHAR(64),
    title           VARCHAR(100),
    description     VARCHAR(500),
    cta_text        VARCHAR(50),
    brand_name      VARCHAR(100),
    brand_logo      VARCHAR(2048),
    landing_url     VARCHAR(2048) NOT NULL,
    deeplink_url    VARCHAR(2048),
    imp_tracker     VARCHAR(2048),
    click_tracker   VARCHAR(2048),
    third_party_trackers JSONB DEFAULT '[]',
    audit_status    SMALLINT DEFAULT 0,
    audit_reason    VARCHAR(500),
    is_valid        SMALLINT DEFAULT 1,
    version         BIGINT DEFAULT 1,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE media (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    code            VARCHAR(32) NOT NULL UNIQUE,
    domain          VARCHAR(256),
    status          SMALLINT DEFAULT 1,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE ad_position (
    id              BIGSERIAL PRIMARY KEY,
    media_id        BIGINT NOT NULL REFERENCES media(id),
    name            VARCHAR(128) NOT NULL,
    position_type   SMALLINT NOT NULL,
    ad_format       SMALLINT NOT NULL,
    width           INTEGER,
    height          INTEGER,
    max_size        INTEGER,
    duration_min    INTEGER DEFAULT 0,
    duration_max    INTEGER DEFAULT 0,
    mime_types      VARCHAR(256),
    status          SMALLINT DEFAULT 1,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE ad_position_price (
    id              BIGSERIAL PRIMARY KEY,
    ad_position_id  BIGINT NOT NULL REFERENCES ad_position(id),
    charge_type     SMALLINT NOT NULL,
    floor_price     DECIMAL(10,4) NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE content_target (
    id              BIGSERIAL PRIMARY KEY,
    media_id        BIGINT NOT NULL REFERENCES media(id),
    content_id      VARCHAR(128) NOT NULL,
    content_title   VARCHAR(256),
    content_url     VARCHAR(2048),
    content_type    SMALLINT DEFAULT 1,
    category        VARCHAR(64),
    tags            VARCHAR(512),
    duration        INTEGER,
    start_time      INTEGER,
    slot_duration   INTEGER,
    slot_id         VARCHAR(128),
    ext             JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(media_id, content_id, slot_id)
);

CREATE TABLE dmp_audience (
    id              BIGSERIAL PRIMARY KEY,
    advertiser_id   BIGINT NOT NULL REFERENCES advertiser(id),
    name            VARCHAR(128) NOT NULL,
    audience_type   SMALLINT DEFAULT 1,
    device_count    BIGINT DEFAULT 0,
    rules           JSONB DEFAULT '{}',
    -- 007_dmp additions
    status          SMALLINT DEFAULT 1,
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- 2. Advertiser finance (002_advertiser)
-- ---------------------------------------------------------------------------
CREATE TABLE proof_material (
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

CREATE TABLE balance_transaction (
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

CREATE INDEX idx_proof_material_advertiser ON proof_material(advertiser_id);
CREATE INDEX idx_balance_tx_advertiser ON balance_transaction(advertiser_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- 3. Stat events & reports (001_init + 004_attribution)
-- ---------------------------------------------------------------------------
CREATE TABLE stat_event (
    id              BIGSERIAL,
    event_type      SMALLINT NOT NULL,
    adgroup_id      BIGINT NOT NULL,
    creative_id     BIGINT NOT NULL,
    campaign_id     BIGINT NOT NULL,
    advertiser_id   BIGINT NOT NULL,
    media_id        BIGINT NOT NULL,
    ad_position_id  BIGINT NOT NULL,
    price           DECIMAL(10,4) DEFAULT 0,
    charge_type     SMALLINT,
    device_id       VARCHAR(128),
    ip              VARCHAR(45),
    ua              VARCHAR(512),
    geo_city        VARCHAR(20),
    freq_result     VARCHAR(32),
    click_id        VARCHAR(64),
    event_time      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ DEFAULT NOW()
) PARTITION BY RANGE (event_time);

CREATE TABLE stat_event_202506 PARTITION OF stat_event
    FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');

CREATE TABLE conversion_event (
    id              BIGSERIAL PRIMARY KEY,
    click_id        VARCHAR(64) NOT NULL,
    event_type      VARCHAR(32) NOT NULL DEFAULT 'install',
    adgroup_id      BIGINT,
    creative_id     BIGINT,
    campaign_id     BIGINT,
    advertiser_id   BIGINT,
    media_id        BIGINT,
    ad_position_id  BIGINT,
    price           DECIMAL(10,4) DEFAULT 0,
    revenue         DECIMAL(14,4) DEFAULT 0,
    device_id       VARCHAR(128),
    ip              VARCHAR(45),
    ua              VARCHAR(512),
    geo_city        VARCHAR(20),
    extra           JSONB DEFAULT '{}',
    event_time      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_conversion_click_id ON conversion_event(click_id);
CREATE INDEX idx_conversion_advertiser ON conversion_event(advertiser_id, event_time DESC);
CREATE INDEX idx_conversion_event_time ON conversion_event(event_time);

CREATE TABLE report_hourly (
    id              BIGSERIAL PRIMARY KEY,
    hour            TIMESTAMPTZ NOT NULL,
    advertiser_id   BIGINT NOT NULL,
    campaign_id     BIGINT NOT NULL,
    ad_group_id     BIGINT NOT NULL,
    creative_id     BIGINT NOT NULL,
    media_id        BIGINT NOT NULL,
    ad_position_id  BIGINT NOT NULL,
    impressions     BIGINT DEFAULT 0,
    clicks          BIGINT DEFAULT 0,
    conversions     BIGINT DEFAULT 0,
    revenue         DECIMAL(14,4) DEFAULT 0,
    cost            DECIMAL(14,4) DEFAULT 0,
    win_count       BIGINT DEFAULT 0,
    bid_count       BIGINT DEFAULT 0,
    UNIQUE(hour, ad_group_id, creative_id, media_id, ad_position_id)
);

-- ---------------------------------------------------------------------------
-- 4. File gateway (005_file_gateway)
-- ---------------------------------------------------------------------------
CREATE TABLE file_record (
    id           VARCHAR(32) PRIMARY KEY,
    namespace    VARCHAR(16) NOT NULL,
    storage_key  VARCHAR(512) NOT NULL,
    filename     VARCHAR(256),
    size         BIGINT DEFAULT 0,
    content_type VARCHAR(128),
    md5          VARCHAR(64),
    status       SMALLINT DEFAULT 1,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_file_record_status ON file_record(status, created_at);

-- ---------------------------------------------------------------------------
-- 5. Stat event maintenance functions (006_stat_event_cleanup)
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION ensure_stat_event_partition(target_month TIMESTAMPTZ)
RETURNS VOID AS $$
DECLARE
    partition_name TEXT;
    start_date TIMESTAMPTZ;
    end_date TIMESTAMPTZ;
BEGIN
    start_date := date_trunc('month', target_month);
    end_date := start_date + INTERVAL '1 month';
    partition_name := 'stat_event_' || to_char(start_date, 'YYYYMM');

    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF stat_event FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );

    EXECUTE format(
        'CREATE INDEX IF NOT EXISTS %I ON %I (event_type, event_time)',
        'idx_' || partition_name || '_type_time', partition_name
    );
END;
$$ LANGUAGE plpgsql;

SELECT ensure_stat_event_partition(NOW());
SELECT ensure_stat_event_partition(NOW() + INTERVAL '1 month');
SELECT ensure_stat_event_partition(NOW() + INTERVAL '2 months');

CREATE OR REPLACE FUNCTION cleanup_stat_event_partitions(retention_days INTEGER DEFAULT 90)
RETURNS SETOF TEXT AS $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
          AND tablename ~ '^stat_event_\d{6}$'
          AND tablename < 'stat_event_' || to_char(NOW() - (retention_days || ' days')::INTERVAL, 'YYYYMM')
    LOOP
        EXECUTE format('DROP TABLE IF EXISTS %I', rec.tablename);
        RETURN NEXT 'dropped: ' || rec.tablename;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION delete_aggregated_events(before_time TIMESTAMPTZ)
RETURNS BIGINT AS $$
DECLARE
    deleted BIGINT;
BEGIN
    DELETE FROM stat_event WHERE event_time < before_time;
    GET DIAGNOSTICS deleted = ROW_COUNT;
    RETURN deleted;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- 6. DMP (007_dmp)
-- ---------------------------------------------------------------------------
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
CREATE INDEX idx_dmp_audience_advertiser ON dmp_audience(advertiser_id, status);

CREATE TABLE dmp_device (
    device_id       VARCHAR(128) NOT NULL,
    device_type     VARCHAR(16) NOT NULL,
    tag_ids         BIGINT[] DEFAULT '{}',
    first_seen      TIMESTAMPTZ DEFAULT NOW(),
    last_seen       TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (device_id, device_type)
);

-- ---------------------------------------------------------------------------
-- 7. Platform sync (008_platform_sync)
-- ---------------------------------------------------------------------------
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

-- ---------------------------------------------------------------------------
-- 8. Anti-fraud (009_antifraud)
-- ---------------------------------------------------------------------------
CREATE TABLE fraud_blacklist (
    id         BIGSERIAL PRIMARY KEY,
    rule_type  VARCHAR(32)  NOT NULL,
    rule_value VARCHAR(512) NOT NULL,
    reason     VARCHAR(256),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(rule_type, rule_value)
);

CREATE TABLE fraud_events (
    id          BIGSERIAL PRIMARY KEY,
    request_id  VARCHAR(64)  NOT NULL,
    rule_type   VARCHAR(32)  NOT NULL,
    rule_value  VARCHAR(512) NOT NULL,
    risk_score  NUMERIC(5,4),
    action      VARCHAR(16)  NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_fraud_events_created ON fraud_events(created_at);
CREATE INDEX idx_fraud_events_request ON fraud_events(request_id);

-- ---------------------------------------------------------------------------
-- 9. Ledger (010_ledger)
-- ---------------------------------------------------------------------------
CREATE TABLE ledger_accounts (
    id             BIGSERIAL PRIMARY KEY,
    advertiser_id  BIGINT NOT NULL,
    balance_micros BIGINT NOT NULL DEFAULT 0,
    frozen_micros  BIGINT NOT NULL DEFAULT 0,
    spent_micros   BIGINT NOT NULL DEFAULT 0,
    version        INT NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_ledger_accounts_advertiser ON ledger_accounts(advertiser_id);

CREATE TABLE ledger_transactions (
    id             BIGSERIAL PRIMARY KEY,
    account_id     BIGINT NOT NULL REFERENCES ledger_accounts(id),
    type           VARCHAR(16) NOT NULL,
    amount_micros  BIGINT NOT NULL,
    balance_after  BIGINT NOT NULL,
    reference_id   VARCHAR(128),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ledger_txns_account ON ledger_transactions(account_id, created_at);
CREATE INDEX idx_ledger_txns_ref ON ledger_transactions(reference_id);

-- ---------------------------------------------------------------------------
-- 10. ROI (011_roi)
-- ---------------------------------------------------------------------------
CREATE TABLE roi_metrics (
    id             BIGSERIAL PRIMARY KEY,
    advertiser_id  BIGINT NOT NULL,
    campaign_id    BIGINT,
    adgroup_id     BIGINT,
    date           DATE NOT NULL,
    cost_micros    BIGINT NOT NULL DEFAULT 0,
    revenue_micros BIGINT NOT NULL DEFAULT 0,
    conversions    INT NOT NULL DEFAULT 0,
    roas           NUMERIC(10,4),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(advertiser_id, campaign_id, adgroup_id, date)
);

CREATE INDEX idx_roi_metrics_advertiser_date ON roi_metrics(advertiser_id, date);

-- ---------------------------------------------------------------------------
-- 11. Seed data
-- ---------------------------------------------------------------------------
INSERT INTO media (name, code, domain) VALUES
    ('爱奇艺', 'iqiyi', 'iqiyi.com'),
    ('优酷', 'youku', 'youku.com'),
    ('芒果TV', 'mgtv', 'mgtv.com'),
    ('风行', 'funadx', 'fun.tv');

INSERT INTO ad_position (media_id, name, position_type, ad_format, width, height, max_size, duration_min, duration_max, mime_types)
SELECT id, '前贴片', 1, 2, 1920, 1080, 102400, 5, 90, 'video/mp4,video/flv' FROM media WHERE code = 'iqiyi'
UNION ALL SELECT id, '中插', 2, 2, 1920, 1080, 102400, 5, 90, 'video/mp4,video/flv' FROM media WHERE code = 'iqiyi'
UNION ALL SELECT id, '后贴片', 3, 2, 1920, 1080, 102400, 5, 90, 'video/mp4,video/flv' FROM media WHERE code = 'iqiyi'
UNION ALL SELECT id, '暂停', 4, 1, 1920, 1080, 2048, 0, 0, 'image/jpeg,image/png' FROM media WHERE code = 'iqiyi'
UNION ALL SELECT id, '浮层', 5, 1, 600, 400, 1024, 0, 0, 'image/jpeg,image/png' FROM media WHERE code = 'iqiyi';

INSERT INTO advertiser (name, industry) VALUES ('默认广告主', 'other');

-- Seed admin user (password: admin123)
INSERT INTO users (email, password_hash, name, role)
VALUES ('admin@opendsp.io', '240be518fabd2724ddb6f04eeb1da5967448d7e831c08c8fa822809f74c720a9', 'Super Admin', 'admin')
ON CONFLICT (email) DO NOTHING;
