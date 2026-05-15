// Command worker is the Phase 4 background reconciler.
//
// Every WORKER_INTERVAL (default 30s) it competes for a Redis distributed
// lock; the winner drains today/yesterday tracking counters into the
// Postgres daily_stats table and pauses campaigns whose total_budget has
// been spent. Multiple replicas can run concurrently; the lock guarantees
// only one tick at a time across the fleet.
package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/eliau2005/openadsource/server/internal/config"
	"github.com/eliau2005/openadsource/server/internal/db"
	"github.com/eliau2005/openadsource/server/internal/tracking"
	"github.com/eliau2005/openadsource/server/internal/worker"
)

func main() {
	cfg := config.Load()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if lvl, err := zerolog.ParseLevel(cfg.LogLevel); err == nil {
		zerolog.SetGlobalLevel(lvl)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("db pool init failed")
	}
	defer pool.Close()
	if err := db.PingWithRetry(ctx, pool, 15, 2*time.Second); err != nil {
		log.Fatal().Err(err).Msg("postgres unreachable")
	}

	if cfg.RedisURL == "" {
		log.Fatal().Msg("REDIS_URL is required for the worker — counters live in Redis")
	}
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("REDIS_URL parse failed")
	}
	rc := redis.NewClient(opt)
	if err := rc.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("redis unreachable")
	}
	defer rc.Close()

	lock := worker.NewLock(rc, 60_000)
	drainer := worker.New(pool, rc, tracking.TrackedEvents)

	interval := cfg.WorkerInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	log.Info().
		Dur("interval", interval).
		Str("lock_id", lock.ID()).
		Msg("worker started")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("worker stopping")
			return
		case <-ticker.C:
			runTick(ctx, lock, drainer)
		}
	}
}

func runTick(ctx context.Context, lock *worker.Lock, drainer *worker.Drainer) {
	tickCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	ok, err := lock.TryAcquire(tickCtx)
	if err != nil {
		log.Warn().Err(err).Msg("worker tick: lock acquire failed; skipping")
		return
	}
	if !ok {
		log.Debug().Msg("worker tick: another replica holds the lock; skipping")
		return
	}
	defer func() {
		if err := lock.Release(tickCtx); err != nil {
			log.Warn().Err(err).Msg("worker tick: lock release failed")
		}
	}()

	if err := drainer.Tick(tickCtx); err != nil && !errors.Is(err, context.Canceled) {
		log.Warn().Err(err).Msg("worker tick: drain failed")
	}
}
