// Package capping owns the Redis-backed atomic counters that gate ad
// delivery. Phase 3 implements the campaign-level budget check on the /vast
// hot path; Phase 4 will extend the package with the per-user frequency
// cap.
package capping

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

// budgetScript: KEYS[1] = "campaign:{id}:imps_total", ARGV[1] = budget cap
// (0 = unlimited). INCR; if the new total exceeds the cap, DECR back and
// return -1 so the selector tries another candidate. Otherwise return the
// new total. Single round trip, no key scans.
const budgetScript = `
local n = redis.call('INCR', KEYS[1])
local cap = tonumber(ARGV[1])
if cap > 0 and n > cap then
  redis.call('DECR', KEYS[1])
  return -1
end
return n
`

// BudgetExhausted is returned by TryReserve when the campaign has hit its
// total_budget_impressions ceiling. Handlers should treat it as "drop this
// candidate" and try the next one.
var BudgetExhausted = errors.New("campaign budget exhausted")

// Enforcer wraps the Lua script + EVALSHA cache. Construct one per process;
// it's goroutine-safe. A nil Enforcer always succeeds (handy for tests and
// dev stacks that haven't configured Redis).
type Enforcer struct {
	client *redis.Client
	sha    string
}

// New returns an enforcer that loads the budget script into Redis up front
// (so the hot path is purely EVALSHA). If client is nil, the returned
// pointer is also nil and TryReserve becomes a no-op success path.
func New(ctx context.Context, client *redis.Client) (*Enforcer, error) {
	if client == nil {
		return nil, nil
	}
	sha, err := client.ScriptLoad(ctx, budgetScript).Result()
	if err != nil {
		return nil, fmt.Errorf("load budget script: %w", err)
	}
	return &Enforcer{client: client, sha: sha}, nil
}

// TryReserve atomically reserves a single impression slot for the given
// campaign. Returns the new in-flight total on success, or
// BudgetExhausted when the campaign has already served budget impressions.
// A nil receiver always succeeds (dev mode without Redis).
//
// cap == 0 means unlimited; the script in that case always returns the
// incremented value.
func (e *Enforcer) TryReserve(ctx context.Context, campaignID string, cap int32) (int64, error) {
	if e == nil {
		return 1, nil
	}
	keys := []string{"campaign:" + campaignID + ":imps_total"}
	args := []any{int64(cap)}
	res, err := e.client.EvalSha(ctx, e.sha, keys, args...).Int64()
	if err == nil {
		if res == -1 {
			return 0, BudgetExhausted
		}
		return res, nil
	}
	// Fall back to EVAL on NOSCRIPT (Redis flushed its script cache, e.g.
	// after a restart). This also re-caches the SHA implicitly.
	if isNoScriptErr(err) {
		res, err = e.client.Eval(ctx, budgetScript, keys, args...).Int64()
		if err == nil {
			if res == -1 {
				return 0, BudgetExhausted
			}
			return res, nil
		}
	}
	return 0, fmt.Errorf("budget enforcer redis call: %w", err)
}

func isNoScriptErr(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "NOSCRIPT")
}
