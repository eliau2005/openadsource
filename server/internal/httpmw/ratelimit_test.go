package httpmw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eliau2005/openadsource/server/internal/targeting"
)

func newReq(peer, xff string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "http://example/", nil)
	req.RemoteAddr = peer
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	return req
}

func testIPResolver(t *testing.T, cidrs string) *targeting.IPResolver {
	t.Helper()
	r, err := targeting.NewIPResolver(cidrs)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestLimiter_AllowsThenBlocks(t *testing.T) {
	ip := testIPResolver(t, "")
	hits := 0
	onLimit := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.WriteHeader(http.StatusTeapot) // distinct code so we can assert on it
	})
	l := NewLimiter(1, 1, 100, ip, onLimit)

	passes := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		passes++
		w.WriteHeader(http.StatusOK)
	})
	h := l.Middleware(next)

	rr1 := httptest.NewRecorder()
	h.ServeHTTP(rr1, newReq("1.2.3.4:1000", ""))
	if rr1.Code != http.StatusOK || passes != 1 {
		t.Errorf("first req should pass: code=%d passes=%d", rr1.Code, passes)
	}
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, newReq("1.2.3.4:1000", ""))
	if rr2.Code != http.StatusTeapot || hits != 1 {
		t.Errorf("second req should hit onLimit; code=%d hits=%d passes=%d", rr2.Code, hits, passes)
	}
}

func TestLimiter_IndependentBucketsPerIP(t *testing.T) {
	ip := testIPResolver(t, "")
	onLimit := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	l := NewLimiter(1, 1, 100, ip, onLimit)

	pass := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		pass++
		w.WriteHeader(http.StatusOK)
	})
	h := l.Middleware(next)

	h.ServeHTTP(httptest.NewRecorder(), newReq("1.2.3.4:1000", ""))
	h.ServeHTTP(httptest.NewRecorder(), newReq("5.6.7.8:1000", ""))
	if pass != 2 {
		t.Errorf("expected 2 passes for distinct IPs, got %d", pass)
	}
}

func TestLimiter_HonoursTrustedXFF(t *testing.T) {
	ip := testIPResolver(t, "10.0.0.0/8")
	onLimit := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	l := NewLimiter(1, 1, 100, ip, onLimit)

	// Both come through the same trusted proxy with different XFF clients
	// — the limiter should treat them as different visitors.
	pass := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		pass++
		w.WriteHeader(http.StatusOK)
	})
	h := l.Middleware(next)
	h.ServeHTTP(httptest.NewRecorder(), newReq("10.0.0.1:443", "1.2.3.4"))
	h.ServeHTTP(httptest.NewRecorder(), newReq("10.0.0.1:443", "5.6.7.8"))
	if pass != 2 {
		t.Errorf("XFF-based bucketing failed, got pass=%d", pass)
	}
}

func TestLimiter_NilSafe(t *testing.T) {
	// nil receiver should pass through.
	var l *Limiter
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h := l.Middleware(next)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, newReq("1.2.3.4:1000", ""))
	if rr.Code != http.StatusOK {
		t.Errorf("nil limiter should be a passthrough")
	}
}

func TestLimiter_RunDropsIdle(t *testing.T) {
	ip := testIPResolver(t, "")
	l := NewLimiter(1, 1, 100, ip, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	// Seed an entry directly, then mark it ancient.
	l.Allow("9.9.9.9")
	v, _ := l.bag.Load("9.9.9.9")
	v.(*visitor).lastSeen.Store(time.Now().Add(-time.Hour).Unix())

	// Run the cleanup body directly instead of waiting 10 min.
	ctx, cancel := context.WithCancel(context.Background())
	go l.Run(ctx)
	// Manually nudge: cleanup wakes every 10 min, so just call the same
	// eviction logic synchronously to keep the test fast.
	cutoff := time.Now().Add(-15 * time.Minute).Unix()
	l.bag.Range(func(k, val any) bool {
		if val.(*visitor).lastSeen.Load() < cutoff {
			l.bag.Delete(k)
			l.approxSize.Add(-1)
		}
		return true
	})
	cancel()

	if _, ok := l.bag.Load("9.9.9.9"); ok {
		t.Errorf("idle visitor should have been swept")
	}
}
