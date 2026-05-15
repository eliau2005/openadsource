package registry

import (
	"time"

	"github.com/google/uuid"
)

// Snapshot is the immutable read view consumed by the decision engine.
// Producers (the loader) build a fresh Snapshot off-path and atomically
// swap the Refresher's pointer; readers see a consistent view via a single
// atomic load.
//
// Bitset semantics: ByPosition / ByCountry / ByDevice are all sized
// BitsetSize words and indexed by ad position in the Ads slice. A bit set
// in ByCountry["US"] means "ad at index i is eligible when country=US".
// WildcardKey ("*") is the fallback bucket for ads whose targeting array
// was NULL on that dimension.
type Snapshot struct {
	Ads        []*Ad
	BitsetSize int // number of uint64 words per bitset

	ByPosition map[string]Bitset
	ByCountry  map[string]Bitset
	ByDevice   map[string]Bitset

	ByID map[uuid.UUID]*Ad // direct test-mode lookup (?ad_id=)

	LoadedAt time.Time
}

// MatchingCountry returns the bitset of ads eligible for the given ISO
// country code. Unknown countries fall back to the wildcard bucket so a
// stub GeoIP resolver still returns the wildcard-targeted ads (which is the
// useful behaviour: a campaign with NULL countries genuinely matches every
// request, including the unknown ones).
func (s *Snapshot) MatchingCountry(iso string) Bitset {
	if iso == "" {
		return s.ByCountry[WildcardKey]
	}
	if b, ok := s.ByCountry[iso]; ok {
		return b
	}
	return s.ByCountry[WildcardKey]
}

// MatchingDevice mirrors MatchingCountry but for device class.
func (s *Snapshot) MatchingDevice(device string) Bitset {
	if device == "" {
		return s.ByDevice[WildcardKey]
	}
	if b, ok := s.ByDevice[device]; ok {
		return b
	}
	return s.ByDevice[WildcardKey]
}

// MatchingPosition is a straight map lookup with no wildcard fallback —
// every ad belongs to exactly one position type.
func (s *Snapshot) MatchingPosition(pos string) Bitset {
	return s.ByPosition[pos]
}
