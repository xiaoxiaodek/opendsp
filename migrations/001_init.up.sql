CREATE TABLE advertiser (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    industry        VARCHAR(64),
    contact_name    VARCHAR(64),
    contact_email   VARCHAR(128),
    balance         DECIMAL(14,2) DEFAULT 0,
    status          SMALLINT DEFAULT 1,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
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
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

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
    event_time      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ DEFAULT NOW()
) PARTITION BY RANGE (event_time);

CREATE TABLE stat_event_202506 PARTITION OF stat_event
    FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');

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
    cost            DECIMAL(14,4) DEFAULT 0,
    win_count       BIGINT DEFAULT 0,
    bid_count       BIGINT DEFAULT 0,
    UNIQUE(hour, ad_group_id, creative_id, media_id, ad_position_id)
);

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
