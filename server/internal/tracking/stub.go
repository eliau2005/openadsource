// Package tracking owns the /track endpoint that VAST players ping for
// impression, quartile, and click events. Phase 1 is a no-op stub that
// returns a 1x1 transparent GIF; Phase 4 replaces this with the real
// Redis-backed counter recorder.
package tracking

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

// onePixelGIF is the 43-byte canonical 1x1 transparent GIF89a. It's what
// every tracking-pixel endpoint on the web returns so <img> requests get a
// valid (and visually invisible) response.
var onePixelGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00,
	0x01, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xff, 0xff, 0xff, 0x21, 0xf9, 0x04, 0x01, 0x00,
	0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00,
	0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
	0x01, 0x00, 0x3b,
}

// Stub serves the 1x1 GIF and logs the inbound query string so Phase 1
// developers can confirm impressions are firing. Returns 200 (not 204)
// because RFC 9110 forbids a body on 204 responses; players accept both.
func Stub(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	log.Info().
		Str("event", q.Get("event")).
		Str("ad_id", q.Get("ad_id")).
		Str("imp_id", q.Get("imp_id")).
		Str("remote", r.RemoteAddr).
		Msg("track stub hit (Phase 4 will record this)")

	h := w.Header()
	h.Set("Content-Type", "image/gif")
	h.Set("Cache-Control", "no-store, max-age=0")
	h.Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(onePixelGIF)
}
