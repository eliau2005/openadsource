package registry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/eliau2005/openadsource/server/internal/metrics"
)

// InvalidateChannel is the Redis pub/sub channel the dashboard publishes to
// when a campaign / ad / targeting / cap_rules row changes. The Refresher
// reloads on any message; payload is ignored.
const InvalidateChannel = "oas:registry:invalidate"

// Refresher owns the live Snapshot pointer and the goroutine that keeps it
// fresh. Readers consume via Get(); the pointer is updated atomically.
type Refresher struct {
	pool     *pgxpool.Pool
	interval time.Duration
	redis    *redis.Client // optional

	snapshot atomic.Pointer[Snapshot]

	readyOnce sync.Once
	ready     chan struct{}
}

// New constructs a refresher. Pass nil for redisClient to disable pub/sub
// triggered reloads; TTL-based reloads still run.
func New(pool *pgxpool.Pool, interval time.Duration, redisClient *redis.Client) *Refresher {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &Refresher{
		pool:     pool,
		interval: interval,
		redis:    redisClient,
		ready:    make(chan struct{}),
	}
}

// Get returns the current snapshot. A single atomic load — no lock, no
// pointer indirection beyond the load itself. Returns nil before the first
// successful load; callers should consult Ready() / WaitReady() to gate.
func (r *Refresher) Get() *Snapshot { return r.snapshot.Load() }

// Ready returns a channel closed once the first successful reload completes.
func (r *Refresher) Ready() <-chan struct{} { return r.ready }

// WaitReady blocks until the first reload finishes or ctx is cancelled.
func (r *Refresher) WaitReady(ctx context.Context) error {
	select {
	case <-r.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Reload runs a single load cycle. Exposed for tests + the boot sequence
// (which wants to surface the first error).
func (r *Refresher) Reload(ctx context.Context) error {
	start := time.Now()
	snap, err := Load(ctx, r.pool)
	if err != nil {
		return fmt.Errorf("snapshot load: %w", err)
	}
	metrics.SnapshotLoadDuration.Observe(time.Since(start).Seconds())
	metrics.SnapshotAds.Set(float64(len(snap.Ads)))
	r.snapshot.Store(snap)
	r.readyOnce.Do(func() { close(r.ready) })
	log.Info().
		Int("ads", len(snap.Ads)).
		Int("positions", len(snap.ByPosition)).
		Int("countries", len(snap.ByCountry)).
		Int("devices", len(snap.ByDevice)).
		Msg("registry snapshot loaded")
	return nil
}

// Run blocks until ctx is cancelled. It does a synchronous first reload,
// then schedules TTL and pub/sub triggered reloads. After the first
// success, reload errors keep the previous snapshot (fail-open).
func (r *Refresher) Run(ctx context.Context) error {
	if err := r.Reload(ctx); err != nil {
		return fmt.Errorf("initial snapshot load: %w", err)
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	var pubsub *redis.PubSub
	var msgs <-chan *redis.Message
	if r.redis != nil {
		pubsub = r.redis.Subscribe(ctx, InvalidateChannel)
		defer pubsub.Close()
		msgs = pubsub.Channel()
		log.Info().Str("channel", InvalidateChannel).Msg("registry subscribed to invalidation channel")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.Reload(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Warn().Err(err).Msg("scheduled snapshot reload failed; keeping previous snapshot")
			}
		case msg, ok := <-msgs:
			if !ok {
				msgs = nil
				continue
			}
			log.Info().Str("payload", msg.Payload).Msg("registry invalidation received")
			if err := r.Reload(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Warn().Err(err).Msg("invalidation-triggered reload failed; keeping previous snapshot")
			}
		}
	}
}
