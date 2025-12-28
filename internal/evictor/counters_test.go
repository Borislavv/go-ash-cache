package evictor

import (
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

// TestEvictorCounters_Snapshot verifies that evictor counters correctly track metrics.
func TestEvictorCounters_Snapshot(t *testing.T) {
	c := newEvictorCounters()

	// Initial snapshot should be zero
	scans, hits, items, bytes := c.snapshot()
	require.Equal(t, int64(0), scans)
	require.Equal(t, int64(0), hits)
	require.Equal(t, int64(0), items)
	require.Equal(t, int64(0), bytes)

	// Increment counters
	c.scans.Add(100)
	c.scanHits.Add(50)
	c.evictedItems.Add(25)
	c.evictedBytes.Add(51200)

	// Snapshot should reflect increments
	scans, hits, items, bytes = c.snapshot()
	require.Equal(t, int64(100), scans)
	require.Equal(t, int64(50), hits)
	require.Equal(t, int64(25), items)
	require.Equal(t, int64(51200), bytes)
}

// TestEvictorCounters_Concurrent verifies thread-safety.
func TestEvictorCounters_Concurrent(t *testing.T) {
	c := newEvictorCounters()

	const numGoroutines = 10
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				c.scans.Add(1)
				c.scanHits.Add(1)
				c.evictedItems.Add(1)
				c.evictedBytes.Add(1024)
			}
		}()
	}

	wg.Wait()

	scans, hits, items, bytes := c.snapshot()
	require.Equal(t, int64(numGoroutines*opsPerGoroutine), scans)
	require.Equal(t, int64(numGoroutines*opsPerGoroutine), hits)
	require.Equal(t, int64(numGoroutines*opsPerGoroutine), items)
	require.Equal(t, int64(numGoroutines*opsPerGoroutine*1024), bytes)
}
