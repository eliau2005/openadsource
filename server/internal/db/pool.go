package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool builds a pgx connection pool with sensible defaults for the
// adserver hot path: a small min-size to keep idle cost low, a max-size
// that comfortably absorbs bursty traffic, and an active health check so
// stale connections are pruned before they reach a handler.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, errors.New("database url is empty")
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse pgx config: %w", err)
	}
	cfg.MinConns = 2
	cfg.MaxConns = 20
	cfg.HealthCheckPeriod = 30 * time.Second
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pgx pool: %w", err)
	}
	return pool, nil
}

// PingWithRetry blocks until the pool can serve a Ping or the budget is
// exhausted. Phase 0 compose ordering already waits for postgres to report
// healthy before booting the adserver, so a short retry budget here is just
// belt-and-braces for cold-start races.
func PingWithRetry(ctx context.Context, pool *pgxpool.Pool, attempts int, interval time.Duration) error {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := pool.Ping(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
	return fmt.Errorf("postgres unreachable after %d attempts: %w", attempts, lastErr)
}
