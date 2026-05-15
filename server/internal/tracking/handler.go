package tracking

import (
	"context"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/eliau2005/openadsource/server/internal/metrics"
)

// onePixelGIF is the 43-byte canonical 1x1 transparent GIF89a — what every
// tracking-pixel endpoint on the web returns so <img> requests get a valid
// (and visually invisible) response.
var onePixelGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00,
	0x01, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xff, 0xff, 0xff, 0x21, 0xf9, 0x04, 0x01, 0x00,
	0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00,
	0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
	0x01, 0x00, 0x3b,
}

// IdempotencyTTL is how long a (imp_id, event) tuple is remembered as
// already-fired. 1 day matches the default token TTL — any duplicate fire
// within the token's validity window is ignored.
const IdempotencyTTL = 24 * time.Hour

// Handler is the Phase 4 /track endpoint. It verifies the HMAC signature
// stamped on the URL, deduplicates by (imp_id, event), and INCRs the daily
// counter the worker will drain to daily_stats.
type Handler struct {
	signer *Signer
	client *redis.Client // may be nil — then writes are no-ops, response is the GIF
}

// NewHandler builds the handler. Either dependency can be nil in dev
// (handler then becomes a permissive logger that always returns the GIF).
func NewHandler(signer *Signer, client *redis.Client) *Handler {
	return &Handler{signer: signer, client: client}
}

// ServeTrack handles GET /track. Always returns a 1x1 GIF; the response
// status is the only side channel. We deliberately do not log the
// rejection reason back to the caller — players retry, and a noisy 4xx
// response would amplify abuse vectors.
func (h *Handler) ServeTrack(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	event := q.Get("event")
	adID := q.Get("ad_id")
	impID := q.Get("imp_id")
	sig := q.Get("sig")
	exp := parseUnix(q.Get("exp"))

	if !IsTracked(event) || adID == "" || impID == "" || sig == "" || exp == 0 {
		metrics.TrackEventsTotal.WithLabelValues(event, "invalid").Inc()
		writeGIF(w, http.StatusNoContent)
		return
	}
	if h.signer != nil {
		if err := h.signer.Verify(adID, impID, event, sig, exp, time.Now()); err != nil {
			// Silent reject — pixel responses should never leak which
			// validation step failed.
			metrics.TrackEventsTotal.WithLabelValues(event, "invalid").Inc()
			writeGIF(w, http.StatusNoContent)
			return
		}
	}

	// Try to claim the event for this imp_id. Already-claimed = duplicate.
	if h.client != nil {
		key := "track:" + impID + ":" + event
		ok, err := h.client.SetNX(r.Context(), key, "1", IdempotencyTTL).Result()
		if err != nil {
			log.Warn().Err(err).Msg("tracking: idempotency SETNX failed; serving GIF without recording")
			metrics.TrackEventsTotal.WithLabelValues(event, "ok").Inc()
			writeGIF(w, http.StatusOK)
			return
		}
		if !ok {
			// Duplicate — already counted.
			metrics.TrackEventsTotal.WithLabelValues(event, "duplicate").Inc()
			writeGIF(w, http.StatusNoContent)
			return
		}

		date := time.Now().UTC().Format("2006-01-02")
		counterKey := "ad:" + adID + ":event:" + event + ":" + date
		if _, err := h.client.Incr(r.Context(), counterKey).Result(); err != nil {
			log.Warn().Err(err).Str("key", counterKey).Msg("tracking: INCR failed")
		}
	}
	metrics.TrackEventsTotal.WithLabelValues(event, "ok").Inc()
	writeGIF(w, http.StatusOK)
}

// WriteOnePixelGIF writes the 1x1 transparent GIF with HTTP 200. Exported
// so the rate-limit middleware can reuse it as its "over-limit response"
// for /track (we never surface a 429 to a tracking pixel — players retry).
func WriteOnePixelGIF(w http.ResponseWriter, _ *http.Request) {
	writeGIF(w, http.StatusOK)
}

func writeGIF(w http.ResponseWriter, status int) {
	h := w.Header()
	h.Set("Content-Type", "image/gif")
	h.Set("Cache-Control", "no-store, max-age=0")
	h.Set("Pragma", "no-cache")
	w.WriteHeader(status)
	if status == http.StatusOK {
		_, _ = w.Write(onePixelGIF)
	}
}

func parseUnix(s string) int64 {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int64(c-'0')
		if n > 1<<62 {
			return 0
		}
	}
	return n
}

// Compile-time check that Handler exposes the std http handler shape (for
// chi route mounting).
var _ http.HandlerFunc = (*Handler)(nil).ServeTrack

// Stub remains for callers that haven't migrated yet (legacy import path
// from Phase 1 — internal/delivery wires up Handler directly now).
func Stub(w http.ResponseWriter, _ *http.Request) {
	_ = context.TODO()
	writeGIF(w, http.StatusOK)
}
