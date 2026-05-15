// Command seed populates the database with one advertiser, one campaign, and
// two ads so a fresh stack can serve an end-to-end VAST response and play it
// in examples/test-player/. Stable UUIDs + ON CONFLICT DO NOTHING make it
// safe to re-run.
//
// The two seeded ads cover both BYO-URL paths:
//
//   - "internal_s3" ad: the seed downloads SAMPLE_MP4_URL into memory and
//     uploads it to MinIO at key "seed/sample.mp4". The resolver serves it
//     via S3_PUBLIC_BASE_URL (or a presigned URL when that isn't set).
//   - "external_url" ad: media_url is set to SAMPLE_MP4_URL directly. The
//     resolver passes it straight through, exercising the no-bytes-through-us
//     path even when no S3 is configured.
//
// If SAMPLE_MP4_URL is unreachable or S3 isn't configured, the internal_s3
// ad insert is skipped (logged) and the script still exits 0. The
// external_url ad always lands so the test player has at least one ad to
// fetch.
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/eliau2005/openadsource/server/internal/config"
	"github.com/eliau2005/openadsource/server/internal/db"
	"github.com/eliau2005/openadsource/server/internal/storage"
)

const (
	seedAdvertiserID = "00000000-0000-0000-0000-000000000001"
	seedCampaignID   = "00000000-0000-0000-0000-000000000002"
	seedAdInternalID = "00000000-0000-0000-0000-000000000003"
	seedAdExternalID = "00000000-0000-0000-0000-000000000004"

	defaultSampleURL = "https://www.w3schools.com/html/mov_bbb.mp4"
	seedObjectKey    = "seed/sample.mp4"
)

func main() {
	cfg := config.Load()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if lvl, err := zerolog.ParseLevel(cfg.LogLevel); err == nil {
		zerolog.SetGlobalLevel(lvl)
	}

	sampleURL := os.Getenv("SAMPLE_MP4_URL")
	if sampleURL == "" {
		sampleURL = defaultSampleURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("db pool init failed")
	}
	defer pool.Close()
	if err := db.PingWithRetry(ctx, pool, 15, 2*time.Second); err != nil {
		log.Fatal().Err(err).Msg("postgres unreachable")
	}

	if err := insertAdvertiser(ctx, pool); err != nil {
		log.Fatal().Err(err).Msg("insert advertiser failed")
	}
	if err := insertCampaign(ctx, pool); err != nil {
		log.Fatal().Err(err).Msg("insert campaign failed")
	}

	// Best-effort internal_s3 ad. Any failure (S3 unconfigured, download
	// fails, upload fails) is logged and the seed continues with the
	// external_url ad below.
	internalOK := seedInternalAd(ctx, cfg, pool, sampleURL)

	if err := insertExternalAd(ctx, pool, sampleURL); err != nil {
		log.Fatal().Err(err).Msg("insert external_url ad failed")
	}

	log.Info().
		Str("advertiser_id", seedAdvertiserID).
		Str("campaign_id", seedCampaignID).
		Str("ad_external_id", seedAdExternalID).
		Bool("internal_s3_seeded", internalOK).
		Msg("seed complete")
}

func seedInternalAd(ctx context.Context, cfg config.Config, pool *pgxpool.Pool, sampleURL string) bool {
	if !cfg.S3Configured() {
		log.Info().Msg("S3 not configured; skipping internal_s3 ad")
		return false
	}
	s3Client, err := storage.NewS3Client(ctx, cfg)
	if err != nil || s3Client == nil {
		log.Warn().Err(err).Msg("s3 client init failed; skipping internal_s3 ad")
		return false
	}

	body, err := downloadSample(ctx, sampleURL)
	if err != nil {
		log.Warn().Str("url", sampleURL).Err(err).Msg("sample download failed; skipping internal_s3 ad")
		return false
	}
	if err := s3Client.PutObject(ctx, seedObjectKey, bytes.NewReader(body), "video/mp4"); err != nil {
		log.Warn().Err(err).Msg("s3 PutObject failed; skipping internal_s3 ad")
		return false
	}
	log.Info().Str("key", seedObjectKey).Int("bytes", len(body)).Msg("sample uploaded to s3")

	if err := insertInternalAd(ctx, pool); err != nil {
		log.Warn().Err(err).Msg("insert internal_s3 ad failed")
		return false
	}
	return true
}

func downloadSample(ctx context.Context, url string) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func insertAdvertiser(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
INSERT INTO advertisers (id, name, status)
VALUES ($1::uuid, $2, 'active')
ON CONFLICT (id) DO NOTHING`,
		seedAdvertiserID, "OpenAdSource Demo Advertiser")
	return err
}

func insertCampaign(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
INSERT INTO campaigns (id, advertiser_id, name, start_date, end_date,
                       total_budget_impressions, status)
VALUES ($1::uuid, $2::uuid, $3, now() - interval '1 day',
        now() + interval '30 days', 1000000, 'active')
ON CONFLICT (id) DO NOTHING`,
		seedCampaignID, seedAdvertiserID, "Demo Campaign")
	return err
}

func insertInternalAd(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
INSERT INTO ads (id, campaign_id, name, status, position_type, priority,
                 landing_page_url, media_source, media_url, media_mime,
                 media_duration_ms, media_width, media_height, media_bitrate_kbps)
VALUES ($1::uuid, $2::uuid, $3, 'active', 'pre', 10,
        $4, 'internal_s3', $5, 'video/mp4',
        10000, 1280, 720, 1500)
ON CONFLICT (id) DO NOTHING`,
		seedAdInternalID, seedCampaignID,
		"Big Buck Bunny (MinIO)",
		"https://example.com/landing", seedObjectKey)
	return err
}

func insertExternalAd(ctx context.Context, pool *pgxpool.Pool, mediaURL string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO ads (id, campaign_id, name, status, position_type, priority,
                 landing_page_url, media_source, media_url, media_mime,
                 media_duration_ms, media_width, media_height, media_bitrate_kbps)
VALUES ($1::uuid, $2::uuid, $3, 'active', 'pre', 5,
        $4, 'external_url', $5, 'video/mp4',
        10000, 1280, 720, 1500)
ON CONFLICT (id) DO NOTHING`,
		seedAdExternalID, seedCampaignID,
		"Big Buck Bunny (BYO-URL)",
		"https://example.com/landing", mediaURL)
	return err
}
