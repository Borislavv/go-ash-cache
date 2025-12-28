package random

import (
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

// TestFloat64_ReturnsValidRange verifies that Float64 returns values in [0, 1).
func TestFloat64_ReturnsValidRange(t *testing.T) {
	for i := 0; i < 1000; i++ {
		val := Float64()
		require.GreaterOrEqual(t, val, 0.0, "Float64 should return >= 0")
		require.Less(t, val, 1.0, "Float64 should return < 1")
	}
}

// TestFloat64_Distribution verifies that Float64 produces diverse values.
func TestFloat64_Distribution(t *testing.T) {
	values := make(map[uint64]bool)
	for i := 0; i < 100; i++ {
		val := Float64()
		// Convert to integer bucket for uniqueness check
		bucket := uint64(val * 1000)
		values[bucket] = true
	}

	// Should have reasonable diversity (at least 50 unique buckets)
	require.Greater(t, len(values), 50, "Float64 should produce diverse values")
}

// TestInit_ConfiguresShards verifies that Init configures shards correctly.
func TestInit_ConfiguresShards(t *testing.T) {
	// Test with explicit count
	Init(8)
	val1 := Float64()

	// Reinitialize with different count
	Init(16)
	val2 := Float64()

	// Both should be valid
	require.GreaterOrEqual(t, val1, 0.0)
	require.Less(t, val1, 1.0)
	require.GreaterOrEqual(t, val2, 0.0)
	require.Less(t, val2, 1.0)
}

// TestInit_ZeroOrNegative uses default (GOMAXPROCS*4).
func TestInit_ZeroOrNegative(t *testing.T) {
	Init(0)
	val := Float64()
	require.GreaterOrEqual(t, val, 0.0)
	require.Less(t, val, 1.0)

	Init(-1)
	val = Float64()
	require.GreaterOrEqual(t, val, 0.0)
	require.Less(t, val, 1.0)
}

// TestFloat64_Concurrent verifies thread-safety.
func TestFloat64_Concurrent(t *testing.T) {
	const numGoroutines = 10
	const callsPerGoroutine = 100

	results := make(chan float64, numGoroutines*callsPerGoroutine)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				results <- Float64()
			}
		}()
	}

	wg.Wait()
	close(results)

	// Verify all results are in valid range
	for val := range results {
		require.GreaterOrEqual(t, val, 0.0)
		require.Less(t, val, 1.0)
	}
}
