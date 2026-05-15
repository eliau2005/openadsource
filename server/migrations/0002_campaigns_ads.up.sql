-- 0002_campaigns_ads.up.sql
-- Phase 1 schema. Replaces the spec's single `video_url` column on `ads`
-- with a storage-agnostic media_source / media_url / media_mime trio so
-- creatives can be hosted on any public CDN (`external_url`) or in an
-- S3-compatible bucket (`internal_s3`). See ROADMAP §Phase 1.

CREATE TABLE advertisers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active','archived')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE campaigns (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    advertiser_id            UUID NOT NULL REFERENCES advertisers(id) ON DELETE CASCADE,
    name                     TEXT NOT NULL,
    start_date               TIMESTAMPTZ,
    end_date                 TIMESTAMPTZ,
    total_budget_impressions INTEGER,
    status                   TEXT NOT NULL DEFAULT 'active'
                                 CHECK (status IN ('active','paused','completed','archived')),
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_campaigns_advertiser_id ON campaigns(advertiser_id);
CREATE INDEX idx_campaigns_status        ON campaigns(status);

CREATE TABLE ads (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id         UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active','paused','archived')),
    position_type       TEXT NOT NULL DEFAULT 'pre'
                            CHECK (position_type IN ('pre','mid','post')),
    mid_roll_offset     INTEGER,
    priority            INTEGER NOT NULL DEFAULT 1,
    landing_page_url    TEXT,
    -- BYO-URL extension ------------------------------------------------------
    media_source        TEXT NOT NULL DEFAULT 'external_url'
                            CHECK (media_source IN ('external_url','internal_s3')),
    media_url           TEXT NOT NULL,
    media_mime          TEXT NOT NULL,
    media_duration_ms   INTEGER,
    media_width         INTEGER,
    media_height        INTEGER,
    media_bitrate_kbps  INTEGER,
    -- audit ------------------------------------------------------------------
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ads_campaign_id      ON ads(campaign_id);
CREATE INDEX idx_ads_status_position  ON ads(status, position_type);

CREATE TABLE cap_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ad_id               UUID NOT NULL REFERENCES ads(id) ON DELETE CASCADE,
    max_impressions     INTEGER NOT NULL,
    time_window_seconds INTEGER NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cap_rules_ad_id ON cap_rules(ad_id);
