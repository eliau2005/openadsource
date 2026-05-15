package registry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// loadQuery returns every active, in-flight ad with its campaign's window /
// budget and per-campaign targeting. One round trip. Run off-path by the
// Refresher; never on the /vast hot path.
const loadQuery = `
SELECT a.id, a.campaign_id, a.name,
       a.position_type, COALESCE(a.mid_roll_offset, 0), a.priority,
       COALESCE(a.landing_page_url, ''),
       a.media_source, a.media_url, a.media_mime,
       COALESCE(a.media_duration_ms, 0),
       COALESCE(a.media_width, 0), COALESCE(a.media_height, 0),
       COALESCE(a.media_bitrate_kbps, 0),
       c.end_date, COALESCE(c.total_budget_impressions, 0),
       ct.countries, ct.devices,
       COALESCE(cr.max_impressions, 0),
       COALESCE(cr.time_window_seconds, 0)
FROM ads a
JOIN campaigns c ON c.id = a.campaign_id
LEFT JOIN campaign_targeting ct ON ct.campaign_id = c.id
LEFT JOIN LATERAL (
    SELECT max_impressions, time_window_seconds
    FROM cap_rules
    WHERE cap_rules.ad_id = a.id
    ORDER BY created_at DESC
    LIMIT 1
) cr ON true
WHERE a.status = 'active'
  AND c.status = 'active'
  AND (c.start_date IS NULL OR c.start_date <= now())
  AND (c.end_date   IS NULL OR c.end_date   >  now())
ORDER BY a.priority DESC, a.id`

// Load reads the active-ads + campaigns + targeting universe and assembles
// a fresh Snapshot. Errors leave the caller's previous snapshot intact —
// the Refresher does not Store(nil) on failure.
func Load(ctx context.Context, pool *pgxpool.Pool) (*Snapshot, error) {
	rows, err := pool.Query(ctx, loadQuery)
	if err != nil {
		return nil, fmt.Errorf("registry load query: %w", err)
	}
	defer rows.Close()

	now := time.Now()
	ads := make([]*Ad, 0, 256)

	// Per-ad campaign end_date so we can compute pacing weights after the
	// scan. nil = unbounded campaign.
	endDates := make([]*time.Time, 0, 256)

	for rows.Next() {
		var (
			ad        Ad
			endDate   *time.Time
			budget    int32
			countries []string
			devices   []string
			capMax    int32
			capWindow int32
		)
		if err := rows.Scan(
			&ad.ID, &ad.CampaignID, &ad.Name,
			&ad.PositionType, &ad.MidRollOffset, &ad.Priority,
			&ad.LandingPageURL,
			&ad.MediaSource, &ad.MediaURL, &ad.MediaMime,
			&ad.MediaDurationMs,
			&ad.MediaWidth, &ad.MediaHeight, &ad.MediaBitrate,
			&endDate, &budget,
			&countries, &devices,
			&capMax, &capWindow,
		); err != nil {
			return nil, fmt.Errorf("registry load scan: %w", err)
		}
		ad.BudgetTotal = budget
		ad.Countries = normaliseCountries(countries)
		ad.Devices = normaliseDevices(devices)
		ad.CapMaxImpressions = capMax
		ad.CapTimeWindowSecs = capWindow
		ads = append(ads, &ad)
		endDates = append(endDates, endDate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("registry load iter: %w", err)
	}

	return buildSnapshot(ads, endDates, now), nil
}

func buildSnapshot(ads []*Ad, endDates []*time.Time, now time.Time) *Snapshot {
	n := len(ads)
	bitsetSize := (n + 63) >> 6
	snap := &Snapshot{
		Ads:        ads,
		BitsetSize: bitsetSize,
		ByPosition: make(map[string]Bitset),
		ByCountry:  make(map[string]Bitset),
		ByDevice:   make(map[string]Bitset),
		ByID:       make(map[uuid.UUID]*Ad, n),
		LoadedAt:   now,
	}

	// Track wildcard rows so we can fold them into every concrete bucket
	// after we've seen every concrete country/device. Otherwise a wildcard
	// ad indexed before a country=US ad would miss the US bucket.
	wildCountry := make([]int, 0, n)
	wildDevice := make([]int, 0, n)

	for i, ad := range ads {
		snap.ByID[ad.ID] = ad

		pos := ad.PositionType
		if pos == "" {
			pos = "pre"
		}
		ensure(snap.ByPosition, pos, bitsetSize).Set(i)

		if len(ad.Countries) == 0 {
			ensure(snap.ByCountry, WildcardKey, bitsetSize).Set(i)
			wildCountry = append(wildCountry, i)
		} else {
			for _, c := range ad.Countries {
				ensure(snap.ByCountry, c, bitsetSize).Set(i)
			}
		}
		if len(ad.Devices) == 0 {
			ensure(snap.ByDevice, WildcardKey, bitsetSize).Set(i)
			wildDevice = append(wildDevice, i)
		} else {
			for _, d := range ad.Devices {
				ensure(snap.ByDevice, d, bitsetSize).Set(i)
			}
		}

		ad.PacingWeight = pacingWeight(ad.BudgetTotal, endDates[i], now)
	}

	// Second pass — every wildcard ad gets its bit added to every concrete
	// country/device bucket so a single map lookup at request time returns
	// "ads targeting this country OR matching any country".
	for k, bs := range snap.ByCountry {
		if k == WildcardKey {
			continue
		}
		for _, i := range wildCountry {
			bs.Set(i)
		}
	}
	for k, bs := range snap.ByDevice {
		if k == WildcardKey {
			continue
		}
		for _, i := range wildDevice {
			bs.Set(i)
		}
	}

	return snap
}

func ensure(m map[string]Bitset, k string, size int) Bitset {
	if bs, ok := m[k]; ok {
		return bs
	}
	bs := NewBitset(size << 6) // size words → size*64 bits
	if bs == nil {
		bs = make(Bitset, size)
	}
	if len(bs) < size {
		bs = make(Bitset, size)
	}
	m[k] = bs
	return bs
}

func pacingWeight(budgetTotal int32, endDate *time.Time, now time.Time) float64 {
	budget := float64(budgetTotal)
	if budget <= 0 {
		budget = 1
	}
	if endDate == nil {
		return budget
	}
	hours := endDate.Sub(now).Hours()
	if hours < 1 {
		hours = 1
	}
	return budget / hours
}

func normaliseCountries(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := in[:0]
	for _, c := range in {
		c = strings.ToUpper(strings.TrimSpace(c))
		if c == "" {
			continue
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normaliseDevices(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := in[:0]
	for _, d := range in {
		d = strings.ToLower(strings.TrimSpace(d))
		switch d {
		case "mobile", "tablet", "desktop", "ctv":
			out = append(out, d)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
