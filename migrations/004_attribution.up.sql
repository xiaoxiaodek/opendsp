-- Attribution: click_id for tracking, conversion_event for postback
ALTER TABLE stat_event ADD COLUMN IF NOT EXISTS click_id VARCHAR(64);

CREATE TABLE IF NOT EXISTS conversion_event (
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

CREATE INDEX IF NOT EXISTS idx_conversion_click_id ON conversion_event(click_id);
CREATE INDEX IF NOT EXISTS idx_conversion_advertiser ON conversion_event(advertiser_id, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_conversion_event_time ON conversion_event(event_time);

ALTER TABLE report_hourly ADD COLUMN IF NOT EXISTS conversions BIGINT DEFAULT 0;
ALTER TABLE report_hourly ADD COLUMN IF NOT EXISTS revenue DECIMAL(14,4) DEFAULT 0;
