package httpmw

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"

	"github.com/eliau2005/openadsource/server/internal/targeting"
)

// Limiter is a per-source-IP token-bucket rate limiter. Visitors that
// exceed the configured rate get the configured "over-limit" response
// (typically an empty VAST for /vast or a 1x1 GIF for /track) instead of a
// 429 — leaking a 429 back to a client just tells an attacker they've
// hit the gate.
type Limiter struct {
	r     rate.Limit
	b     int
	cap   int
	ip    *targeting.IPResolver
	onHit http.HandlerFunc

	bag sync.Map // string -> *visitor

	// approxSize lets Run() short-circuit the cleanup pass when the map
	// is still small. We don't need a precise count — sync.Map doesn't
	// give us one for free, but an atomic counter is close enough.
	approxSize atomic.Int64
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen atomic.Int64 // unix-second
}

// NewLimiter builds a per-IP limiter. capN bounds map size; when full, the
// next admission evicts the oldest. ip resolves the canonical source IP
// using the same trusted-proxy CIDR list the rest of the server uses.
// onLimit is what we write when the bucket is empty — pass a writer that
// emits the protocol's "natural no-fill", not 429.
func NewLimiter(rps, burst float64, capN int, ip *targeting.IPResolver, onLimit http.HandlerFunc) *Limiter {
	if capN <= 0 {
		capN = 100_000
	}
	return &Limiter{
		r:     rate.Limit(rps),
		b:     int(burst),
		cap:   capN,
		ip:    ip,
		onHit: onLimit,
	}
}

// Allow returns true when this request fits within the IP's bucket.
// Updates lastSeen on every call so Run() can prune idle entries.
func (l *Limiter) Allow(key string) bool {
	if l == nil || l.r <= 0 {
		return true
	}
	now := time.Now().Unix()
	if v, ok := l.bag.Load(key); ok {
		v := v.(*visitor)
		v.lastSeen.Store(now)
		return v.limiter.Allow()
	}
	// Cap-bounded admission. Best-effort eviction; sync.Map doesn't have
	// "remove oldest", so we pick whichever bucket the next Range hits
	// first that hasn't been seen recently. Keeps map size sub-linear in
	// abuse scenarios without paying for an LRU.
	if l.approxSize.Load() >= int64(l.cap) {
		cutoff := now - 60
		l.bag.Range(func(k, val any) bool {
			if val.(*visitor).lastSeen.Load() < cutoff {
				l.bag.Delete(k)
				l.approxSize.Add(-1)
				return false
			}
			return true
		})
	}
	v := &visitor{limiter: rate.NewLimiter(l.r, l.b)}
	v.lastSeen.Store(now)
	actual, loaded := l.bag.LoadOrStore(key, v)
	if !loaded {
		l.approxSize.Add(1)
	}
	return actual.(*visitor).limiter.Allow()
}

// Middleware is the chi-compatible wrapper. It looks up the visitor's
// canonical IP via the IPResolver, calls Allow, and either delegates to
// next or invokes the configured over-limit writer.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l == nil {
			next.ServeHTTP(w, r)
			return
		}
		key := "_"
		if l.ip != nil {
			if addr := l.ip.Resolve(r); addr.IsValid() {
				key = addr.String()
			}
		}
		if l.Allow(key) {
			next.ServeHTTP(w, r)
			return
		}
		l.onHit(w, r)
	})
}

// Run is the background sweeper. Walks the map every interval and deletes
// visitors that haven't been seen in the last `idleAge`. Cheap because
// it only fires every ~10 min, not per request.
func (l *Limiter) Run(ctx context.Context) {
	const interval = 10 * time.Minute
	const idleAge = 15 * time.Minute
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			cutoff := time.Now().Add(-idleAge).Unix()
			l.bag.Range(func(k, val any) bool {
				if val.(*visitor).lastSeen.Load() < cutoff {
					l.bag.Delete(k)
					l.approxSize.Add(-1)
				}
				return true
			})
		}
	}
}
