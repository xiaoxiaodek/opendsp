CREATE TABLE IF NOT EXISTS roi_metrics (
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

CREATE INDEX IF NOT EXISTS idx_roi_metrics_advertiser_date ON roi_metrics(advertiser_id, date);
