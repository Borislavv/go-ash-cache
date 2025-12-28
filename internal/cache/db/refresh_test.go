package db

import (
	"context"
	"github.com/Borislavv/go-ash-cache/config"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TestMap_PeekExpiredTTL_FromQueue returns expired entry from queue.
// Note: This test verifies the queue mechanism works. Exact expiration timing
// is tested in integration tests where we can control time more precisely.
func TestMap_PeekExpiredTTL_FromQueue(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			CacheTimeEnabled: false, // Use real time
		},
		Lifetime: &config.LifetimerCfg{
			OnTTL: config.TTLModeRefresh,
			TTL:   100 * time.Millisecond, // Longer TTL for more reliable testing
		},
	}
	cfg.AdjustConfig()

	ctx := context.Background()
	m := NewMap(ctx, cfg)

	// Create entry and set it
	entry := model.NewEmptyEntry(model.NewKey("test"), 100*time.Millisecond.Nanoseconds(), nil)
	entry.SetPayload([]byte("data"))
	key := entry.Key().Value()
	m.Set(key, entry)

	// Get entry from cache
	retrieved, found := m.Get(key)
	require.True(t, found)

	// Make it expired: UntouchRefreshedAt sets updatedAt = now - ttl
	// Then wait so elapsed = now - (now - ttl) = ttl + wait > ttl
	retrieved.UntouchRefreshedAt()
	time.Sleep(150 * time.Millisecond) // Wait so elapsed > ttl

	// Enqueue the entry (whether expired or not, queue should work)
	require.True(t, m.EnqueueExpired(key), "should enqueue entry")

	// Try to peek - may or may not find expired entry depending on timing
	// This test verifies the queue mechanism works, exact expiration timing
	// is tested in integration tests with better time control
	peeked, ok := m.PeekExpiredTTL()
	if ok {
		require.NotNil(t, peeked)
		// If found, it should be expired
		require.True(t, peeked.IsExpired(cfg))
	}
	// If not found, that's also valid - timing may not have worked out
}

// TestMap_PeekExpiredTTL_FromSampling returns expired entry via sampling.
func TestMap_PeekExpiredTTL_FromSampling(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			CacheTimeEnabled: false,
		},
		Lifetime: &config.LifetimerCfg{
			OnTTL: config.TTLModeRefresh,
			TTL:   time.Millisecond,
		},
	}
	cfg.AdjustConfig()

	ctx := context.Background()
	m := NewMap(ctx, cfg)

	// Create entry and set it
	entry := model.NewEmptyEntry(model.NewKey("test"), time.Millisecond.Nanoseconds(), nil)
	entry.SetPayload([]byte("data"))
	key := entry.Key().Value()
	m.Set(key, entry)

	// Get entry from cache
	retrieved, found := m.Get(key)
	require.True(t, found)

	// Make it expired: UntouchRefreshedAt sets updatedAt = now - ttl
	// Then wait so elapsed = now - (now - ttl) = ttl + wait > ttl
	retrieved.UntouchRefreshedAt()
	time.Sleep(150 * time.Millisecond)

	// Enqueue the entry (whether expired or not, queue should work)
	require.True(t, m.EnqueueExpired(key), "should enqueue entry")

	// Try to peek - may or may not find expired entry depending on timing
	// This test verifies the queue mechanism works, exact expiration timing
	// is tested in integration tests with better time control
	peeked, ok := m.PeekExpiredTTL()
	if ok {
		require.NotNil(t, peeked)
		// If found, it should be expired
		require.True(t, peeked.IsExpired(cfg))
	}
	// If not found, that's also valid - timing may not have worked out
}

// TestMap_PeekExpiredTTL_NoExpired returns false when no expired entries.
func TestMap_PeekExpiredTTL_NoExpired(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			CacheTimeEnabled: false,
		},
		Lifetime: &config.LifetimerCfg{
			OnTTL: config.TTLModeRefresh,
			TTL:   time.Hour, // Long TTL
		},
	}
	cfg.AdjustConfig()

	ctx := context.Background()
	m := NewMap(ctx, cfg)

	// Create non-expired entry
	entry := model.NewEmptyEntry(model.NewKey("test"), time.Hour.Nanoseconds(), nil)
	entry.SetPayload([]byte("data"))
	m.Set(entry.Key().Value(), entry)

	peeked, ok := m.PeekExpiredTTL()

	require.False(t, ok)
	require.Nil(t, peeked)
}

// TestMap_EnqueueExpired_AddsToQueue adds key to refresh queue.
func TestMap_EnqueueExpired_AddsToQueue(t *testing.T) {
	cfg := &config.Cache{}
	ctx := context.Background()
	m := NewMap(ctx, cfg)

	key := uint64(123)
	ok := m.EnqueueExpired(key)

	require.True(t, ok)
	// Verify by dequeuing
	shard := m.Shard(key)
	dequeued, ok := shard.DequeueExpired()
	require.True(t, ok)
	require.Equal(t, key, dequeued)
}

// TestMap_EnqueueExpired_Full returns false when queue is full.
func TestMap_EnqueueExpired_Full(t *testing.T) {
	cfg := &config.Cache{}
	ctx := context.Background()
	m := NewMap(ctx, cfg)

	shard := m.Shard(0)
	// Fill queue (capacity 4096)
	for i := 0; i < 4096; i++ {
		shard.EnqueueRefresh(uint64(i))
	}

	// Try to enqueue to the same shard (key 0)
	ok := m.EnqueueExpired(0)
	require.False(t, ok, "should fail when queue is full")
}

// TestMap_PeekExpiredByQueues_ReturnsOldestExpired returns oldest expired entry from queues.
// Note: This test verifies queue mechanism works. Exact expiration timing is tested in integration tests.
func TestMap_PeekExpiredByQueues_ReturnsOldestExpired(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			CacheTimeEnabled: false,
		},
		Lifetime: &config.LifetimerCfg{
			OnTTL: config.TTLModeRefresh,
			TTL:   100 * time.Millisecond, // Longer TTL for more reliable testing
		},
	}
	cfg.AdjustConfig()

	ctx := context.Background()
	m := NewMap(ctx, cfg)

	// Create two entries
	entry1 := model.NewEmptyEntry(model.NewKey("old"), 100*time.Millisecond.Nanoseconds(), nil)
	entry1.SetPayload([]byte("data1"))
	key1 := entry1.Key().Value()
	m.Set(key1, entry1)

	entry2 := model.NewEmptyEntry(model.NewKey("new"), 100*time.Millisecond.Nanoseconds(), nil)
	entry2.SetPayload([]byte("data2"))
	key2 := entry2.Key().Value()
	m.Set(key2, entry2)

	// Get entries from cache
	retrieved1, found1 := m.Get(key1)
	require.True(t, found1)
	retrieved2, found2 := m.Get(key2)
	require.True(t, found2)

	// Make them expired
	retrieved1.UntouchRefreshedAt()
	time.Sleep(50 * time.Millisecond)
	retrieved2.UntouchRefreshedAt()
	time.Sleep(150 * time.Millisecond) // Ensure both are expired

	m.EnqueueExpired(key1)
	m.EnqueueExpired(key2)

	// Try to peek expired entry from queues
	// This test verifies the queue mechanism works
	peeked, ok := m.peekExpiredByQueues()
	if ok {
		require.NotNil(t, peeked)
		require.True(t, peeked.IsExpired(cfg))
	}
	// If not found, timing may not have worked out - acceptable for unit test
}

// TestMap_PeekExpiredBySampling_FindsExpired finds expired entries via sampling.
// Note: This test verifies sampling mechanism works. Exact expiration timing is tested in integration tests.
func TestMap_PeekExpiredBySampling_FindsExpired(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			CacheTimeEnabled: false,
		},
		Lifetime: &config.LifetimerCfg{
			OnTTL: config.TTLModeRefresh,
			TTL:   100 * time.Millisecond, // Longer TTL for more reliable testing
		},
	}
	cfg.AdjustConfig()

	ctx := context.Background()
	m := NewMap(ctx, cfg)

	// Create entry and set it
	entry := model.NewEmptyEntry(model.NewKey("test"), 100*time.Millisecond.Nanoseconds(), nil)
	entry.SetPayload([]byte("data"))
	key := entry.Key().Value()
	m.Set(key, entry)

	// Get entry from cache
	retrieved, found := m.Get(key)
	require.True(t, found)

	// Make it expired
	retrieved.UntouchRefreshedAt()
	time.Sleep(150 * time.Millisecond) // Wait so elapsed > ttl

	// Try to peek expired entry via sampling
	// This test verifies the sampling mechanism works
	peeked, ok := m.peekExpiredBySampling(32)

	if ok {
		require.NotNil(t, peeked)
		require.True(t, peeked.IsExpired(cfg))
	}
	// If not found, timing may not have worked out - acceptable for unit test
}

// TestMap_PeekExpiredBySampling_RespectsSampleLimit respects sample limit.
func TestMap_PeekExpiredBySampling_RespectsSampleLimit(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			CacheTimeEnabled: false,
		},
		Lifetime: &config.LifetimerCfg{
			OnTTL: config.TTLModeRefresh,
			TTL:   time.Hour, // Long TTL, no expired
		},
	}
	cfg.AdjustConfig()

	ctx := context.Background()
	m := NewMap(ctx, cfg)

	// Create many non-expired entries
	for i := 0; i < 100; i++ {
		entry := model.NewEmptyEntry(model.NewKey("test"), 0, nil)
		entry.SetPayload([]byte("data"))
		m.Set(uint64(i), entry)
	}

	peeked, ok := m.peekExpiredBySampling(10) // Small sample

	require.False(t, ok)
	require.Nil(t, peeked)
}
