package bloom

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// TestNoOp_Record does nothing (no-op).
func TestNoOp_Record(t *testing.T) {
	noop := newNoOp()
	// Should not panic
	noop.Record(123)
	noop.Record(456)
}

// TestNoOp_Allow always returns true.
func TestNoOp_Allow(t *testing.T) {
	noop := newNoOp()

	require.True(t, noop.Allow(1, 2))
	require.True(t, noop.Allow(100, 200))
	require.True(t, noop.Allow(0, 0))
}

// TestNoOp_Estimate always returns 0.
func TestNoOp_Estimate(t *testing.T) {
	noop := newNoOp()

	require.Equal(t, uint8(0), noop.Estimate(1))
	require.Equal(t, uint8(0), noop.Estimate(100))
	require.Equal(t, uint8(0), noop.Estimate(0))
}

// TestNoOp_Reset does nothing (no-op).
func TestNoOp_Reset(t *testing.T) {
	noop := newNoOp()
	// Should not panic
	noop.Reset()
	noop.Record(123)
	noop.Reset()
}
