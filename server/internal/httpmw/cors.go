// Package httpmw holds the few HTTP middlewares the adserver routes use.
package httpmw

import "net/http"

// CORS returns a middleware that adds permissive CORS headers and short-
// circuits OPTIONS preflight requests with 204. Used only on the public
// delivery routes (/vast, /track) which are designed to be loaded
// cross-origin by publisher pages and video players.
func CORS(allowOrigin string) func(http.Handler) http.Handler {
	if allowOrigin == "" {
		allowOrigin = "*"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", allowOrigin)
			h.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Content-Type")
			h.Set("Access-Control-Max-Age", "3600")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
