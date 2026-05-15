package selection

import (
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/eliau2005/openadsource/server/internal/registry"
)

// buildBenchSnapshot makes a synthetic snapshot of 1000 campaigns × 5 ads =
// 5000 ads, distributed across positions, 200 distinct country codes, and
// 4 device classes. Half of the ads have wildcard targeting on each
// dimension; the other half is pinned to a random concrete value.
func buildBenchSnapshot(b *testing.B) (*registry.Snapshot, []string) {
	b.Helper()
	r := rand.New(rand.NewSource(1))
	const campaigns = 1000
	const adsPerCampaign = 5
	const numCountries = 200
	devices := []string{"mobile", "tablet", "desktop", "ctv"}
	positions := []string{"pre", "mid", "post"}

	countries := make([]string, numCountries)
	for i := 0; i < numCountries; i++ {
		countries[i] = string(rune('A'+i%26)) + string(rune('A'+(i/26)%26))
	}

	ads := make([]*registry.Ad, 0, campaigns*adsPerCampaign)
	for c := 0; c < campaigns; c++ {
		campID := uuid.New()
		pri := int32(1 + r.Intn(5))
		var camCountries []string
		var camDevices []string
		if r.Intn(2) == 0 {
			camCountries = []string{countries[r.Intn(numCountries)]}
		}
		if r.Intn(2) == 0 {
			camDevices = []string{devices[r.Intn(len(devices))]}
		}
		for j := 0; j < adsPerCampaign; j++ {
			pos := positions[r.Intn(len(positions))]
			off := int32(0)
			if pos == "mid" {
				off = int32(5 + r.Intn(50))
			}
			ads = append(ads, &registry.Ad{
				ID:           uuid.New(),
				CampaignID:   campID,
				Name:         "ad",
				PositionType: pos,
				MidRollOffset: off,
				Priority:     pri,
				MediaSource:  "external_url",
				MediaURL:     "https://example.com/x.mp4",
				MediaMime:    "video/mp4",
				Countries:    camCountries,
				Devices:      camDevices,
				PacingWeight: 1.0 + r.Float64(),
			})
		}
	}
	endDates := make([]*time.Time, len(ads))
	return registry.BuildSnapshotForTest(ads, endDates, time.Now()), countries
}

// BenchmarkSelect_1k_campaigns is the hard contract from ROADMAP §Phase 3:
// against a synthetic 1000-campaign × 5-ad snapshot (5000 ads, 200
// countries, 4 devices), Select must run in well under 50 µs/op with no
// more than 2 allocations per op.
func BenchmarkSelect_1k_campaigns(b *testing.B) {
	snap, countries := buildBenchSnapshot(b)
	devices := []string{"mobile", "tablet", "desktop", "ctv"}
	r := rand.New(rand.NewSource(7))

	// Warm up the scratch pool so the first iteration isn't penalised for
	// growing the bitset buffer to fit the snapshot.
	for i := 0; i < 10; i++ {
		_ = Select(snap, Request{
			Pos: "pre", Country: countries[r.Intn(len(countries))], Device: devices[r.Intn(len(devices))],
		})
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Select(snap, Request{
			Pos:     "pre",
			Country: countries[r.Intn(len(countries))],
			Device:  devices[r.Intn(len(devices))],
		})
	}
}

// TestSelect_AllocsBudget asserts the ≤2 allocs/op contract directly so
// regressions break CI even when no one runs `go test -bench`.
func TestSelect_AllocsBudget(t *testing.T) {
	snap, countries := buildBenchSnapshotForTest(t)
	devices := []string{"mobile", "tablet", "desktop", "ctv"}
	r := rand.New(rand.NewSource(7))

	// Warm up scratch.
	for i := 0; i < 100; i++ {
		_ = Select(snap, Request{
			Pos: "pre", Country: countries[r.Intn(len(countries))], Device: devices[r.Intn(len(devices))],
		})
	}

	avg := testing.AllocsPerRun(2000, func() {
		Select(snap, Request{
			Pos:     "pre",
			Country: countries[r.Intn(len(countries))],
			Device:  devices[r.Intn(len(devices))],
		})
	})
	if avg > 2.0 {
		t.Errorf("Select allocates %.2f allocs/op; ROADMAP cap is 2", avg)
	}
	t.Logf("Select: %.2f allocs/op (warm)", avg)
}

func buildBenchSnapshotForTest(t *testing.T) (*registry.Snapshot, []string) {
	t.Helper()
	r := rand.New(rand.NewSource(1))
	const campaigns = 1000
	const adsPerCampaign = 5
	const numCountries = 200
	devices := []string{"mobile", "tablet", "desktop", "ctv"}
	positions := []string{"pre", "mid", "post"}
	countries := make([]string, numCountries)
	for i := 0; i < numCountries; i++ {
		countries[i] = string(rune('A'+i%26)) + string(rune('A'+(i/26)%26))
	}
	ads := make([]*registry.Ad, 0, campaigns*adsPerCampaign)
	for c := 0; c < campaigns; c++ {
		campID := uuid.New()
		pri := int32(1 + r.Intn(5))
		var camCountries []string
		var camDevices []string
		if r.Intn(2) == 0 {
			camCountries = []string{countries[r.Intn(numCountries)]}
		}
		if r.Intn(2) == 0 {
			camDevices = []string{devices[r.Intn(len(devices))]}
		}
		for j := 0; j < adsPerCampaign; j++ {
			pos := positions[r.Intn(len(positions))]
			off := int32(0)
			if pos == "mid" {
				off = int32(5 + r.Intn(50))
			}
			ads = append(ads, &registry.Ad{
				ID: uuid.New(), CampaignID: campID, Name: "ad",
				PositionType: pos, MidRollOffset: off, Priority: pri,
				MediaSource: "external_url", MediaURL: "https://example.com/x.mp4", MediaMime: "video/mp4",
				Countries: camCountries, Devices: camDevices,
				PacingWeight: 1.0 + r.Float64(),
			})
		}
	}
	endDates := make([]*time.Time, len(ads))
	return registry.BuildSnapshotForTest(ads, endDates, time.Now()), countries
}
