package selection

import (
	"math/rand"
	"sort"

	"github.com/eliau2005/openadsource/server/internal/registry"
)

// Request captures the bits of an incoming /vast call that the selector
// needs. All fields are pre-normalised by the handler (uppercased country,
// lowercased device, defaulted position).
type Request struct {
	Pos     string // "pre" | "mid" | "post"
	Offset  int32  // seconds, only meaningful for Pos=="mid"
	Country string // ISO-3166 alpha-2 or ""
	Device  string // mobile|tablet|desktop|ctv or ""

	// Exclude is consulted before each candidate is returned. Lets the
	// handler retry after a Redis budget DECR rejection without re-running
	// selection from scratch (the rejected candidate is added to this set
	// and the same Scratch is re-evaluated). nil = no exclusions.
	Exclude map[int]bool

	// Rand can be set in tests to inject a deterministic source. nil =
	// math/rand's global source (good enough for production).
	Rand *rand.Rand
}

// MidRollOffsetTolerance is the ±window in seconds inside which a mid-roll
// ad's offset is considered eligible.
const MidRollOffsetTolerance = 2

// Select picks the ad to serve for this request, or nil if no candidate
// survives filtering. The returned *Ad is a pointer into the snapshot — the
// caller MUST treat it as read-only.
func Select(snap *registry.Snapshot, req Request) *registry.Ad {
	if snap == nil || len(snap.Ads) == 0 {
		return nil
	}
	posBits := snap.MatchingPosition(req.Pos)
	if posBits == nil {
		return nil
	}
	countryBits := snap.MatchingCountry(req.Country)
	deviceBits := snap.MatchingDevice(req.Device)
	if countryBits == nil || deviceBits == nil {
		return nil
	}

	s := getScratch()
	defer putScratch(s)
	s.reset(snap.BitsetSize, len(snap.Ads))

	// Three-way AND into scratch.intersect — single linear pass per word.
	for i := 0; i < snap.BitsetSize; i++ {
		s.intersect[i] = posBits[i] & countryBits[i] & deviceBits[i]
	}

	// Walk set bits → candidate indices.
	s.intersect.ForEach(func(idx int) {
		if req.Exclude != nil && req.Exclude[idx] {
			return
		}
		if req.Pos == "mid" {
			off := snap.Ads[idx].MidRollOffset
			if off < req.Offset-MidRollOffsetTolerance || off > req.Offset+MidRollOffsetTolerance {
				return
			}
		}
		s.indices = append(s.indices, idx)
	})
	if len(s.indices) == 0 {
		return nil
	}

	// Compact to the top-priority bucket.
	s.indices = topPriority(snap, s.indices)
	if len(s.indices) == 1 {
		return snap.Ads[s.indices[0]]
	}

	// Build cumulative pacing-weight table over the top-priority bucket.
	total := 0.0
	s.cumWeight = s.cumWeight[:0]
	for _, idx := range s.indices {
		w := snap.Ads[idx].PacingWeight
		if w <= 0 {
			w = 1
		}
		total += w
		s.cumWeight = append(s.cumWeight, total)
	}

	// Weighted-random pick via binary search. rand.Float64 returns a value
	// in [0.0, 1.0); multiplying by total puts pick in [0, total). The
	// SearchFloat64s contract returns the smallest i where slice[i] > pick.
	var pick float64
	if req.Rand != nil {
		pick = req.Rand.Float64() * total
	} else {
		pick = rand.Float64() * total
	}
	i := sort.SearchFloat64s(s.cumWeight, pick)
	if i >= len(s.indices) {
		i = len(s.indices) - 1
	}
	return snap.Ads[s.indices[i]]
}

// SelectByID is the test-mode lookup for ?ad_id=<uuid>. It stays a memory
// read so the "zero PG on hot path" contract holds for the test handler
// path as well.
func SelectByID(snap *registry.Snapshot, id [16]byte) *registry.Ad {
	if snap == nil {
		return nil
	}
	return snap.ByID[id]
}
