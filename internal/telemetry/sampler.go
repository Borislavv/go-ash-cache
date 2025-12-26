package telemetry

import (
	"github.com/Borislavv/go-ash-cache/internal/cache"
	"github.com/Borislavv/go-ash-cache/internal/evictor"
	"github.com/Borislavv/go-ash-cache/internal/lifetimer"
)

type sampler struct {
	cache     cache.Cacher
	evictor   evictor.Evictor
	lifetimer lifetimer.Lifetimer
}

func newSampler(c cache.Cacher, e evictor.Evictor, lt lifetimer.Lifetimer) sampler {
	return sampler{cache: c, evictor: e, lifetimer: lt}
}

// snapshot holds cumulative counters (monotonic).
type snapshot struct {
	admissionAllowed    uint64
	admissionNotAllowed uint64

	softScans        uint64
	softHits         uint64
	softEvictedItems uint64
	softEvictedBytes uint64
	hardEvictedItems uint64
	hardEvictedBytes uint64

	lifetimeAffected uint64
	lifetimeErrors   uint64
	lifetimeScans    uint64
	lifetimeHits     uint64
	lifetimeMisses   uint64
}

func (s sampler) snapshot() snapshot {
	aAllowed, aNotAllowed, hardItems, hardBytes := s.cache.Metrics()
	softScans, softHits, softItems, softBytes := s.evictor.Metrics()
	affected, errs, scans, hits, misses := s.lifetimer.Metrics()

	return snapshot{
		admissionAllowed:    uint64(max(aAllowed, 0)),
		admissionNotAllowed: uint64(max(aNotAllowed, 0)),

		softScans:        uint64(max(softScans, 0)),
		softHits:         uint64(max(softHits, 0)),
		softEvictedItems: uint64(max(softItems, 0)),
		softEvictedBytes: uint64(max(softBytes, 0)),
		hardEvictedItems: uint64(max(hardItems, 0)),
		hardEvictedBytes: uint64(max(hardBytes, 0)),

		lifetimeAffected: uint64(max(affected, 0)),
		lifetimeErrors:   uint64(max(errs, 0)),
		lifetimeScans:    uint64(max(scans, 0)),
		lifetimeHits:     uint64(max(hits, 0)),
		lifetimeMisses:   uint64(max(misses, 0)),
	}
}

// deltaSnapshot converts cumulative snapshots to per-interval deltas.
// If counters reset (cur < prev), it treats cur as the delta.
func deltaSnapshot(prev, cur snapshot) snapshot {
	return snapshot{
		admissionAllowed:    delta(prev.admissionAllowed, cur.admissionAllowed),
		admissionNotAllowed: delta(prev.admissionNotAllowed, cur.admissionNotAllowed),
		hardEvictedItems:    delta(prev.hardEvictedItems, cur.hardEvictedItems),
		hardEvictedBytes:    delta(prev.hardEvictedBytes, cur.hardEvictedBytes),

		softScans:        delta(prev.softScans, cur.softScans),
		softHits:         delta(prev.softHits, cur.softHits),
		softEvictedItems: delta(prev.softEvictedItems, cur.softEvictedItems),
		softEvictedBytes: delta(prev.softEvictedBytes, cur.softEvictedBytes),

		lifetimeAffected: delta(prev.lifetimeAffected, cur.lifetimeAffected),
		lifetimeErrors:   delta(prev.lifetimeErrors, cur.lifetimeErrors),
		lifetimeScans:    delta(prev.lifetimeScans, cur.lifetimeScans),
		lifetimeHits:     delta(prev.lifetimeHits, cur.lifetimeHits),
		lifetimeMisses:   delta(prev.lifetimeMisses, cur.lifetimeMisses),
	}
}

func delta(prev, cur uint64) uint64 {
	if cur >= prev {
		return cur - prev
	}
	return cur
}
