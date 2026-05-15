// Package delivery wires Postgres reads, the storage resolver, and the
// VAST builder into the `GET /vast` HTTP endpoint.
//
// Phase 1 contract:
//   - GET /vast?ad_id=<uuid> → 200 application/xml with an InLine VAST 4.2
//     response when the ad + campaign are active and not expired.
//   - GET /vast (no ad_id) or any failure path (bad UUID, ad not found,
//     campaign paused / completed / archived / expired, resolver error) →
//     200 application/xml with a no-fill VAST. Never 5xx to a player.
//
// Phase 3 will keep this handler but replace the DB read with an in-memory
// snapshot lookup and add selection logic; the request → response shape
// stays the same.
package delivery

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/eliau2005/openadsource/server/internal/config"
	"github.com/eliau2005/openadsource/server/internal/db"
	"github.com/eliau2005/openadsource/server/internal/storage"
	"github.com/eliau2005/openadsource/server/internal/vast"
)

// Handler holds the dependencies needed to serve /vast. Constructed once at
// boot and shared across requests; methods must stay goroutine-safe.
type Handler struct {
	cfg      config.Config
	queries  *db.Queries
	resolver storage.Resolver
}

// New constructs a Handler. The caller is responsible for closing the pool
// behind queries when shutting down.
func New(cfg config.Config, queries *db.Queries, resolver storage.Resolver) *Handler {
	return &Handler{cfg: cfg, queries: queries, resolver: resolver}
}

// ServeVAST is the GET /vast handler.
func (h *Handler) ServeVAST(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")

	rawAdID := r.URL.Query().Get("ad_id")
	if rawAdID == "" {
		h.writeEmpty(w)
		return
	}

	adID, err := uuid.Parse(rawAdID)
	if err != nil {
		log.Debug().Str("ad_id", rawAdID).Err(err).Msg("invalid ad_id, serving no-fill")
		h.writeEmpty(w)
		return
	}

	pgID := pgtype.UUID{Bytes: adID, Valid: true}
	row, err := h.queries.GetAdByID(r.Context(), pgID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Warn().Str("ad_id", adID.String()).Err(err).Msg("GetAdByID failed, serving no-fill")
		}
		h.writeEmpty(w)
		return
	}

	if !adEligible(row) {
		log.Debug().Str("ad_id", adID.String()).Msg("ad ineligible, serving no-fill")
		h.writeEmpty(w)
		return
	}

	mediaURL, mediaMime, err := h.resolver.ResolveMediaURL(r.Context(), row)
	if err != nil {
		log.Warn().Str("ad_id", adID.String()).Err(err).Msg("resolver failed, serving no-fill")
		h.writeEmpty(w)
		return
	}

	input := vast.InlineInput{
		AdID:          adID.String(),
		Title:         row.Name,
		ImpressionURL: h.impressionURL(adID),
		MediaURL:      mediaURL,
		MediaMime:     mediaMime,
		MediaWidth:    derefInt32(row.MediaWidth),
		MediaHeight:   derefInt32(row.MediaHeight),
		MediaBitrate:  derefInt32(row.MediaBitrateKbps),
		MediaDuration: formatDuration(row.MediaDurationMs),
		LandingURL:    derefString(row.LandingPageUrl),
	}
	body, err := vast.BuildInline(input)
	if err != nil {
		log.Error().Str("ad_id", adID.String()).Err(err).Msg("BuildInline failed, serving no-fill")
		h.writeEmpty(w)
		return
	}
	_, _ = w.Write(body)
}

func (h *Handler) writeEmpty(w http.ResponseWriter) {
	body, err := vast.BuildEmpty()
	if err != nil {
		// BuildEmpty marshals a static struct — practically infallible. If
		// it ever fails, write a hard-coded minimal VAST so the player still
		// gets parseable XML.
		log.Error().Err(err).Msg("BuildEmpty failed, falling back to literal")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<VAST version="4.2"></VAST>` + "\n"))
		return
	}
	_, _ = w.Write(body)
}

func (h *Handler) impressionURL(adID uuid.UUID) string {
	return fmt.Sprintf("%s/track?event=impression&ad_id=%s", h.cfg.PublicBaseURL, adID.String())
}

func adEligible(row db.GetAdByIDRow) bool {
	if row.Status != "active" {
		return false
	}
	if row.CampaignStatus != "active" {
		return false
	}
	if row.CampaignEndDate.Valid && row.CampaignEndDate.Time.Before(time.Now()) {
		return false
	}
	return true
}

func formatDuration(ms *int32) string {
	if ms == nil || *ms <= 0 {
		return ""
	}
	secs := *ms / 1000
	hh := secs / 3600
	mm := (secs / 60) % 60
	ss := secs % 60
	return fmt.Sprintf("%02d:%02d:%02d", hh, mm, ss)
}

func derefInt32(p *int32) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
