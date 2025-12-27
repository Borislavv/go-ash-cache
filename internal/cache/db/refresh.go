package db

import (
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"sync/atomic"
)

const rLockSpins, rwLockSpins = 8, 16

func (m *Map) PeekExpiredTTL() (*model.Entry, bool) {
	if v, ok := m.NextQueuedWithExpiredTTL(); ok {
		return v, true
	} else {
		const defaultSample = 32
		return m.peekExpired(defaultSample)
	}
}

// EnqueueExpired tries to put key to its shard refresh queue.
func (m *Map) EnqueueExpired(key uint64) bool { return m.Shard(key).EnqueueRefresh(key) }

// NextQueuedWithExpiredTTL tries to pop one queued key from up to 'probes' shards.
func (m *Map) NextQueuedWithExpiredTTL() (*model.Entry, bool) {
	start := int((atomic.AddUint64(&m.iter, 1) - 1) & shardMask)
	for i := 0; i < NumOfShards; i++ {
		sh := m.shards[(start+i)&shardMask]
		if k, ok := sh.DequeueExpired(); ok {
			if v, ok2 := sh.Get(k); ok2 { // under RLock
				// double-check freshness
				if v.IsExpired(m.cfg) {
					// caller refreshes; the flag will be cleared after success
					return v, true
				} else {
					// not ready; reset flag
					v.DequeueExpired()
				}
			}
		}
	}
	return nil, false
}

func (m *Map) peekExpired(sample int) (*model.Entry, bool) {
	var (
		best    *model.Entry
		seen    int
		hitSeen int
		set     bool
		maxSeen = sample * rwLockSpins
		shards  = maxSeen
	)

loop:
	for shard := 0; shard < shards; shard++ {
		sh := m.NextShard()
		if sh.Len() == 0 || !sh.tryRLock() {
			continue
		}

		for _, entry := range sh.items {
			if seen >= maxSeen || hitSeen >= sample {
				sh.RUnlock()
				break loop
			}
			if entry.IsExpired(m.cfg) {
				hitSeen++
				if !set {
					best = entry
					set = true
				} else if best != nil && best.UpdatedAt() > entry.UpdatedAt() {
					best = entry
				}
			}
			seen++
		}
		sh.RUnlock()
	}

	return best, set
}
