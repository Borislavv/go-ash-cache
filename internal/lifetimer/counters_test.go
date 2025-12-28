package lifetimer

import (
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

// TestLifetimerCounters_Snapshot verifies that lifetimer counters correctly track metrics.
func TestLifetimerCounters_Snapshot(t *testing.T) {
	c := newLifetimerCounters()

	// Initial snapshot should be zero
	affected, errors, scans, hits, misses := c.snapshot()
	require.Equal(t, int64(0), affected)
	require.Equal(t, int64(0), errors)
	require.Equal(t, int64(0), scans)
	require.Equal(t, int64(0), hits)
	require.Equal(t, int64(0), misses)

	// Increment counters
	c.affected.Add(100)
	c.errors.Add(5)
	c.scans.Add(200)
	c.scanHits.Add(150)
	c.scanMisses.Add(50)

	// Snapshot should reflect increments
	affected, errors, scans, hits, misses = c.snapshot()
	require.Equal(t, int64(100), affected)
	require.Equal(t, int64(5), errors)
	require.Equal(t, int64(200), scans)
	require.Equal(t, int64(150), hits)
	require.Equal(t, int64(50), misses)
}

// TestLifetimerCounters_Concurrent verifies thread-safety.
func TestLifetimerCounters_Concurrent(t *testing.T) {
	c := newLifetimerCounters()

	const numGoroutines = 10
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				c.affected.Add(1)
				c.errors.Add(1)
				c.scans.Add(1)
				c.scanHits.Add(1)
				c.scanMisses.Add(1)
			}
		}()
	}

	wg.Wait()

	affected, errors, scans, hits, misses := c.snapshot()
	require.Equal(t, int64(numGoroutines*opsPerGoroutine), affected)
	require.Equal(t, int64(numGoroutines*opsPerGoroutine), errors)
	require.Equal(t, int64(numGoroutines*opsPerGoroutine), scans)
	require.Equal(t, int64(numGoroutines*opsPerGoroutine), hits)
	require.Equal(t, int64(numGoroutines*opsPerGoroutine), misses)
}
