package bloom

import (
	"runtime"
	"sync/atomic"
	"time"
)

// doorkeeper is a lightweight, Bloom-like admission filter.
// It tracks "probably seen" keys using k independent bit positions per key.
// We periodically reset() the table to keep FPR bounded under churn.
//
// Thread-safety: all operations are lock-free and use atomic loads/CAS.
type doorkeeper struct {
	bits []uint64 // packed bit-array (64 bits per word)
	mask uint32   // index mask: (numBitsRoundedToPow2 - 1)
}

// init prepares a bit-array sized to the next power of two, so we can
// index with a cheap bitmask (h & mask). totalBits may be any positive value.
func (d *doorkeeper) init(totalBits uint32) {
	if totalBits == 0 {
		totalBits = 1 // keep structure valid; nextPow2(1) == 1
	}
	n := nextPow2(int(totalBits)) // round up to power of two
	wordCount := (n + 63) / 64    // 64 bits per uint64 word
	d.bits = make([]uint64, wordCount)
	d.mask = uint32(n - 1)
}

// reset clears all bits (best-effort full reset). This is O(len(bits))
// and should be called infrequently (e.g., on aging window boundary).
func (d *doorkeeper) reset() {
	for i := range d.bits {
		atomic.StoreUint64(&d.bits[i], 0)
	}
}

// probablySeen returns true if all k (here: 3) probed bits are set.
// This is a read-only check and does not modify the table.
//
// NOTE: mix64 is SplitMix64 mixing (see sketch.go); it provides
// well-diffused, pseudo-independent indices from the same 64-bit seed.
func (d *doorkeeper) probablySeen(h uint64) bool {
	i0 := uint32(h) & d.mask
	h = mix64(h)
	i1 := uint32(h) & d.mask
	h = mix64(h)
	i2 := uint32(h) & d.mask
	return d.get(i0) && d.get(i1) && d.get(i2)
}

// seenOrAdd returns true if the key was probably seen already. Otherwise,
// it sets the k bits and returns false. This is the common admission path.
func (d *doorkeeper) seenOrAdd(h uint64) bool {
	i0 := uint32(h) & d.mask
	h = mix64(h)
	i1 := uint32(h) & d.mask
	h = mix64(h)
	i2 := uint32(h) & d.mask

	b0 := d.get(i0)
	b1 := d.get(i1)
	b2 := d.get(i2)
	if b0 && b1 && b2 {
		return true
	}
	d.set(i0)
	d.set(i1)
	d.set(i2)
	return false
}

// wordBit maps a flat bit index to (wordIndex, bitMask) within d.bits.
func (d *doorkeeper) wordBit(i uint32) (uint32, uint64) {
	w := i >> 6                // i / 64
	b := uint64(1) << (i & 63) // 1 << (i % 64)
	return w, b
}

// get atomically checks if a single bit is set.
func (d *doorkeeper) get(i uint32) bool {
	w, b := d.wordBit(i)
	v := atomic.LoadUint64(&d.bits[w])
	return (v & b) != 0
}

// set atomically sets a single bit using a bounded CAS loop.
// If many goroutines race on the same word, we cooperatively yield
// to avoid hot spinning (bounded to maxCASTries attempts).
func (d *doorkeeper) set(i uint32) {
	w, b := d.wordBit(i)
	ptr := &d.bits[w]

	for tries := 1; tries <= maxCASTries; tries++ {
		old := atomic.LoadUint64(ptr)
		neu := old | b
		// Fast path: already set or CAS succeeds.
		if neu == old || atomic.CompareAndSwapUint64(ptr, old, neu) {
			return
		}
		// Cooperative backoff:
		if tries%yieldEveryTries == 0 {
			runtime.Gosched()
			if tries >= sleepAfterTries {
				time.Sleep(0)
			}
		}
	}
	// Best-effort semantics: if we fail all attempts, we give up.
	// This is acceptable for a probabilistic pre-admission filter.
}
