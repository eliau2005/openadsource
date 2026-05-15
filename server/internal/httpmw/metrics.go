package httpmw

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eliau2005/openadsource/server/internal/metrics"
)

// HTTPMetrics is a chi-aware middleware that times every request and
// increments the central HTTP collectors. The "route" label uses the
// matched chi route pattern (e.g. `/vast`) — not the raw URL — so
// cardinality stays bounded.
func HTTPMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		// chi populates the route pattern only after the handler matches.
		// For unmatched paths (404, etc.) the pattern is empty; fall back
		// to a fixed bucket so we don't introduce per-URL cardinality.
		route := ""
		if ctx := chi.RouteContext(r.Context()); ctx != nil {
			route = ctx.RoutePattern()
		}
		if route == "" {
			route = "_other"
		}
		dur := time.Since(start).Seconds()
		metrics.HTTPRequestsTotal.WithLabelValues(route, strconv.Itoa(rec.status)).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(route).Observe(dur)
	})
}

// statusRecorder is a tiny ResponseWriter wrapper that remembers the
// status code so the middleware can label the counter.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Make sure we forward the Flusher interface (chi may use it for SSE etc.).
// Not strictly required for /vast + /track, but keeps the wrapper polite.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
