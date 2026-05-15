-- 0005_daily_stats.up.sql
-- Per-(ad, day) counters drained from Redis by the worker every 30s. The
-- dashboard reads from this table for the /reports/[campaignId] funnel +
-- daily bars. Inserts use ON CONFLICT (ad_id, date) DO UPDATE so each
-- worker tick contributes a delta cleanly.

CREATE TABLE daily_stats (
    id              UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id     UUID    NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    ad_id           UUID    NOT NULL REFERENCES ads(id)       ON DELETE CASCADE,
    date            DATE    NOT NULL,
    impressions     INTEGER NOT NULL DEFAULT 0,
    clicks          INTEGER NOT NULL DEFAULT 0,
    start_count     INTEGER NOT NULL DEFAULT 0,
    q25             INTEGER NOT NULL DEFAULT 0,
    q50             INTEGER NOT NULL DEFAULT 0,
    q75             INTEGER NOT NULL DEFAULT 0,
    complete        INTEGER NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (ad_id, date)
);

CREATE INDEX idx_daily_stats_campaign_date ON daily_stats(campaign_id, date DESC);
