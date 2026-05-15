package capping

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// freqScript: KEYS[1] = "user:{uid}:ad:{ad_id}:count",
//
//	ARGV[1] = max_impressions, ARGV[2] = time_window_seconds.
//
// INCR; on first set establish the EXPIRE; if cap exceeded DECR back and
// return -1 so the selector tries another candidate.
const freqScript = `
local n = redis.call('INCR', KEYS[1])
if n == 1 then
  redis.call('EXPIRE', KEYS[1], tonumber(ARGV[2]))
end
local cap = tonumber(ARGV[1])
if cap > 0 and n > cap then
  redis.call('DECR', KEYS[1])
  return -1
end
return n
`

// ErrFreqCapExceeded is returned by FrequencyEnforcer.TryConsume when the
// user has hit their per-ad cap within the configured window.
var ErrFreqCapExceeded = errors.New("user frequency cap exceeded")

// FrequencyEnforcer wraps the per-user-per-ad cap Lua script. Construct
// once per process. A nil receiver is a no-op success path so dev stacks
// without Redis still serve.
type FrequencyEnforcer struct {
	client *redis.Client
	sha    string
}

// NewFrequencyEnforcer pre-loads the cap Lua script into Redis so the hot
// path is pure EVALSHA. Returns (nil, nil) when client is nil.
func NewFrequencyEnforcer(ctx context.Context, client *redis.Client) (*FrequencyEnforcer, error) {
	if client == nil {
		return nil, nil
	}
	sha, err := client.ScriptLoad(ctx, freqScript).Result()
	if err != nil {
		return nil, fmt.Errorf("load frequency script: %w", err)
	}
	return &FrequencyEnforcer{client: client, sha: sha}, nil
}

// TryConsume atomically increments the user's per-ad counter and rejects
// if the new value exceeds the cap. cap == 0 → no cap rule for this ad,
// the call is a no-op success. A nil receiver returns nil too.
func (e *FrequencyEnforcer) TryConsume(ctx context.Context, userID, adID string, cap int32, windowSec int32) error {
	if e == nil || cap <= 0 {
		return nil
	}
	keys := []string{"user:" + userID + ":ad:" + adID + ":count"}
	args := []any{int64(cap), int64(windowSec)}
	res, err := e.client.EvalSha(ctx, e.sha, keys, args...).Int64()
	if err == nil {
		if res == -1 {
			return ErrFreqCapExceeded
		}
		return nil
	}
	if isNoScriptErr(err) {
		res, err = e.client.Eval(ctx, freqScript, keys, args...).Int64()
		if err == nil {
			if res == -1 {
				return ErrFreqCapExceeded
			}
			return nil
		}
	}
	return fmt.Errorf("freq enforcer redis call: %w", err)
}
