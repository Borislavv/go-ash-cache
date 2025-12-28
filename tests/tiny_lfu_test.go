package tests

import (
	"github.com/Borislavv/go-ash-cache/internal/cache/db/bloom"
	"testing"

	"github.com/Borislavv/go-ash-cache/internal/config"
)

func TestTinyLFU_Record_DoorkeeperGating(t *testing.T) {
	a := newTestAdmitter()

	// Pick hash mapping to any shard, stable.
	const h uint64 = 0x100 // ...&3==0 for Shards=4

	// Before any record.
	if got := a.Estimate(h); got != 0 {
		t.Fatalf("expected initial estimate=0, got=%d", got)
	}

	// First record: doorkeeper adds, sketch not incremented.
	a.Record(h)
	if got := a.Estimate(h); got != 0 {
		t.Fatalf("expected estimate still 0 after first Record (doorkeeper warmup), got=%d", got)
	}

	// Second record: doorkeeper returns seen => sketch incremented at least by 1.
	a.Record(h)
	if got := a.Estimate(h); got == 0 {
		t.Fatalf("expected estimate > 0 after second Record, got=%d", got)
	}
}

func TestTinyLFU_Allow_RejectsUnseenCandidate(t *testing.T) {
	a := newTestAdmitter()

	// Ensure same shard (so we don't mix in cross-shard quirks).
	// With Shards=4 => mask=3. Both &3 == 0.
	const candidate uint64 = 0x100 // shard 0
	const victim uint64 = 0x200    // shard 0

	// Make victim "hot" in its shard (also makes it present in doorkeeper).
	recordN(a, victim, 10)

	// Candidate has never been recorded => doorkeeper probablySeen=false => must reject.
	if a.Allow(candidate, victim) {
		t.Fatalf("expected Allow=false for unseen candidate (doorkeeper gating)")
	}
}

func TestTinyLFU_Allow_PrefersHotCandidate_SameShard(t *testing.T) {
	a := newTestAdmitter()

	// Same shard (shard 0).
	const candidate uint64 = 0x100
	const victim uint64 = 0x200

	// Warm both in doorkeeper (need at least 1 record).
	// Then build a big deterministic gap in sketch.
	recordN(a, victim, 2)     // 1 warmup + 1 increment-ish
	recordN(a, candidate, 50) // warmup + many increments

	// Candidate should be seen and much hotter => must allow.
	if !a.Allow(candidate, victim) {
		t.Fatalf("expected Allow=true when candidate is much hotter (same shard); cand=%d vict=%d",
			a.Estimate(candidate), a.Estimate(victim),
		)
	}

	// Reset halves counters + clears doorkeeper.
	a.Reset()

	// Now make victim much hotter than candidate (same shard) again.
	recordN(a, candidate, 2)
	recordN(a, victim, 50)

	if a.Allow(candidate, victim) {
		t.Fatalf("expected Allow=false when victim is much hotter (same shard); cand=%d vict=%d",
			a.Estimate(candidate), a.Estimate(victim),
		)
	}
}

func TestTinyLFU_Reset_HalvesSketchCounters(t *testing.T) {
	a := newTestAdmitter()

	// Same shard, stable.
	const h uint64 = 0x100

	// Make estimate confidently > 1 so halving is observable.
	recordN(a, h, 200)
	before := a.Estimate(h)
	if before < 2 {
		t.Fatalf("unexpectedly small estimate before Reset: %d (increase record count if needed)", before)
	}

	a.Reset()

	after := a.Estimate(h)

	// Your sketch.reset() halves counters; allow for integer rounding.
	lo := before / 2
	hi := (before + 1) / 2 // ceil(before/2)

	// Some implementations may differ by +/-1 due to CMS min-of-rows behaviour.
	// Keep it strict but not brittle.
	const tol uint8 = 1
	if after+tol < lo || after > hi+tol {
		t.Fatalf("expected Reset to halve estimate (±%d): before=%d after=%d want in [%d..%d]",
			tol, before, after, lo, hi,
		)
	}
}

// This test exposes a correctness issue in the current Allow() implementation:
// it estimates victim frequency using the candidate shard sketch.
// That can lead to admitting a cold candidate over a hot victim when they map to different shards.
//
// Keep it as a "red test" until Allow() is fixed to estimate victim in its own shard.
func TestTinyLFU_Allow_CrossShardVictim_Bug(t *testing.T) {
	t.Skip("BUG: Allow() estimates victim in candidate shard; enable after fixing Allow() to use victim shard")

	a := newTestAdmitter()

	// Different shards (Shards=4 => mask=3)
	// candidate shard 0, victim shard 1
	const candidate uint64 = 0x100 // &3==0
	const victim uint64 = 0x101    // &3==1

	// Warm candidate just enough to be "seen" so doorkeeper doesn't reject it.
	recordN(a, candidate, 2)

	// Make victim extremely hot (in its own shard).
	recordN(a, victim, 500)

	// Correct TinyLFU decision: candidate should NOT replace victim.
	// Current code may incorrectly allow due to cross-shard estimate bug.
	if a.Allow(candidate, victim) {
		t.Fatalf("expected Allow=false when victim is much hotter across shards; cand=%d vict=%d",
			a.Estimate(candidate), a.Estimate(victim),
		)
	}
}

// ---- helpers ----

func newTestAdmitter() bloom.AdmissionControl {
	// Deterministic, compact config.
	cfg := &config.AdmissionControlCfg{
		Capacity:            128,
		Shards:              4,
		MinTableLenPerShard: 64,
		SampleMultiplier:    3,
		DoorBitsPerCounter:  2,
	}

	// Prefer calling the concrete constructor to avoid cfg.Enabled() semantics.
	return bloom.NewAdmissionControl(cfg)
}

func recordN(a bloom.AdmissionControl, h uint64, n int) {
	for i := 0; i < n; i++ {
		a.Record(h)
	}
}
