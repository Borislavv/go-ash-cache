package lifetimer

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// TestNoOpLifetimer_LifetimerMetrics returns zero values.
func TestNoOpLifetimer_LifetimerMetrics(t *testing.T) {
	var lt NoOpLifetimer

	affected, errors, scans, hits, misses := lt.LifetimerMetrics()
	require.Equal(t, int64(0), affected)
	require.Equal(t, int64(0), errors)
	require.Equal(t, int64(0), scans)
	require.Equal(t, int64(0), hits)
	require.Equal(t, int64(0), misses)
}

// TestNoOpLifetimer_Close returns nil.
func TestNoOpLifetimer_Close(t *testing.T) {
	var lt NoOpLifetimer

	err := lt.Close()
	require.NoError(t, err)
}
