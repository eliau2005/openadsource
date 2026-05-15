package selection

import (
	"math"

	"github.com/eliau2005/openadsource/server/internal/registry"
)

// topPriority compacts indices in place to keep only ads whose Priority
// equals the maximum among the slice. Returns the new length. Single pass
// to find the max, second pass to compact — both linear, no allocation.
func topPriority(snap *registry.Snapshot, indices []int) []int {
	if len(indices) == 0 {
		return indices
	}
	maxPri := int32(math.MinInt32)
	for _, idx := range indices {
		if p := snap.Ads[idx].Priority; p > maxPri {
			maxPri = p
		}
	}
	n := 0
	for _, idx := range indices {
		if snap.Ads[idx].Priority == maxPri {
			indices[n] = idx
			n++
		}
	}
	return indices[:n]
}
