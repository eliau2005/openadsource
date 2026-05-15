package registry

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func makeTestSnapshot(t *testing.T, n int) *Snapshot {
	t.Helper()
	ads := make([]*Ad, n)
	ends := make([]*time.Time, n)
	for i := 0; i < n; i++ {
		ads[i] = &Ad{
			ID:           uuid.New(),
			CampaignID:   uuid.New(),
			Name:         "ad",
			PositionType: "pre",
			Priority:     1,
			MediaSource:  "external_url",
			MediaURL:     "https://example.com/clip.mp4",
			MediaMime:    "video/mp4",
		}
		if i%2 == 0 {
			ads[i].Countries = []string{"US"}
		} else {
			ads[i].Devices = []string{"mobile"}
		}
	}
	return buildSnapshot(ads, ends, time.Now())
}

func TestBuildSnapshot_WildcardFolding(t *testing.T) {
	ads := []*Ad{
		{ID: uuid.New(), CampaignID: uuid.New(), PositionType: "pre", Priority: 1},                            // wildcard everything
		{ID: uuid.New(), CampaignID: uuid.New(), PositionType: "pre", Priority: 1, Countries: []string{"US"}}, // US only
		{ID: uuid.New(), CampaignID: uuid.New(), PositionType: "pre", Priority: 1, Countries: []string{"IL"}}, // IL only
	}
	ends := []*time.Time{nil, nil, nil}
	snap := buildSnapshot(ads, ends, time.Now())

	if got := snap.ByCountry["US"].PopCount(); got != 2 {
		t.Errorf("US bucket: want 2 (US-only + wildcard), got %d", got)
	}
	if got := snap.ByCountry["IL"].PopCount(); got != 2 {
		t.Errorf("IL bucket: want 2 (IL-only + wildcard), got %d", got)
	}
	if got := snap.ByCountry[WildcardKey].PopCount(); got != 1 {
		t.Errorf("wildcard bucket: want 1, got %d", got)
	}
}

func TestMatchingCountry_FallsBackToWildcard(t *testing.T) {
	ads := []*Ad{
		{ID: uuid.New(), CampaignID: uuid.New(), PositionType: "pre", Priority: 1},
		{ID: uuid.New(), CampaignID: uuid.New(), PositionType: "pre", Priority: 1, Countries: []string{"US"}},
	}
	ends := []*time.Time{nil, nil}
	snap := buildSnapshot(ads, ends, time.Now())

	// Request from FR (not explicitly targeted) → wildcard fallback, 1 ad
	if got := snap.MatchingCountry("FR").PopCount(); got != 1 {
		t.Errorf("FR (unknown): want fallback to wildcard=1, got %d", got)
	}
	// Request from US → wildcard + US = 2 ads
	if got := snap.MatchingCountry("US").PopCount(); got != 2 {
		t.Errorf("US: want 2, got %d", got)
	}
	// Empty (unknown country) → wildcard only
	if got := snap.MatchingCountry("").PopCount(); got != 1 {
		t.Errorf("empty: want wildcard=1, got %d", got)
	}
}

// TestSnapshotSwap_Race verifies that concurrent atomic Load() against the
// Refresher pointer while a writer Store()s rapidly never returns nil or a
// torn struct. Run with `go test -race` on a CGO-enabled platform to actually
// detect races; on Windows-without-CGO the test still functionally exercises
// the pattern.
func TestSnapshotSwap_Race(t *testing.T) {
	var ptr atomic.Pointer[Snapshot]
	ptr.Store(makeTestSnapshot(t, 10))

	const readers = 8
	const writes = 1000

	var wg sync.WaitGroup
	stop := make(chan struct{})

	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					snap := ptr.Load()
					if snap == nil {
						t.Error("reader saw nil snapshot")
						return
					}
					if len(snap.Ads) == 0 {
						t.Error("reader saw empty Ads")
						return
					}
				}
			}
		}()
	}

	// Single writer drives the swap loop, then signals readers to stop.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < writes; i++ {
			ptr.Store(makeTestSnapshot(t, 10+i%5))
		}
		close(stop)
	}()

	wg.Wait()
}
