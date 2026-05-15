package targeting

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newReqFromPeer(t *testing.T, peer, xff string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "http://example/x", nil)
	req.RemoteAddr = peer
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	return req
}

func TestIPResolver_PeerIgnoredWhenUntrusted(t *testing.T) {
	r, err := NewIPResolver("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	req := newReqFromPeer(t, "8.8.8.8:12345", "1.2.3.4")
	got := r.Resolve(req).String()
	if got != "8.8.8.8" {
		t.Errorf("untrusted peer should win over XFF, got %q", got)
	}
}

func TestIPResolver_XFFTrusted(t *testing.T) {
	r, err := NewIPResolver("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	req := newReqFromPeer(t, "10.20.30.40:12345", "5.6.7.8")
	got := r.Resolve(req).String()
	if got != "5.6.7.8" {
		t.Errorf("trusted peer + XFF should yield XFF leftmost public, got %q", got)
	}
}

func TestIPResolver_XFFStripsTrustedHops(t *testing.T) {
	r, err := NewIPResolver("10.0.0.0/8,172.16.0.0/12")
	if err != nil {
		t.Fatal(err)
	}
	// Real client 4.4.4.4 -> trusted proxy 172.20.0.1 -> trusted lb 10.0.0.1
	req := newReqFromPeer(t, "10.0.0.1:443", "4.4.4.4, 172.20.0.1")
	got := r.Resolve(req).String()
	if got != "4.4.4.4" {
		t.Errorf("trusted hops should be stripped, got %q", got)
	}
}

func TestIPResolver_EmptyXFFFallsBackToPeer(t *testing.T) {
	r, err := NewIPResolver("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	req := newReqFromPeer(t, "10.0.0.7:443", "")
	got := r.Resolve(req).String()
	if got != "10.0.0.7" {
		t.Errorf("missing XFF should fall back to peer, got %q", got)
	}
}

func TestIPResolver_EmptyTrustedList(t *testing.T) {
	r, err := NewIPResolver("")
	if err != nil {
		t.Fatal(err)
	}
	req := newReqFromPeer(t, "8.8.8.8:443", "1.2.3.4")
	got := r.Resolve(req).String()
	if got != "8.8.8.8" {
		t.Errorf("empty trusted list: peer always wins, got %q", got)
	}
}

func TestIPResolver_BadCIDR(t *testing.T) {
	if _, err := NewIPResolver("not-a-cidr"); err == nil {
		t.Error("expected error for bad CIDR")
	}
}
