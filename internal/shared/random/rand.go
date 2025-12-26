package random

import (
	"runtime"
	"sync/atomic"
	"time"
)

type shard struct {
	// SplitMix64 64-bit state. Updated via atomic CAS.
	state uint64
}

var (
	_shards []shard
	_mask   uint32
	_rr     uint32 // round-robin counter
)

// Init optionally reconfigures shard count. If n<=0, it uses GOMAXPROCS*4.
// Shard count is rounded up to power of two for a cheap mask.
func Init(n int) {
	if n <= 0 {
		n = runtime.GOMAXPROCS(0) * 4
		if n < 1 {
			n = 1
		}
	}
	// round to pow2
	p := 1
	for p < n {
		p <<= 1
	}
	n = p

	_shards = make([]shard, n)
	_mask = uint32(n - 1)

	seed := splitmixSeed(time.Now().UnixNano())
	for i := range _shards {
		// Different, well-mixed seeds per shard.
		_shards[i].state = splitmixNext(&seed)
		if _shards[i].state == 0 {
			_shards[i].state = 0x9e3779b97f4a7c15
		}
	}
	atomic.StoreUint32(&_rr, 0)
}

// Float64 returns a uniform in [0,1) using 53 random bits (double precision).
func Float64() float64 {
	i := atomic.AddUint32(&_rr, 1) & _mask
	x := splitmixNext(&_shards[i].state)
	// take top 53 bits -> [0,1)
	const inv53 = 1.0 / 9007199254740992.0 // 2^53
	return float64(x>>11) * inv53
}

// ---------- SplitMix64 (lock-free) ----------

// splitmixNext advances s atomically and returns a mixed 64-bit value.
// This is the canonical SplitMix64 step: x += golden; mix(x).
func splitmixNext(s *uint64) uint64 {
	for {
		old := atomic.LoadUint64(s)
		x := old + 0x9e3779b97f4a7c15
		if atomic.CompareAndSwapUint64(s, old, x) {
			// mix x
			z := x
			z ^= z >> 30
			z *= 0xbf58476d1ce4e5b9
			z ^= z >> 27
			z *= 0x94d049bb133111eb
			z ^= z >> 31
			return z
		}
	}
}

// splitmixSeed turns a signed seed into a decent 64-bit starting state.
func splitmixSeed(seed int64) uint64 {
	z := uint64(seed) + 0x9e3779b97f4a7c15
	z ^= z >> 30
	z *= 0xbf58476d1ce4e5b9
	z ^= z >> 27
	z *= 0x94d049bb133111eb
	z ^= z >> 31
	if z == 0 {
		z = 0x9e3779b97f4a7c15
	}
	return z
}

func init() { Init(0) }
