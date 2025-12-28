package bloom

import (
	"github.com/Borislavv/go-ash-cache/config"
)

type ShardedAdmitter struct {
	mask   uint32
	shards []shard
}

type shard struct {
	// 4-bit counters packed in 64-bit words (16 counters per word).
	sketch sketch
	// simple Bloom-like bitset; reset with sketch aging.
	door doorkeeper
	_    [64]byte // cacheline padding (isolation between shards)
}

// newShardedAdmitter (legacy) builds an admitter from an explicit Config.
// Kept for backward compatibility with existing code/tests.
func newShardedAdmitter(cfg *config.AdmissionControlCfg) *ShardedAdmitter {
	perShardCap := cfg.Capacity / cfg.Shards
	if perShardCap < 1 {
		perShardCap = 1
	}

	// Table length is a power-of-two >= perShardCap, clamped by MinTableLenPerShard.
	tblLen := nextPow2(perShardCap)
	if tblLen < cfg.MinTableLenPerShard {
		tblLen = cfg.MinTableLenPerShard
	}

	// Doorkeeper size is proportional to the counter space.
	doorBits := tblLen * cfg.DoorBitsPerCounter

	out := &ShardedAdmitter{
		mask:   uint32(cfg.Shards - 1),
		shards: make([]shard, cfg.Shards),
	}
	for i := range out.shards {
		out.shards[i].sketch.init(uint32(tblLen), uint32(cfg.SampleMultiplier))
		out.shards[i].door.init(uint32(doorBits))
	}
	return out
}

// Record observes a key access. We use the doorkeeper to gate noise: first sight -> set bit only.
// Second (or FP) sight -> increment TinyLFU sketch (approximate frequency).
func (a *ShardedAdmitter) Record(h uint64) {
	sh := &a.shards[h&uint64(a.mask)]
	if sh.door.seenOrAdd(h) {
		sh.sketch.increment(h)
	}
}

// Allow returns true if the candidate should replace a victim according to TinyLFU.
// If the candidate is unseen by the doorkeeper we conservatively reject (unless caller
// uses a small “window” segment to bypass admission).
func (a *ShardedAdmitter) Allow(candidate, victim uint64) bool {
	if candidate == victim {
		// Same entry: no replacement needed, but "allow" is safe.
		return true
	}

	mask := uint64(a.mask)

	// Gate on candidate doorkeeper: unseen candidates are rejected early.
	candSh := &a.shards[candidate&mask]
	if !candSh.door.probablySeen(candidate) {
		return false
	}

	// Estimate candidate frequency in candidate shard.
	cf := candSh.sketch.estimate(candidate)

	// Estimate victim frequency in victim shard (IMPORTANT for correctness).
	victSh := &a.shards[victim&mask]
	vf := victSh.sketch.estimate(victim)

	// Strict preference: admit only if candidate is strictly more frequent.
	// (Keeps eviction stable and avoids churn on ties.)
	return cf > vf
}

// Estimate exposes freq estimate (for metrics/diagnostics).
func (a *ShardedAdmitter) Estimate(h uint64) uint8 {
	sh := &a.shards[h&uint64(a.mask)]
	return sh.sketch.estimate(h)
}

// Reset forces aging now (useful for tests or ops hooks). Also resets the doorkeeper.
func (a *ShardedAdmitter) Reset() {
	for i := range a.shards {
		a.shards[i].sketch.reset()
		a.shards[i].door.reset()
	}
}
