-- name: GetAdByID :one
-- Joins the ad with its campaign so callers can decide eligibility (campaign
-- status, end_date) in a single round trip. Phase 3 will replace per-request
-- DB reads with the in-memory snapshot loader.
SELECT
    a.id,
    a.campaign_id,
    a.name,
    a.status,
    a.position_type,
    a.mid_roll_offset,
    a.priority,
    a.landing_page_url,
    a.media_source,
    a.media_url,
    a.media_mime,
    a.media_duration_ms,
    a.media_width,
    a.media_height,
    a.media_bitrate_kbps,
    a.created_at,
    a.updated_at,
    c.status    AS campaign_status,
    c.end_date  AS campaign_end_date
FROM ads a
JOIN campaigns c ON c.id = a.campaign_id
WHERE a.id = $1;
