package bloom

import (
	"runtime"
	"sync/atomic"
	"time"
)

// sketch is a TinyLFU-style Count-Min Sketch using 4-bit (nibble) counters.
// Each uint64 holds 16 packed nibbles. For each key, we touch 4 independent
// indices derived from a single 64-bit hash. Aging halves all counters when
// the logical window (adds) passes a threshold (resetAt).
type sketch struct {
	// words holds packed 4-bit counters: 16 counters per uint64.
	// Total counters = 16 * len(words) == numCounters.
	words []uint64

	// mask is numCounters-1; numCounters must be a power of two.
	mask uint32

	// adds is the total number of successful increments that passed admission.
	adds atomic.Uint64

	// resetAt defines the logical aging window: when adds >= resetAt, we age.
	resetAt uint64

	// agingActive is a best-effort guard to avoid concurrent full-table aging.
	agingActive atomic.Uint32
}

const (
	nibbleMask    = 0xF                // one 4-bit lane mask
	maskNibbles64 = 0x7777777777777777 // keeps nibble boundaries after right-shift

	// Bounded CAS retry policy:
	maxCASTries     = 64
	yieldEveryTries = 8  // call Gosched() every 8 failed CAS attempts
	sleepAfterTries = 32 // after 32 tries, also Sleep(0) every 8 attempts

	defaultSamples = 10 // if sampleMultiplier is 0, use this fallback
)

// init allocates the table with length tableLenPow2 (must be power of two).
// sampleMultiplier controls the logical window size: resetAt = sampleMultiplier * numCounters.
func (s *sketch) init(tableLenPow2 uint32, sampleMultiplier uint32) {
	if tableLenPow2 == 0 || (tableLenPow2&(tableLenPow2-1)) != 0 {
		panic("sketch: tableLen must be power-of-two and > 0")
	}

	numCounters := uint64(tableLenPow2)
	wordCount := (numCounters + 15) / 16 // 16 nibbles per uint64
	s.words = make([]uint64, wordCount)
	s.mask = uint32(numCounters - 1)

	if sampleMultiplier == 0 {
		sampleMultiplier = defaultSamples
	}
	s.resetAt = uint64(sampleMultiplier) * numCounters
}

// increment bumps 4 counters chosen by 4 mixed indices (min-of-4 scheme).
// Each nibble saturates at 15. The operation is lock-free and uses bounded
// CAS loops; under heavy contention we may drop an increment (acceptable for TinyLFU).
func (s *sketch) increment(h uint64) {
	s.maybeReset()

	i0 := uint32(h) & s.mask // eq. ->  uint32(h) % (s.mask + 1) -> bit shifting is just faster
	h = mix64(h)
	i1 := uint32(h) & s.mask
	h = mix64(h)
	i2 := uint32(h) & s.mask
	h = mix64(h)
	i3 := uint32(h) & s.mask

	s.incAt(i0)
	s.incAt(i1)
	s.incAt(i2)
	s.incAt(i3)

	s.adds.Add(1)
}

// estimate returns the min of 4 counters for the 4 mixed indices of hash h.
// This is a non-blocking read (atomic loads only).
func (s *sketch) estimate(h uint64) uint8 {
	i0 := uint32(h) & s.mask
	h = mix64(h)
	i1 := uint32(h) & s.mask
	h = mix64(h)
	i2 := uint32(h) & s.mask
	h = mix64(h)
	i3 := uint32(h) & s.mask

	c0 := s.getAt(i0)
	c1 := s.getAt(i1)
	if c1 < c0 {
		c0 = c1
	}
	c2 := s.getAt(i2)
	if c2 < c0 {
		c0 = c2
	}
	c3 := s.getAt(i3)
	if c3 < c0 {
		c0 = c3
	}
	return c0
}

// incAt increments a single 4-bit lane at index idx, saturating at 15.
// Uses a bounded CAS retry loop with cooperative yielding to avoid spinning.
func (s *sketch) incAt(idx uint32) {
	w, sh := s.wordShift(idx)
	ptr := &s.words[w]

	for tries := 1; tries <= maxCASTries; tries++ {
		old := atomic.LoadUint64(ptr)
		n := (old >> sh) & nibbleMask
		if n == nibbleMask {
			return // already saturated (15)
		}
		neu := old + (1 << sh) // add exactly into our nibble

		if atomic.CompareAndSwapUint64(ptr, old, neu) {
			return
		}

		// Cooperative backoff: yield every N tries; after threshold also Sleep(0).
		if tries%yieldEveryTries == 0 {
			runtime.Gosched()
			if tries >= sleepAfterTries {
				time.Sleep(0)
			}
		}
	}
	// Give up after bounded attempts (lossy by design under contention).
}

// getAt reads a single 4-bit lane at index idx.
func (s *sketch) getAt(idx uint32) uint8 {
	w, sh := s.wordShift(idx)
	val := atomic.LoadUint64(&s.words[w])
	return uint8((val >> sh) & nibbleMask)
}

// wordShift maps a counter index to (word index, bit shift) inside words[].
func (s *sketch) wordShift(idx uint32) (uint32, uint) {
	// 16 nibbles per word => word = idx / 16, shift = (idx % 16) * 4
	return idx >> 4, uint((idx & 0xF) << 2)
}

// maybeReset triggers aging once per window in a best-effort manner.
// Exactly one goroutine performs reset(); others continue without blocking.
func (s *sketch) maybeReset() {
	if s.adds.Load() < s.resetAt {
		return
	}
	if s.agingActive.CompareAndSwap(0, 1) {
		// Double-check under the guard to avoid redundant resets.
		if s.adds.Load() >= s.resetAt {
			s.reset()
			s.adds.Store(0)
		}
		s.agingActive.Store(0)
	}
}

// reset halves all 4-bit lanes: new = (old >> 1) & maskNibbles64.
// Each word is updated via a bounded CAS loop with cooperative yielding.
// If we fail to CAS a hot word within the bound, we skip it (best-effort aging).
func (s *sketch) reset() {
	for i := range s.words {
		ptr := &s.words[i]

		done := false
		for tries := 1; tries <= maxCASTries; tries++ {
			old := atomic.LoadUint64(ptr)
			neu := (old >> 1) & maskNibbles64
			if atomic.CompareAndSwapUint64(ptr, old, neu) {
				done = true
				break
			}
			if tries%yieldEveryTries == 0 {
				runtime.Gosched()
				if tries >= sleepAfterTries {
					time.Sleep(0)
				}
			}
		}
		_ = done // best-effort: skipping a word under extreme contention is acceptable
	}
}
