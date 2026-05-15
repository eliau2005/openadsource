// Package selection picks the ad to serve given a registry.Snapshot and a
// per-request decision context. The hot path uses sync.Pool-backed scratch
// buffers so steady-state requests allocate nothing.
package selection

import (
	"sync"

	"github.com/eliau2005/openadsource/server/internal/registry"
)

// Scratch holds the per-request buffers the selector mutates. Reused across
// requests via the package-level pool — grow in-place when a snapshot is
// larger than the previous one, never shrink.
type Scratch struct {
	intersect registry.Bitset
	indices   []int
	cumWeight []float64
}

// reset makes sure the scratch buffers are at least the sizes we need. The
// indices / cumWeight slices stay capped at adsCount because that's the
// worst case (every ad matches), but the run-length is set to zero so
// callers can simply append.
func (s *Scratch) reset(bitsetWords, adsCount int) {
	if cap(s.intersect) < bitsetWords {
		s.intersect = make(registry.Bitset, bitsetWords)
	} else {
		s.intersect = s.intersect[:bitsetWords]
		for i := range s.intersect {
			s.intersect[i] = 0
		}
	}
	if cap(s.indices) < adsCount {
		s.indices = make([]int, 0, adsCount)
	} else {
		s.indices = s.indices[:0]
	}
	if cap(s.cumWeight) < adsCount {
		s.cumWeight = make([]float64, 0, adsCount)
	} else {
		s.cumWeight = s.cumWeight[:0]
	}
}

var scratchPool = sync.Pool{
	New: func() any { return &Scratch{} },
}

// getScratch / putScratch wrap the pool. Splitting them keeps the call
// sites tidy and gives the compiler a chance to inline.
func getScratch() *Scratch { return scratchPool.Get().(*Scratch) }
func putScratch(s *Scratch) {
	// Keep the capacities; just trim the lengths. This is the whole point
	// of the pool: every subsequent caller reuses the underlying memory.
	s.indices = s.indices[:0]
	s.cumWeight = s.cumWeight[:0]
	scratchPool.Put(s)
}
