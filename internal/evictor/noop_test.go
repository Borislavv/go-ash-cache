package evictor

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TestNoOpEvictor_ForceCall returns nil immediately.
func TestNoOpEvictor_ForceCall(t *testing.T) {
	var ev NoOpEvictor

	err := ev.ForceCall(time.Second)
	require.NoError(t, err)
}

// TestNoOpEvictor_EvictorMetrics returns zero values.
func TestNoOpEvictor_EvictorMetrics(t *testing.T) {
	var ev NoOpEvictor

	scans, hits, items, bytes := ev.EvictorMetrics()
	require.Equal(t, int64(0), scans)
	require.Equal(t, int64(0), hits)
	require.Equal(t, int64(0), items)
	require.Equal(t, int64(0), bytes)
}

// TestNoOpEvictor_Close returns nil.
func TestNoOpEvictor_Close(t *testing.T) {
	var ev NoOpEvictor

	err := ev.Close()
	require.NoError(t, err)
}
