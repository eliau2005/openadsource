// Package tracking owns the VAST tracking pixel signing + verifying logic
// and (in Phase 4) the /track endpoint that ingests the resulting events.
package tracking

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"time"
)

// Signer mints and verifies HMAC-SHA256 signatures on tracking URLs.
// Canonical signed message: "{adID}|{impID}|{event}|{exp}". Including the
// event in the payload prevents a captured "start" pixel URL from being
// re-used as a "complete" later, even with the same imp_id.
type Signer struct {
	key []byte
	ttl time.Duration
}

// Errors returned by Verify. Handlers should treat all of them as "drop the
// hit silently with a 204+GIF" — never log the input back to the caller.
var (
	ErrSignatureMismatch = errors.New("tracking: signature mismatch")
	ErrExpired           = errors.New("tracking: token expired")
	ErrMalformedExpiry   = errors.New("tracking: malformed expiry")
	ErrMalformedSig      = errors.New("tracking: malformed signature")
)

// NewSigner builds a Signer. secret must be non-empty (boot should fail
// fast when TRACKING_SECRET is missing). ttl is the validity window each
// fresh URL gets when Sign is called.
func NewSigner(secret string, ttl time.Duration) *Signer {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Signer{key: []byte(secret), ttl: ttl}
}

// TTL exposes the signer's default token-validity window so handlers can
// quote it back to the player if needed.
func (s *Signer) TTL() time.Duration { return s.ttl }

// Sign returns the hex-encoded HMAC and the absolute unix-epoch second the
// signature expires at. Callers stitch both into the resulting URL.
func (s *Signer) Sign(adID, impID, event string, now time.Time) (sig string, exp int64) {
	exp = now.Add(s.ttl).Unix()
	sig = s.compute(adID, impID, event, exp)
	return sig, exp
}

// Verify checks the signature, then the expiry. Both must hold.
func (s *Signer) Verify(adID, impID, event, sig string, exp int64, now time.Time) error {
	if exp <= 0 {
		return ErrMalformedExpiry
	}
	want := s.compute(adID, impID, event, exp)
	got, err := hex.DecodeString(sig)
	if err != nil {
		return ErrMalformedSig
	}
	wantBytes, _ := hex.DecodeString(want)
	if !hmac.Equal(got, wantBytes) {
		return ErrSignatureMismatch
	}
	if exp < now.Unix() {
		return ErrExpired
	}
	return nil
}

func (s *Signer) compute(adID, impID, event string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(adID))
	mac.Write([]byte{'|'})
	mac.Write([]byte(impID))
	mac.Write([]byte{'|'})
	mac.Write([]byte(event))
	mac.Write([]byte{'|'})
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}
