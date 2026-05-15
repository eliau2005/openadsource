package tracking

import (
	"errors"
	"testing"
	"time"
)

func TestSigner_RoundTrip(t *testing.T) {
	s := NewSigner("placeholder-tracking-secret", time.Hour)
	now := time.Now()
	sig, exp := s.Sign("ad-1", "imp-1", EventImpression, now)
	if sig == "" || exp == 0 {
		t.Fatalf("Sign returned empty: sig=%q exp=%d", sig, exp)
	}
	if err := s.Verify("ad-1", "imp-1", EventImpression, sig, exp, now); err != nil {
		t.Fatalf("Verify on roundtrip: %v", err)
	}
}

func TestSigner_TamperedAdID(t *testing.T) {
	s := NewSigner("k", time.Hour)
	now := time.Now()
	sig, exp := s.Sign("ad-1", "imp-1", EventImpression, now)
	if err := s.Verify("ad-2", "imp-1", EventImpression, sig, exp, now); !errors.Is(err, ErrSignatureMismatch) {
		t.Errorf("tampered ad_id should fail: %v", err)
	}
}

func TestSigner_TamperedEvent(t *testing.T) {
	s := NewSigner("k", time.Hour)
	now := time.Now()
	sig, exp := s.Sign("ad", "imp", EventStart, now)
	// Reusing the start signature for a complete event must fail — that's
	// the whole reason the event is included in the signed payload.
	if err := s.Verify("ad", "imp", EventComplete, sig, exp, now); !errors.Is(err, ErrSignatureMismatch) {
		t.Errorf("event-swap replay should fail: %v", err)
	}
}

func TestSigner_Expired(t *testing.T) {
	s := NewSigner("k", time.Hour)
	now := time.Now()
	sig, exp := s.Sign("ad", "imp", EventImpression, now)
	future := now.Add(2 * time.Hour)
	if err := s.Verify("ad", "imp", EventImpression, sig, exp, future); !errors.Is(err, ErrExpired) {
		t.Errorf("expired token should fail: %v", err)
	}
}

func TestSigner_MalformedSig(t *testing.T) {
	s := NewSigner("k", time.Hour)
	now := time.Now()
	_, exp := s.Sign("ad", "imp", EventImpression, now)
	if err := s.Verify("ad", "imp", EventImpression, "not-hex-zzz", exp, now); !errors.Is(err, ErrMalformedSig) {
		t.Errorf("malformed sig should fail: %v", err)
	}
}

func TestIsTracked(t *testing.T) {
	if !IsTracked(EventImpression) {
		t.Error("impression should be tracked")
	}
	if IsTracked("nope") {
		t.Error("unknown event should not be tracked")
	}
}
