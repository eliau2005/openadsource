package selection

import (
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/eliau2005/openadsource/server/internal/registry"
)

func mustSnapshot(t *testing.T, ads []*registry.Ad) *registry.Snapshot {
	t.Helper()
	endDates := make([]*time.Time, len(ads))
	return registry.BuildSnapshotForTest(ads, endDates, time.Now())
}

func TestSelect_CountryFilter(t *testing.T) {
	usOnly := &registry.Ad{
		ID: uuid.New(), CampaignID: uuid.New(), Name: "us-only",
		PositionType: "pre", Priority: 1,
		MediaSource: "external_url", MediaURL: "https://x/us.mp4", MediaMime: "video/mp4",
		Countries: []string{"US"}, PacingWeight: 1,
	}
	ilOnly := &registry.Ad{
		ID: uuid.New(), CampaignID: uuid.New(), Name: "il-only",
		PositionType: "pre", Priority: 1,
		MediaSource: "external_url", MediaURL: "https://x/il.mp4", MediaMime: "video/mp4",
		Countries: []string{"IL"}, PacingWeight: 1,
	}
	snap := mustSnapshot(t, []*registry.Ad{usOnly, ilOnly})

	if got := Select(snap, Request{Pos: "pre", Country: "US"}); got != usOnly {
		t.Errorf("US request: want us-only, got %v", got)
	}
	if got := Select(snap, Request{Pos: "pre", Country: "IL"}); got != ilOnly {
		t.Errorf("IL request: want il-only, got %v", got)
	}
	if got := Select(snap, Request{Pos: "pre", Country: "FR"}); got != nil {
		t.Errorf("FR request: want no-fill, got %v", got)
	}
}

func TestSelect_WildcardCountryFallback(t *testing.T) {
	wildcard := &registry.Ad{
		ID: uuid.New(), CampaignID: uuid.New(), Name: "anyone",
		PositionType: "pre", Priority: 1,
		MediaSource: "external_url", MediaURL: "https://x/w.mp4", MediaMime: "video/mp4",
		PacingWeight: 1,
	}
	snap := mustSnapshot(t, []*registry.Ad{wildcard})
	if got := Select(snap, Request{Pos: "pre", Country: "ZZ"}); got != wildcard {
		t.Errorf("unknown country should match wildcard, got %v", got)
	}
}

func TestSelect_PriorityWins(t *testing.T) {
	low := &registry.Ad{
		ID: uuid.New(), CampaignID: uuid.New(), Name: "low",
		PositionType: "pre", Priority: 1, PacingWeight: 100,
	}
	high := &registry.Ad{
		ID: uuid.New(), CampaignID: uuid.New(), Name: "high",
		PositionType: "pre", Priority: 10, PacingWeight: 1,
	}
	snap := mustSnapshot(t, []*registry.Ad{low, high})

	for i := 0; i < 50; i++ {
		got := Select(snap, Request{Pos: "pre", Country: "US"})
		if got != high {
			t.Errorf("higher priority must always win, got %v", got)
		}
	}
}

func TestSelect_MidRollOffsetTolerance(t *testing.T) {
	at10 := &registry.Ad{
		ID: uuid.New(), CampaignID: uuid.New(), Name: "at10",
		PositionType: "mid", MidRollOffset: 10, Priority: 1, PacingWeight: 1,
	}
	at30 := &registry.Ad{
		ID: uuid.New(), CampaignID: uuid.New(), Name: "at30",
		PositionType: "mid", MidRollOffset: 30, Priority: 1, PacingWeight: 1,
	}
	snap := mustSnapshot(t, []*registry.Ad{at10, at30})

	if got := Select(snap, Request{Pos: "mid", Offset: 9}); got != at10 {
		t.Errorf("offset 9 (within ±2 of 10): want at10, got %v", got)
	}
	if got := Select(snap, Request{Pos: "mid", Offset: 30}); got != at30 {
		t.Errorf("offset 30: want at30, got %v", got)
	}
	if got := Select(snap, Request{Pos: "mid", Offset: 20}); got != nil {
		t.Errorf("offset 20 (outside both windows): want no-fill, got %v", got)
	}
}

func TestSelect_PropertyTargetingHolds(t *testing.T) {
	// Build a synthetic snapshot of 100 ads with random targeting, fire
	// 1000 random requests, assert that any ad returned actually matches
	// the request's targeting.
	r := rand.New(rand.NewSource(42))
	countries := []string{"US", "IL", "FR", "DE", "JP", ""}
	devices := []string{"mobile", "tablet", "desktop", "ctv", ""}
	ads := make([]*registry.Ad, 100)
	for i := range ads {
		ad := &registry.Ad{
			ID:           uuid.New(),
			CampaignID:   uuid.New(),
			PositionType: "pre",
			Priority:     1,
			PacingWeight: 1,
		}
		// 50% wildcard country, 50% pinned to one
		if r.Intn(2) == 0 {
			ad.Countries = []string{countries[r.Intn(len(countries)-1)]}
		}
		if r.Intn(2) == 0 {
			ad.Devices = []string{devices[r.Intn(len(devices)-1)]}
		}
		ads[i] = ad
	}
	snap := mustSnapshot(t, ads)

	for i := 0; i < 1000; i++ {
		req := Request{
			Pos:     "pre",
			Country: countries[r.Intn(len(countries))],
			Device:  devices[r.Intn(len(devices))],
			Rand:    r,
		}
		got := Select(snap, req)
		if got == nil {
			continue
		}
		// Country must match: either wildcard or explicit equal.
		if got.Countries != nil {
			ok := false
			for _, c := range got.Countries {
				if c == req.Country {
					ok = true
					break
				}
			}
			if !ok {
				t.Errorf("returned ad targets %v but request was %q", got.Countries, req.Country)
			}
		}
		if got.Devices != nil {
			ok := false
			for _, d := range got.Devices {
				if d == req.Device {
					ok = true
					break
				}
			}
			if !ok {
				t.Errorf("returned ad targets %v but request was %q", got.Devices, req.Device)
			}
		}
	}
}
