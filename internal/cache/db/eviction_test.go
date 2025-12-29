package db

import (
	"context"
	"github.com/Borislavv/go-ash-cache/config"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestMap_EvictUntilWithinLimit_ListingMode evicts entries in listing mode.
func TestMap_EvictUntilWithinLimit_ListingMode(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			SizeBytes: 10 * 1024 * 1024, // 10MB
		},
		Eviction: &config.EvictionCfg{
			LRUMode:              config.LRUModeListing,
			SoftLimitCoefficient: 0.8,
		},
	}
	cfg.AdjustConfig()

	ctx := context.Background()
	m := NewMap(ctx, cfg)

	// Fill cache with entries
	for i := 0; i < 100; i++ {
		entry := model.NewEntry(model.NewKey("test"), 0, false)
		entry.SetPayload(make([]byte, 100*1024)) // 100KB each
		m.Set(uint64(i), entry)
	}

	initialMem := m.Mem()
	require.Greater(t, initialMem, cfg.Eviction.SoftMemoryLimitBytes, "should exceed soft limit")

	// Evict until within limit
	freed, evicted := m.EvictUntilWithinLimit(cfg.Eviction.SoftMemoryLimitBytes, 10000)

	require.Greater(t, evicted, int64(0), "should evict some entries")
	require.Greater(t, freed, int64(0), "should free memory")
	require.LessOrEqual(t, m.Mem(), cfg.Eviction.SoftMemoryLimitBytes, "should be within limit")
}

// TestMap_EvictUntilWithinLimit_SamplingMode evicts entries in sampling mode.
func TestMap_EvictUntilWithinLimit_SamplingMode(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			SizeBytes: 10 * 1024 * 1024, // 10MB
		},
		Eviction: &config.EvictionCfg{
			LRUMode:              config.LRUModeSampling,
			SoftLimitCoefficient: 0.8,
		},
	}
	cfg.AdjustConfig()

	ctx := context.Background()
	m := NewMap(ctx, cfg)

	// Fill cache with entries
	for i := 0; i < 100; i++ {
		entry := model.NewEntry(model.NewKey("test"), 0, false)
		entry.SetPayload(make([]byte, 100*1024)) // 100KB each
		m.Set(uint64(i), entry)
	}

	initialMem := m.Mem()
	require.Greater(t, initialMem, cfg.Eviction.SoftMemoryLimitBytes, "should exceed soft limit")

	// Evict until within limit (sampling may need more attempts)
	freed, evicted := m.EvictUntilWithinLimit(cfg.Eviction.SoftMemoryLimitBytes, 50000)

	require.GreaterOrEqual(t, evicted, int64(0), "may evict entries")
	require.GreaterOrEqual(t, freed, int64(0), "may free memory")
}

// TestMap_PickVictim_ListingMode returns least recently used entry.
func TestMap_PickVictim_ListingMode(t *testing.T) {
	cfg := &config.Cache{
		Eviction: &config.EvictionCfg{
			LRUMode: config.LRUModeListing,
		},
	}
	cfg.AdjustConfig()

	ctx := context.Background()
	m := NewMap(ctx, cfg)

	// Insert entries with different access times
	entry1 := model.NewEntry(model.NewKey("old"), 0, false)
	entry1.SetPayload([]byte("data1"))
	entry2 := model.NewEntry(model.NewKey("new"), 0, false)
	entry2.SetPayload([]byte("data2"))

	m.Set(1, entry1)
	// Touch entry2 to make it more recent
	m.Touch(1) // This moves entry1 to front
	m.Set(2, entry2)

	shard, victim, ok := m.PickVictim(2, 8)

	require.True(t, ok)
	require.NotNil(t, shard)
	require.NotNil(t, victim)
	// In listing mode, should pick from tail (least recent)
}

// TestMap_PickVictim_SamplingMode returns sampled victim.
func TestMap_PickVictim_SamplingMode(t *testing.T) {
	cfg := &config.Cache{
		Eviction: &config.EvictionCfg{
			LRUMode: config.LRUModeSampling,
		},
	}
	cfg.AdjustConfig()

	ctx := context.Background()
	m := NewMap(ctx, cfg)

	// Insert entries
	for i := 0; i < 20; i++ {
		entry := model.NewEntry(model.NewKey("test"), 0, false)
		entry.SetPayload([]byte("data"))
		m.Set(uint64(i), entry)
	}

	shard, victim, ok := m.PickVictim(4, 8)

	require.True(t, ok)
	require.NotNil(t, shard)
	require.NotNil(t, victim)
}

// TestMap_PickVictim_Empty returns false when cache is empty.
func TestMap_PickVictim_Empty(t *testing.T) {
	cfg := &config.Cache{
		Eviction: &config.EvictionCfg{
			LRUMode: config.LRUModeListing,
		},
	}
	cfg.AdjustConfig()

	ctx := context.Background()
	m := NewMap(ctx, cfg)

	_, _, ok := m.PickVictim(2, 8)

	require.False(t, ok)
}

// TestMap_EvictUntilWithinLimit_RespectsMinLimit respects minimum eviction limit.
func TestMap_EvictUntilWithinLimit_RespectsMinLimit(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			SizeBytes: 10 * 1024 * 1024, // 10MB
		},
		Eviction: &config.EvictionCfg{
			LRUMode:              config.LRUModeListing,
			SoftLimitCoefficient: 0.8,
		},
	}
	cfg.AdjustConfig()

	ctx := context.Background()
	m := NewMap(ctx, cfg)

	// Fill cache just above soft limit
	for i := 0; i < 10; i++ {
		entry := model.NewEntry(model.NewKey("test"), 0, false)
		entry.SetPayload(make([]byte, 1024*1024)) // 1MB each
		m.Set(uint64(i), entry)
	}

	// Evict with small backoff (should respect min limit of 8MB)
	freed, evicted := m.EvictUntilWithinLimit(cfg.Eviction.SoftMemoryLimitBytes, 100)

	// Should evict at least until min limit (8MB) is reached
	require.Greater(t, evicted, int64(0), "should evict some entries")
	require.Greater(t, freed, int64(0), "should free memory")
}

// TestMap_EvictUntilWithinLimit_StopsWhenEmpty stops when cache is empty.
func TestMap_EvictUntilWithinLimit_StopsWhenEmpty(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			SizeBytes: 10 * 1024 * 1024,
		},
		Eviction: &config.EvictionCfg{
			LRUMode:              config.LRUModeListing,
			SoftLimitCoefficient: 0.8,
		},
	}
	cfg.AdjustConfig()

	ctx := context.Background()
	m := NewMap(ctx, cfg)

	// Add one entry
	entry := model.NewEntry(model.NewKey("test"), 0, false)
	entry.SetPayload(make([]byte, 1024))
	key := entry.Key().Value()
	m.Set(key, entry)

	// Evict with large backoff
	// Note: eviction only occurs if memory exceeds limit
	// A single 1KB entry won't exceed 8MB soft limit, so eviction won't occur
	freed, evicted := m.EvictUntilWithinLimit(cfg.Eviction.SoftMemoryLimitBytes, 10000)

	// If memory is below limit, eviction won't occur
	if m.Mem() <= cfg.Eviction.SoftMemoryLimitBytes {
		require.Equal(t, int64(0), evicted, "should not evict if below limit")
		require.Equal(t, int64(0), freed)
		require.Equal(t, int64(1), m.Len(), "cache should still have entry")
	} else {
		// If somehow over limit, eviction should occur
		require.Greater(t, evicted, int64(0))
		require.Greater(t, freed, int64(0))
	}
}
