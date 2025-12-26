package db

import (
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"runtime"
	"sync/atomic"
)

const shardsSample, keysSample = 4, 8

func (m *Map) EvictUntilWithinLimit(limit, backoff int64) (freed, evicted int64) {
	if m.mode == Listing {
		return m.evictUntilWithinLimitByList(limit, backoff)
	} else {
		return m.evictUntilWithinLimitBySample(limit, backoff)
	}
}

func (m *Map) evictUntilWithinLimitByList(limit, backoff int64) (freed, evicted int64) {
	if m.mode != Listing {
		return 0, 0
	}

	// min over eviction (8MiB)
	var minLimit int64 = 8 << 20

	// eviction loop
	for backoff > 0 {
		curUsage := atomic.LoadInt64(&m.mem)
		if (curUsage <= limit && freed <= minLimit) || m.Len() == 0 {
			return freed, evicted
		}
		sh := m.NextShard()
		if sh.Len() == 0 {
			backoff--
			runtime.Gosched()
			continue
		}
		if _, v, ok := sh.lruPopTail(); ok {
			w := v.Weight()
			atomic.AddInt64(&m.mem, -w)
			atomic.AddInt64(&m.len, -1)
			freed += w
			evicted++
		}
		backoff--
	}
	return
}

func (m *Map) evictUntilWithinLimitBySample(limit, backoff int64) (freed, evicted int64) {
	if m.mode != Sampling || m.Mem() <= limit || m.Len() <= 0 {
		return 0, 0
	}

	for atomic.LoadInt64(&m.mem) > limit && backoff > 0 {
		sh, victim, found := m.pickVictimBySample(shardsSample, keysSample)
		if !found || !sh.tryLock() {
			backoff--
			continue
		}
		bytesFreed, hit := sh.RemoveUnlocked(victim.Key().Value())
		sh.Unlock()
		if bytesFreed > 0 || hit {
			atomic.AddInt64(&m.mem, -bytesFreed)
			atomic.AddInt64(&m.len, -1)
			freed += bytesFreed
			evicted++
		}
		backoff--
	}
	return freed, evicted
}

func (m *Map) PickVictim(shardsSample, keysSample int64) (bestShard *Shard, victim *model.Entry, ok bool) {
	if m.mode == Listing {
		return m.pickVictimByList()
	} else {
		return m.pickVictimBySample(shardsSample, keysSample)
	}
}

func (m *Map) pickVictimByList() (bestShard *Shard, victim *model.Entry, ok bool) {
	if m.mode != Listing {
		return nil, victim, false
	}

	const probes = 8
	start := int((atomic.AddUint64(&m.iter, 1) - 1) & shardMask)

	var (
		haveBest bool
		bestAt   int64
		bestV    *model.Entry
		bestSh   *Shard
	)

	for i := 0; i < probes; i++ {
		sh := m.shards[(start+i)&shardMask]
		if sh.Len() == 0 {
			continue
		}
		if _, v, ok2 := sh.lruPeekTail(); ok2 {
			at := v.TouchedAt()
			if !haveBest || at < bestAt {
				haveBest, bestAt, bestV, bestSh = true, at, v, sh
			}
		}
	}

	if !haveBest {
		return nil, victim, false
	}
	return bestSh, bestV, true
}

func (m *Map) pickVictimBySample(shardsSample, keysSample int64) (bestShard *Shard, victim *model.Entry, ok bool) {
	if m.mode != Sampling {
		return
	}

	var (
		bestV    *model.Entry
		bestAt   int64
		bestSh   *Shard
		haveBest bool
	)

	for i := int64(0); i < shardsSample; i++ {
		sh := m.NextShard()
		if sh.Len() == 0 {
			continue
		} else if !sh.tryRLock() {
			runtime.Gosched()
			continue
		}

		shardLen := sh.Len()
		if shardLen == 0 {
			sh.RUnlock()
			continue
		}

		toScanPerShard := keysSample
		if toScanPerShard > shardLen {
			toScanPerShard = shardLen
		}

		for _, reviewEntry := range sh.items {
			at := reviewEntry.TouchedAt()
			if !haveBest || at < bestAt {
				bestV, bestAt, bestSh, haveBest = reviewEntry, at, sh, true
			}

			if toScanPerShard--; toScanPerShard <= 0 {
				break
			}
		}
		sh.RUnlock()
	}

	if !haveBest {
		return nil, victim, false
	}
	return bestSh, bestV, true
}
