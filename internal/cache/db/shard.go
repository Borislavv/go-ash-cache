//go:build go1.20

package db

import (
	"container/list"
	"context"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"github.com/Borislavv/go-ash-cache/internal/shared/queue"
	model2 "github.com/Borislavv/go-ash-cache/model"
	"runtime"
	"sync"
	"sync/atomic"
)

const queueCap = 4096

// Shard is an independent segment of the sharded map.
// It keeps per-shard counters read with atomics so global readers can avoid locks.
type Shard struct {
	sync.RWMutex
	items map[uint64]*model.Entry

	id       uint64
	mem      int64  // total payload weight in bytes (atomic)
	len      int64  // number of items (atomic)
	randIter uint64 // cheap pseudo-random offset & probes

	// LRU (enabled in Listing mode)
	lruOn bool
	lru   *list.List
	lidx  map[uint64]*list.Element

	rq queue.Queue
}

// NewShard creates a shard with small map capacity and fixed-size reservoirs.
func NewShard(id uint64) *Shard {
	sh := &Shard{id: id, items: make(map[uint64]*model.Entry)}
	sh.rq.Init(queueCap)
	return sh
}

func (sh *Shard) ID() uint64         { return sh.id }
func (sh *Shard) Weight() int64      { return atomic.LoadInt64(&sh.mem) }
func (sh *Shard) Len() int64         { return atomic.LoadInt64(&sh.len) }
func (sh *Shard) AddMem(delta int64) { atomic.AddInt64(&sh.mem, delta) }

// Set inserts or updates a key. Returns deltas for global aggregations.
func (sh *Shard) Set(key uint64, new *model.Entry) (bytesDelta int64, lenDelta int64) {
	sh.Lock()
	if old, hit := sh.items[key]; hit {
		sh.items[key] = new
		sh.lruOnAccessUnlocked(key)

		lenDelta = 0
		bytesDelta = new.Weight() - old.Weight()
		atomic.AddInt64(&sh.mem, bytesDelta)
	} else {
		sh.items[key] = new
		sh.lruOnInsertUnlocked(key)

		lenDelta = 1
		bytesDelta = new.Weight()
		atomic.AddInt64(&sh.len, lenDelta)
		atomic.AddInt64(&sh.mem, bytesDelta)
	}
	sh.Unlock()
	return
}

// Get reads a value under a shared lock.
func (sh *Shard) Get(key uint64) (value *model.Entry, hit bool) {
	sh.RLock()
	value, hit = sh.items[key]
	sh.RUnlock()
	return
}

// Remove deletes a key under the write lock.
func (sh *Shard) Remove(key uint64) (freedBytes int64, hit bool) {
	sh.Lock()
	freedBytes, hit = sh.RemoveUnlocked(key)
	sh.Unlock()
	return
}

// RemoveUnlocked deletes a key when the shard is already exclusively locked.
func (sh *Shard) RemoveUnlocked(key uint64) (freedBytes int64, hit bool) {
	var old *model.Entry
	if old, hit = sh.items[key]; hit {
		delete(sh.items, key)
		sh.lruOnDeleteUnlocked(key)

		freedBytes = old.Weight()
		atomic.AddInt64(&sh.mem, -freedBytes)
		atomic.AddInt64(&sh.len, -1)
	}
	return
}

// Clear removes all entries and returns (freedBytes, itemsRemoved).
// Reservoirs are kept intact; stale keys are naturally validated&skipped.
func (sh *Shard) Clear() (freedBytes int64, items int64) {
	sh.Lock()
	items = atomic.LoadInt64(&sh.len)
	freedBytes = atomic.LoadInt64(&sh.mem)

	sh.items = make(map[uint64]*model.Entry, items)

	atomic.StoreInt64(&sh.len, 0)
	atomic.StoreInt64(&sh.mem, 0)
	if sh.lru != nil {
		sh.lru.Init()
	}
	if sh.lidx != nil {
		clear(sh.lidx)
	}
	sh.Unlock()
	return
}

// WalkR iterates (k,v) under a shared lock. The callback must be lightweight.
func (sh *Shard) WalkR(ctx context.Context, fn func(uint64, *model.Entry) bool) {
	sh.RLock()
	defer sh.RUnlock()
	for k, v := range sh.items {
		select {
		case <-ctx.Done():
			return
		default:
			if !fn(k, v) {
				return
			}
		}
	}
}

// Walk iterates (k,v) under a shared lock. The callback must be lightweight.
func (sh *Shard) Walk(ctx context.Context, fn func(cache model2.CacheItem) bool, write bool) {
	if write {
		sh.Lock()
		defer sh.Unlock()
	} else {
		sh.RLock()
		defer sh.RUnlock()
	}
	for _, v := range sh.items {
		select {
		case <-ctx.Done():
			return
		default:
			if !fn(v) {
				return
			}
		}
	}
}

// WalkW iterates under the write lock. Use with care.
func (sh *Shard) WalkW(ctx context.Context, fn func(uint64, *model.Entry) bool) {
	sh.Lock()
	defer sh.Unlock()
	for k, v := range sh.items {
		select {
		case <-ctx.Done():
			return
		default:
			if !fn(k, v) {
				return
			}
		}
	}
}

// EnqueueRefresh insert key into refresh bucket
func (sh *Shard) EnqueueRefresh(key uint64) bool { return sh.rq.TryPush(key) }

// DequeueExpired remove key from refresh bucket
func (sh *Shard) DequeueExpired() (uint64, bool) { return sh.rq.TryPop() }

func (sh *Shard) tryRLock() bool {
	for i := 0; i < rLockSpins; i++ {
		if sh.TryRLock() {
			return true
		}
		runtime.Gosched()
	}
	return false
}

func (sh *Shard) tryLock() bool {
	for i := 0; i < rwLockSpins; i++ {
		if sh.TryLock() {
			return true
		}
		runtime.Gosched()
	}
	return false
}
