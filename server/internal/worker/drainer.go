package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/eliau2005/openadsource/server/internal/registry"
	"github.com/eliau2005/openadsource/server/internal/tracking"
)

// Drainer runs the per-tick work: load the current ad universe, pull
// counters out of Redis with GETDEL, upsert deltas into daily_stats, and
// pause campaigns that have spent their total budget.
type Drainer struct {
	pool   *pgxpool.Pool
	redis  *redis.Client
	events []string
}

// New constructs a Drainer. events is the allowlist of Redis-counter keys
// to drain per ad (matches tracking.TrackedEvents in production).
func New(pool *pgxpool.Pool, client *redis.Client, events []string) *Drainer {
	return &Drainer{pool: pool, redis: client, events: events}
}

// statsDelta accumulates per-event counts for a single (campaign, ad, date)
// tuple. The worker is the sole writer of daily_stats so we don't need to
// read existing rows before upserting — the SQL adds EXCLUDED.x to the
// stored value via the ON CONFLICT clause.
type statsDelta struct {
	campaignID uuid.UUID
	adID       uuid.UUID
	date       string
	imps, clk  int64
	start, q25 int64
	q50, q75   int64
	complete   int64
}

func (d *statsDelta) anyNonZero() bool {
	return d.imps+d.clk+d.start+d.q25+d.q50+d.q75+d.complete > 0
}

// Tick runs one cycle: snapshot the universe, drain counters, upsert,
// reconcile budgets. Returns the first non-recoverable error encountered.
func (d *Drainer) Tick(ctx context.Context) error {
	snap, err := registry.Load(ctx, d.pool)
	if err != nil {
		return fmt.Errorf("registry load: %w", err)
	}
	if len(snap.Ads) == 0 {
		log.Debug().Msg("worker tick: no active ads")
		return nil
	}

	dates := datesToDrain(time.Now().UTC())
	deltas := make(map[string]*statsDelta) // key = adID + "|" + date

	for _, ad := range snap.Ads {
		for _, date := range dates {
			key := ad.ID.String() + "|" + date
			delta := deltas[key]
			if delta == nil {
				delta = &statsDelta{campaignID: ad.CampaignID, adID: ad.ID, date: date}
				deltas[key] = delta
			}
			for _, ev := range d.events {
				redisKey := "ad:" + ad.ID.String() + ":event:" + ev + ":" + date
				val, err := d.redis.GetDel(ctx, redisKey).Int64()
				if err == redis.Nil {
					continue
				}
				if err != nil {
					log.Warn().Err(err).Str("key", redisKey).Msg("worker tick: GETDEL failed")
					continue
				}
				switch ev {
				case tracking.EventImpression:
					delta.imps += val
				case tracking.EventClick:
					delta.clk += val
				case tracking.EventStart:
					delta.start += val
				case tracking.EventFirstQuartile:
					delta.q25 += val
				case tracking.EventMidpoint:
					delta.q50 += val
				case tracking.EventThirdQuartile:
					delta.q75 += val
				case tracking.EventComplete:
					delta.complete += val
				}
			}
		}
	}

	if err := d.upsertDeltas(ctx, deltas); err != nil {
		return fmt.Errorf("upsert daily_stats: %w", err)
	}

	if err := d.reconcileBudgets(ctx, snap); err != nil {
		log.Warn().Err(err).Msg("worker tick: reconcile budgets failed")
	}

	return nil
}

// datesToDrain returns today + yesterday in UTC. Bounded window so worker
// work per tick is O(ads × events × 2), not O(ads × events × history).
func datesToDrain(now time.Time) []string {
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	return []string{today, yesterday}
}

const upsertSQL = `
INSERT INTO daily_stats (campaign_id, ad_id, date,
                         impressions, clicks, start_count, q25, q50, q75, complete,
                         updated_at)
VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8, $9, $10, now())
ON CONFLICT (ad_id, date) DO UPDATE SET
  impressions = daily_stats.impressions + EXCLUDED.impressions,
  clicks      = daily_stats.clicks      + EXCLUDED.clicks,
  start_count = daily_stats.start_count + EXCLUDED.start_count,
  q25         = daily_stats.q25         + EXCLUDED.q25,
  q50         = daily_stats.q50         + EXCLUDED.q50,
  q75         = daily_stats.q75         + EXCLUDED.q75,
  complete    = daily_stats.complete    + EXCLUDED.complete,
  updated_at  = now()
`

func (d *Drainer) upsertDeltas(ctx context.Context, deltas map[string]*statsDelta) error {
	written := 0
	for _, delta := range deltas {
		if !delta.anyNonZero() {
			continue
		}
		if _, err := d.pool.Exec(ctx, upsertSQL,
			delta.campaignID, delta.adID, delta.date,
			delta.imps, delta.clk, delta.start, delta.q25, delta.q50, delta.q75, delta.complete,
		); err != nil {
			return err
		}
		written++
	}
	if written > 0 {
		log.Info().Int("rows", written).Msg("worker tick: daily_stats updated")
	}
	return nil
}

// reconcileBudgets pauses campaigns that have spent their total_budget.
// Reads the live Redis counter for every campaign in the snapshot; runs an
// UPDATE for the ones that crossed the cap. Publishes a registry
// invalidation so the running adserver sees the new "completed" status
// without waiting for the TTL tick.
func (d *Drainer) reconcileBudgets(ctx context.Context, snap *registry.Snapshot) error {
	seen := make(map[uuid.UUID]int32)
	for _, ad := range snap.Ads {
		if ad.BudgetTotal <= 0 {
			continue
		}
		if _, ok := seen[ad.CampaignID]; ok {
			continue
		}
		seen[ad.CampaignID] = ad.BudgetTotal
	}
	if len(seen) == 0 {
		return nil
	}

	paused := 0
	for cmpID, cap := range seen {
		key := "campaign:" + cmpID.String() + ":imps_total"
		count, err := d.redis.Get(ctx, key).Int64()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			log.Warn().Err(err).Str("key", key).Msg("worker tick: reading budget counter failed")
			continue
		}
		if count < int64(cap) {
			continue
		}
		tag, err := d.pool.Exec(ctx,
			`UPDATE campaigns SET status='completed', updated_at=now()
			 WHERE id=$1 AND status='active'`,
			cmpID)
		if err != nil {
			log.Warn().Err(err).Str("campaign", cmpID.String()).Msg("worker tick: pause UPDATE failed")
			continue
		}
		if tag.RowsAffected() > 0 {
			paused++
			log.Info().Str("campaign", cmpID.String()).Int64("imps", count).Int32("cap", cap).Msg("campaign paused (budget exhausted)")
		}
	}
	if paused > 0 {
		if _, err := d.redis.Publish(ctx, registry.InvalidateChannel, "worker:budget-pause").Result(); err != nil {
			log.Warn().Err(err).Msg("worker tick: publish invalidate failed")
		}
	}
	return nil
}
