package cache

import (
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

// TestCounters_Snapshot verifies that counters correctly track and snapshot metrics.
func TestCounters_Snapshot(t *testing.T) {
	c := newCounters()

	// Initial snapshot should be zero
	allowed, notAllowed, items, bytes := c.snapshot()
	require.Equal(t, int64(0), allowed)
	require.Equal(t, int64(0), notAllowed)
	require.Equal(t, int64(0), items)
	require.Equal(t, int64(0), bytes)

	// Increment counters
	c.admissionAllowed.Add(10)
	c.admissionNotAllowed.Add(5)
	c.evictedHardLimitItems.Add(3)
	c.evictedHardLimitBytes.Add(1024)

	// Snapshot should reflect increments
	allowed, notAllowed, items, bytes = c.snapshot()
	require.Equal(t, int64(10), allowed)
	require.Equal(t, int64(5), notAllowed)
	require.Equal(t, int64(3), items)
	require.Equal(t, int64(1024), bytes)
}

// TestCounters_Concurrent verifies that counters are thread-safe.
func TestCounters_Concurrent(t *testing.T) {
	c := newCounters()

	const numGoroutines = 10
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				c.admissionAllowed.Add(1)
				c.admissionNotAllowed.Add(1)
				c.evictedHardLimitItems.Add(1)
				c.evictedHardLimitBytes.Add(100)
			}
		}()
	}

	wg.Wait()

	// Verify final counts
	allowed, notAllowed, items, bytes := c.snapshot()
	require.Equal(t, int64(numGoroutines*opsPerGoroutine), allowed)
	require.Equal(t, int64(numGoroutines*opsPerGoroutine), notAllowed)
	require.Equal(t, int64(numGoroutines*opsPerGoroutine), items)
	require.Equal(t, int64(numGoroutines*opsPerGoroutine*100), bytes)
}
