// Package registry holds the in-memory snapshot of active ads + targeting +
// budgets that the decision engine reads on every /vast request. Loaded
// off-path and atomically swapped, never read with a lock.
package registry

import "math/bits"

// Bitset is a hand-rolled fixed-width bitset backed by []uint64. Operations
// are allocation-free when callers reuse the destination slice. Layout:
// bit i lives at word i>>6, mask 1<<(i&63).
type Bitset []uint64

// NewBitset returns a freshly allocated bitset large enough for n bits.
func NewBitset(n int) Bitset {
	if n <= 0 {
		return nil
	}
	return make(Bitset, (n+63)>>6)
}

// Words returns the underlying slice. Useful for tests and benchmarks.
func (b Bitset) Words() []uint64 { return b }

// Set turns bit i on. Caller is responsible for ensuring i is within range.
func (b Bitset) Set(i int) { b[i>>6] |= 1 << uint(i&63) }

// IsSet returns true if bit i is on.
func (b Bitset) IsSet(i int) bool { return b[i>>6]&(1<<uint(i&63)) != 0 }

// Clear zeroes every word in the bitset. O(n) but cache-friendly.
func (b Bitset) Clear() {
	for i := range b {
		b[i] = 0
	}
}

// AndInto writes (a AND b) into dst. dst must already be large enough; all
// three slices must share the same length. No allocation.
func AndInto(a, b, dst Bitset) {
	for i := range dst {
		dst[i] = a[i] & b[i]
	}
}

// OrInto writes (a OR b) into dst.
func OrInto(a, b, dst Bitset) {
	for i := range dst {
		dst[i] = a[i] | b[i]
	}
}

// CopyInto duplicates src into dst.
func CopyInto(src, dst Bitset) {
	copy(dst, src)
}

// AndAssign updates dst in place: dst = dst AND src.
func AndAssign(dst, src Bitset) {
	for i := range dst {
		dst[i] &= src[i]
	}
}

// OrAssign updates dst in place: dst = dst OR src.
func OrAssign(dst, src Bitset) {
	for i := range dst {
		dst[i] |= src[i]
	}
}

// PopCount counts the number of set bits across the whole slice.
func (b Bitset) PopCount() int {
	n := 0
	for _, w := range b {
		n += bits.OnesCount64(w)
	}
	return n
}

// ForEach calls fn for each set bit index, in ascending order. No
// allocation; the bit-by-bit traversal uses TrailingZeros64 so it skips
// empty words in O(1).
func (b Bitset) ForEach(fn func(i int)) {
	for w := 0; w < len(b); w++ {
		x := b[w]
		for x != 0 {
			tz := bits.TrailingZeros64(x)
			fn(w<<6 + tz)
			x &= x - 1
		}
	}
}
