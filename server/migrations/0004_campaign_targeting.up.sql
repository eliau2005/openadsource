-- 0004_campaign_targeting.up.sql
-- Per-campaign targeting allowlists consumed by the Phase 3 decision engine.
-- Convention: NULL = "all" (no filter); a non-NULL array intersects against
-- the request's country / device. The arrays are case-sensitive at the SQL
-- layer; the loader uppercases countries and lowercases devices into the
-- snapshot, so writers should normalise too.

CREATE TABLE campaign_targeting (
    campaign_id UUID PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    countries   TEXT[],
    devices     TEXT[],
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_campaign_targeting_countries ON campaign_targeting USING GIN (countries);
CREATE INDEX idx_campaign_targeting_devices   ON campaign_targeting USING GIN (devices);
