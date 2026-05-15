package tracking

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/netip"

	"github.com/google/uuid"
)

// UIDCookieName is the cookie /vast sets to remember the visitor's anonymous
// identity across requests. Used as the {uid} component of the frequency
// cap counter key.
const UIDCookieName = "oas_uid"

// UIDCookieMaxAge is the cookie validity in seconds — one year. Long
// enough to keep a single visitor identified across an entire campaign
// lifetime without being a permanent identifier.
const UIDCookieMaxAge = 365 * 24 * 3600

// Resolve picks the canonical user_id for a request. Priority:
//  1. ?user_id= query param (test override; never sets the cookie).
//  2. oas_uid cookie (the normal path for returning visitors).
//  3. Fresh UUID (set=true so the caller writes Set-Cookie back).
//
// The IP+UA hash is exposed via FallbackHash for callers that genuinely
// can't rely on cookies (server-side pixel firings on /track aren't one
// of those — they're keyed on imp_id, not user_id).
func Resolve(r *http.Request) (uid string, set bool) {
	if v := r.URL.Query().Get("user_id"); v != "" {
		return v, false
	}
	if c, err := r.Cookie(UIDCookieName); err == nil && c.Value != "" {
		return c.Value, false
	}
	return uuid.NewString(), true
}

// FallbackHash deterministically derives a user-id from peer IP + UA. Used
// when neither the query param nor a cookie are present and the caller
// doesn't want to set a cookie either (rare).
func FallbackHash(ip netip.Addr, userAgent string) string {
	h := sha256.New()
	h.Write([]byte(ip.String()))
	h.Write([]byte{0})
	h.Write([]byte(userAgent))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// SetUIDCookie writes a fresh oas_uid cookie. HttpOnly + SameSite=Lax;
// Secure flips on when the deploy is behind https — caller passes that bit
// in (matches the Phase 2 session-cookie decision).
func SetUIDCookie(w http.ResponseWriter, uid string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     UIDCookieName,
		Value:    uid,
		Path:     "/",
		MaxAge:   UIDCookieMaxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
