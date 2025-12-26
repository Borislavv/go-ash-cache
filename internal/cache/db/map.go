//go:build go1.20

// Package db Package sharded implements a high‑throughput, zero‑allocation sharded map
// intended for in‑memory cache workloads. Hot paths (Get/Set/TTLModeRemove) avoid
// allocations and keep critical sections short. Global counters are atomics so
// they can be read without locks.
package db

import (
	"context"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"github.com/Borislavv/go-ash-cache/internal/config"
	"runtime"
	"sync"
	"sync/atomic"
)

// Tunables.
const (
	NumOfShards = 1024
	shardMask   = NumOfShards - 1 // faster than division
)

// Map is a sharded concurrent map with precise global counters.
type Map struct {
	mode LRUMode // eviction strategy
	ctx  context.Context
	cfg  *config.Cache

	len  int64  // aggregated number of items (atomic)
	mem  int64  // aggregated payload size in bytes (atomic)
	iter uint64 // round‑robin cursor for NextShard()

	shards [NumOfShards]*Shard
}

// NewMap creates the map and initializes shards. A lightweight gauge updater runs once per second and exits with ctx.
func NewMap(ctx context.Context, cfg *config.Cache) *Map {
	m := &Map{ctx: ctx, cfg: cfg}
	for id := uint64(0); id < NumOfShards; id++ {
		m.shards[id] = NewShard(id)
	}

	if cfg.Eviction.Enabled() && cfg.Eviction.IsListing {
		m.useListingMode()
	} else {
		m.useSamplingMode()
	}
	return m
}

// Set inserts/updates a value and adjusts global counters via per‑shard deltas.
func (m *Map) Set(key uint64, value *model.Entry) {
	bytesDelta, lenDelta := m.Shard(key).Set(key, value)
	if bytesDelta != 0 {
		atomic.AddInt64(&m.mem, bytesDelta)
	}
	if lenDelta != 0 {
		atomic.AddInt64(&m.len, lenDelta)
	}
}

// Get reads a value.
func (m *Map) Get(key uint64) (value *model.Entry, ok bool) {
	return m.Shard(key).Get(key)
}

// Remove deletes a key and adjusts global counters.
func (m *Map) Remove(key uint64) (freedBytes int64, hit bool) {
	freedBytes, hit = m.Shard(key).Remove(key)
	if hit {
		atomic.AddInt64(&m.len, -1)
		atomic.AddInt64(&m.mem, -freedBytes)
	}
	return
}

// WalkShards applies fn to all shards synchronously (zero-alloc).
func (m *Map) WalkShards(ctx context.Context, fn func(key uint64, shard *Shard)) {
	for k, s := range m.shards {
		if ctx.Err() != nil {
			return
		}
		fn(uint64(k), s)
	}
}

// WalkShardsConcurrent executes fn over shards with bounded concurrency.
// Use in maintenance/background tasks; avoid on hot paths.
func (m *Map) WalkShardsConcurrent(ctx context.Context, concurrency int, fn func(key uint64, shard *Shard)) {
	if concurrency <= 0 {
		concurrency = runtime.GOMAXPROCS(0)
	}
	var (
		wg sync.WaitGroup
		ch = make(chan int, NumOfShards)
	)
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for idx := range ch {
				if ctx.Err() != nil {
					return
				}
				fn(uint64(idx), m.shards[idx])
			}
		}()
	}
	for idx := range m.shards {
		select {
		case <-ctx.Done():
			close(ch)
			wg.Wait()
			return
		case ch <- idx:
		}
	}
	close(ch)
	wg.Wait()
}

// Clear wipes all shards and fixes global counters atomically.
func (m *Map) Clear() {
	m.WalkShards(m.ctx, func(_ uint64, shard *Shard) {
		freedBytes, items := shard.Clear()
		if freedBytes != 0 {
			atomic.AddInt64(&m.mem, -freedBytes)
		}
		if items != 0 {
			atomic.AddInt64(&m.len, -items)
		}
	})
}

func (m *Map) Shard(key uint64) *Shard { return m.shards[key&shardMask] }
func (m *Map) NextShard() *Shard       { return m.shards[atomic.AddUint64(&m.iter, 1)&shardMask] }
func (m *Map) Len() int64              { return atomic.LoadInt64(&m.len) }
func (m *Map) Mem() int64              { return atomic.LoadInt64(&m.mem) }
func (m *Map) AddMem(key uint64, delta int64) {
	atomic.AddInt64(&m.mem, delta)
	m.Shard(key).AddMem(delta)
}

func (m *Map) useListingMode() {
	m.mode = Listing
	for _, s := range m.shards {
		s.enableLRU()
	}
}

func (m *Map) useSamplingMode() {
	m.mode = Sampling
	for _, s := range m.shards {
		s.disableLRU()
	}
}

func (m *Map) Touch(key uint64) {
	if m.mode != Listing {
		return
	}
	m.Shard(key).touchLRU(key)
}
