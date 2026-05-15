// Package worker owns the background loop that drains Redis tracking
// counters into Postgres daily_stats and reconciles campaign budgets. A
// Redis SET-NX-PX distributed lock guarantees only one worker replica runs
// a tick at a time.
package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// LockKey is the canonical Redis key the worker uses for its single-owner
// guard. Different worker replicas all contend on the same key; only the
// first to SET NX wins the tick.
const LockKey = "worker:lock"

// releaseScript safely deletes the lock only when the running process
// still owns it. Prevents a slow-tick worker from accidentally releasing
// a lock another replica has already taken over (after TTL expiry).
const releaseScript = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0
`

// ErrLocked is returned by TryAcquire when another worker already owns the
// lock. Callers treat this as "skip this tick".
var ErrLocked = errors.New("worker: lock held by another instance")

// Lock is a per-process distributed-lock client. ID is randomized at
// construction so two replicas writing concurrently won't think they're
// the same owner. Concurrent calls from the same process are not safe;
// the worker only uses one goroutine to drive ticks.
type Lock struct {
	client *redis.Client
	ttl    int64 // milliseconds
	id     string
}

// NewLock builds a lock client. ttlMillis defaults to 60_000 if not set.
func NewLock(client *redis.Client, ttlMillis int64) *Lock {
	if ttlMillis <= 0 {
		ttlMillis = 60_000
	}
	return &Lock{client: client, ttl: ttlMillis, id: uuid.NewString()}
}

// ID exposes the per-process token for diagnostics.
func (l *Lock) ID() string { return l.id }

// TryAcquire atomically SETs the lock key with the current ID if and only
// if the key is unset. Returns (true, nil) on success, (false, nil) when
// another instance already holds it, and (false, err) for Redis errors.
func (l *Lock) TryAcquire(ctx context.Context) (bool, error) {
	if l == nil || l.client == nil {
		return true, nil
	}
	ok, err := l.client.SetNX(ctx, LockKey, l.id, time.Duration(l.ttl)*time.Millisecond).Result()
	if err != nil {
		return false, fmt.Errorf("worker lock SETNX: %w", err)
	}
	return ok, nil
}

// Release runs the Lua compare-and-delete script: only delete the key
// when the value still equals our ID. No-op when the lock has already
// expired or been taken over.
func (l *Lock) Release(ctx context.Context) error {
	if l == nil || l.client == nil {
		return nil
	}
	_, err := l.client.Eval(ctx, releaseScript, []string{LockKey}, l.id).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("worker lock release: %w", err)
	}
	return nil
}

