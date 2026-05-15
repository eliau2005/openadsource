// Package delivery wires the in-memory registry, selection logic, storage
// resolver, and VAST builder into the GET /vast HTTP endpoint.
//
// Phase 3 contract:
//   - Zero Postgres reads on the hot path. The Snapshot is read with a
//     single atomic.Pointer.Load().
//   - At most one Redis round trip per request (the budget Lua script).
//     Redis is optional in dev — a nil enforcer always allows.
//   - Selection allocates ≤ 2 objects per request in steady state (see
//     internal/selection/select_bench_test.go).
//
// Failure modes always emit a valid empty VAST 4.2; the player never sees a
// 5xx (chi's Recoverer middleware also catches panics).
package delivery

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/eliau2005/openadsource/server/internal/capping"
	"github.com/eliau2005/openadsource/server/internal/config"
	"github.com/eliau2005/openadsource/server/internal/registry"
	"github.com/eliau2005/openadsource/server/internal/selection"
	"github.com/eliau2005/openadsource/server/internal/storage"
	"github.com/eliau2005/openadsource/server/internal/targeting"
	"github.com/eliau2005/openadsource/server/internal/tracking"
	"github.com/eliau2005/openadsource/server/internal/vast"
	"time"
)

// Handler holds the dependencies needed to serve /vast. Constructed once at
// boot and shared across requests. Methods must stay goroutine-safe.
type Handler struct {
	cfg      config.Config
	registry *registry.Refresher
	resolver storage.Resolver
	budget   *capping.Enforcer
	ip       *targeting.IPResolver
	geo      targeting.GeoResolver
	signer   *tracking.Signer
}

// New constructs a Handler. registry must already have a snapshot loaded
// (gate on registry.WaitReady before exposing the listener).
func New(
	cfg config.Config,
	reg *registry.Refresher,
	resolver storage.Resolver,
	budget *capping.Enforcer,
	ip *targeting.IPResolver,
	geo targeting.GeoResolver,
	signer *tracking.Signer,
) *Handler {
	return &Handler{
		cfg:      cfg,
		registry: reg,
		resolver: resolver,
		budget:   budget,
		ip:       ip,
		geo:      geo,
		signer:   signer,
	}
}

// maxBudgetRetries is how many candidates we'll cycle through when budget
// reservations fail before falling back to no-fill. Bounded to keep the hot
// path's Redis RTT cost finite.
const maxBudgetRetries = 5

// ServeVAST is the GET /vast handler.
func (h *Handler) ServeVAST(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")

	snap := h.registry.Get()
	if snap == nil {
		// Snapshot not yet loaded — should be impossible because the
		// listener doesn't open until WaitReady fires, but be defensive.
		h.writeEmpty(w)
		return
	}

	q := r.URL.Query()
	req := selection.Request{
		Pos:    defaultStr(q.Get("pos"), "pre"),
		Offset: parseInt32(q.Get("offset")),
	}

	// Country / device: explicit query param wins, else GeoIP + UA derive.
	if c := q.Get("country"); c != "" {
		req.Country = c
	} else {
		ip := h.ip.Resolve(r)
		req.Country = h.geo.CountryISO(ip)
	}
	if d := q.Get("device"); d != "" {
		req.Device = targeting.NormaliseDevice(d)
	} else {
		req.Device = targeting.ClassifyUA(r.UserAgent())
	}

	// Test override: ?ad_id=<uuid> bypasses selection but still goes
	// through budget enforcement. Lookup stays memory-only.
	if rawID := q.Get("ad_id"); rawID != "" {
		id, err := uuid.Parse(rawID)
		if err != nil {
			h.writeEmpty(w)
			return
		}
		ad, ok := snap.ByID[id]
		if !ok {
			h.writeEmpty(w)
			return
		}
		h.serveAd(w, r, ad)
		return
	}

	// Selection loop — retry up to maxBudgetRetries candidates, excluding
	// any whose budget was rejected by Redis.
	var excluded map[int]bool
	for attempt := 0; attempt < maxBudgetRetries; attempt++ {
		req.Exclude = excluded
		cand := selection.Select(snap, req)
		if cand == nil {
			h.writeEmpty(w)
			return
		}
		_, err := h.budget.TryReserve(r.Context(), cand.CampaignID.String(), cand.BudgetTotal)
		if err == nil {
			h.serveAd(w, r, cand)
			return
		}
		if !errors.Is(err, capping.BudgetExhausted) {
			log.Warn().Err(err).Msg("budget enforcer error; treating as no-fill")
			h.writeEmpty(w)
			return
		}
		// Mark exhausted by ad index and retry.
		idx := -1
		for i, a := range snap.Ads {
			if a == cand {
				idx = i
				break
			}
		}
		if idx < 0 {
			h.writeEmpty(w)
			return
		}
		if excluded == nil {
			excluded = make(map[int]bool, maxBudgetRetries)
		}
		excluded[idx] = true
	}
	h.writeEmpty(w)
}

// serveAd resolves the media URL, builds the VAST, and writes the
// response. Reused by both the selection path and the ?ad_id= test path.
// Mints a fresh imp_id and stitches it (plus an HMAC signature shared
// across the event family) into every tracking pixel URL.
func (h *Handler) serveAd(w http.ResponseWriter, r *http.Request, ad *registry.Ad) {
	mediaURL, err := h.resolver.ResolveMediaURL(r.Context(), ad.MediaSource, ad.MediaURL)
	if err != nil {
		log.Warn().Str("ad_id", ad.ID.String()).Err(err).Msg("resolver failed; no-fill")
		h.writeEmpty(w)
		return
	}

	adID := ad.ID.String()
	impID := uuid.New().String()
	now := time.Now()
	impressionURL := h.signedTrackingURL(adID, impID, tracking.EventImpression, now)
	clickTrackURL := h.signedTrackingURL(adID, impID, tracking.EventClick, now)
	quartileURLs := make(map[string]string, len(tracking.QuartileEventsInOrder))
	for _, ev := range tracking.QuartileEventsInOrder {
		quartileURLs[ev] = h.signedTrackingURL(adID, impID, ev, now)
	}

	body, err := vast.BuildInline(vast.InlineInput{
		AdID:          adID,
		Title:         ad.Name,
		ImpressionURL: impressionURL,
		MediaURL:      mediaURL,
		MediaMime:     ad.MediaMime,
		MediaWidth:    int(ad.MediaWidth),
		MediaHeight:   int(ad.MediaHeight),
		MediaBitrate:  int(ad.MediaBitrate),
		MediaDuration: formatDuration(ad.MediaDurationMs),
		LandingURL:    ad.LandingPageURL,
		QuartileURLs:  quartileURLs,
		ClickTrackURL: clickTrackURL,
	})
	if err != nil {
		log.Error().Str("ad_id", ad.ID.String()).Err(err).Msg("BuildInline failed; no-fill")
		h.writeEmpty(w)
		return
	}
	_, _ = w.Write(body)
}

func (h *Handler) writeEmpty(w http.ResponseWriter) {
	body, err := vast.BuildEmpty()
	if err != nil {
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<VAST version="4.2"></VAST>` + "\n"))
		return
	}
	_, _ = w.Write(body)
}

// signedTrackingURL returns the canonical signed pixel URL the player will
// fire for `event`. The signature covers (ad_id, imp_id, event, exp) so a
// captured URL can't be re-used for a different event.
func (h *Handler) signedTrackingURL(adID, impID, event string, now time.Time) string {
	sig, exp := h.signer.Sign(adID, impID, event, now)
	return fmt.Sprintf(
		"%s/track?event=%s&ad_id=%s&imp_id=%s&exp=%d&sig=%s",
		h.cfg.PublicBaseURL, event, adID, impID, exp, sig,
	)
}

func formatDuration(ms int32) string {
	if ms <= 0 {
		return ""
	}
	secs := ms / 1000
	hh := secs / 3600
	mm := (secs / 60) % 60
	ss := secs % 60
	return fmt.Sprintf("%02d:%02d:%02d", hh, mm, ss)
}

func parseInt32(s string) int32 {
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0
	}
	return int32(n)
}

func defaultStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
