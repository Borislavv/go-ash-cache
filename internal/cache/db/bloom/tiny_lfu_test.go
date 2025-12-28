package bloom

import (
	"github.com/Borislavv/go-ash-cache/internal/cache/db"
	"github.com/Borislavv/go-ash-cache/internal/config"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"
)

// Compact, high-signal config for unit tests (not for production).
// Goal: increase per-shard event density so frequency differences become visible fast.
var cfgTest = &config.AdmissionControlCfg{
	Capacity:            100_000,        // enough to "warm up" and get stable frequencies
	Shards:              512,            // fewer shards -> more traffic per shard
	MinTableLenPerShard: db.NumOfShards, // not tiny, not huge — good for unit testing
	DoorBitsPerCounter:  16,             // sufficient for short tests
	SampleMultiplier:    12,             // aging not too frequent
}

// key returns a deterministic key for index i (1-based to avoid zero).
func key(i int) uint64 { return uint64(i + 1) }

// recordTwice simulates two observations per key:
//  1. set doorkeeper bit
//  2. increment sketch (frequency++)
func recordTwice(tlfu *ShardedAdmitter, keys []uint64) {
	for _, k := range keys {
		tlfu.Record(k)
		tlfu.Record(k)
	}
}

// recordOnce simulates a single observation per key:
// sets the doorkeeper bit; may not reach the sketch yet.
func recordOnce(tlfu *ShardedAdmitter, keys []uint64) {
	for _, k := range keys {
		tlfu.Record(k)
	}
}

// admitStats tracks Allow() outcomes.
type admitStats struct {
	yes int
	no  int
}

func (s admitStats) rate() float64 {
	total := s.yes + s.no
	if total == 0 {
		return 0
	}
	return float64(s.yes) / float64(total)
}

// --- 1) Unique stream after warm-up ------------------------------------------
//
// Warm up with a set of keys that have freq >= 1 (victims are "warm").
// Then submit brand-new unique candidates against random warm victims.
// Expect a very low admit rate for uniques (reject-on-tie policy).
func TestTinyLFU_UniqueStreamRejectsAfterWarmup(t *testing.T) {
	tlfu := newShardedAdmitter(cfgTest)

	const warmN = 80_000
	const trials = 50_000

	// Warm up: each warm key observed twice -> sketch frequency >= 1.
	warm := make([]uint64, warmN)
	for i := 0; i < warmN; i++ {
		warm[i] = key(i)
	}
	recordTwice(tlfu, warm)

	// Unique candidates (not present in warm set).
	stats := admitStats{}
	r := rand.New(rand.NewSource(1))

	for i := 0; i < trials; i++ {
		candidate := key(warmN + 1 + i) // guaranteed new
		victim := warm[r.Intn(warmN)]
		if tlfu.Allow(candidate, victim) {
			stats.yes++
		} else {
			stats.no++
		}
	}

	// On a unique stream we expect a very low admit rate (< 10%).
	if rate := stats.rate(); rate >= 0.10 {
		t.Fatalf("unique-stream admit rate too high: got=%.2f%% want<10%% (yes=%d no=%d)",
			100*rate, stats.yes, stats.no)
	}
	t.Logf("unique-stream admit rate: %.2f%% (yes=%d no=%d)", 100*stats.rate(), stats.yes, stats.no)
}

// --- 2) Hot vs Cold preference ------------------------------------------------
//
// Make a small "hot" set truly hot (many observations) and a large "cold" set
// barely seen once. Then:
//
//	a) candidate=hot vs victim=cold  => expect high admit rate
//	b) candidate=cold vs victim=hot  => expect low admit rate
func TestTinyLFU_PrefersHotOverCold(t *testing.T) {
	tlfu := newShardedAdmitter(cfgTest)

	const hotN = 2_000
	const coldN = 60_000
	const trials = 50_000

	hot := make([]uint64, hotN)
	for i := 0; i < hotN; i++ {
		hot[i] = key(i + 1)
	}
	cold := make([]uint64, coldN)
	for i := 0; i < coldN; i++ {
		cold[i] = key(10_000 + i + 1)
	}

	// Make hot keys truly hot: multiple passes of recordTwice.
	for r := 0; r < 8; r++ {
		recordTwice(tlfu, hot)
	}
	// Cold keys: mark once (mostly doorkeeper), minimal frequency lift.
	recordOnce(tlfu, cold)

	rng := rand.New(rand.NewSource(2))

	// a) hot candidate vs cold victim
	hotWins := admitStats{}
	for i := 0; i < trials; i++ {
		candidate := hot[rng.Intn(hotN)]
		victim := cold[rng.Intn(coldN)]
		if tlfu.Allow(candidate, victim) {
			hotWins.yes++
		} else {
			hotWins.no++
		}
	}

	// b) cold candidate vs hot victim
	coldWins := admitStats{}
	for i := 0; i < trials; i++ {
		candidate := cold[rng.Intn(coldN)]
		victim := hot[rng.Intn(hotN)]
		if tlfu.Allow(candidate, victim) {
			coldWins.yes++
		} else {
			coldWins.no++
		}
	}

	hotRate := hotWins.rate()
	coldRate := coldWins.rate()

	// Expect a clear advantage for hot and clear disadvantage for cold.
	if hotRate < 0.85 {
		t.Fatalf("hot vs cold admit too low: got=%.2f%% want>=85%% (yes=%d no=%d)",
			100*hotRate, hotWins.yes, hotWins.no)
	}
	if coldRate > 0.15 {
		t.Fatalf("cold vs hot admit too high: got=%.2f%% want<=15%% (yes=%d no=%d)",
			100*coldRate, coldWins.yes, coldWins.no)
	}

	t.Logf("hot vs cold admit: %.2f%% (yes=%d no=%d)", 100*hotRate, hotWins.yes, hotWins.no)
	t.Logf("cold vs hot admit: %.2f%% (yes=%d no=%d)", 100*coldRate, coldWins.yes, coldWins.no)
}

// --- 3) Concurrent smoke test -------------------------------------------------
//
// Not a correctness proof for frequencies, but a fast concurrency check.
// It should finish within a short time without panics/data races.
func TestTinyLFU_ConcurrentSmoke(t *testing.T) {
	tlfu := newShardedAdmitter(cfgTest)

	var wg sync.WaitGroup
	workers := runtime.GOMAXPROCS(0)
	if workers < 2 {
		workers = 2
	}

	// Writers: Record()
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			r := rand.New(rand.NewSource(seed))
			for j := 0; j < 200_000; j++ {
				tlfu.Record(uint64(r.Int63()))
			}
		}(int64(i + 1))
	}

	// Arbiters: Allow()
	for i := 0; i < workers/2+1; i++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			r := rand.New(rand.NewSource(1<<32 + seed))
			for j := 0; j < 200_000; j++ {
				a := uint64(r.Int63())
				b := uint64(r.Int63())
				_ = tlfu.Allow(a, b)
			}
		}(int64(i + 1))
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(10 * time.Second):
		t.Fatalf("timeout: concurrent smoke took too long")
	}
}
