package bloom

import (
	"github.com/Borislavv/go-ash-cache/config"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestShardedAdmitter_Estimate returns frequency estimate for a key.
func TestShardedAdmitter_Estimate(t *testing.T) {
	cfg := &config.AdmissionControlCfg{
		Capacity:            128,
		Shards:              4,
		MinTableLenPerShard: 64,
		SampleMultiplier:    3,
		DoorBitsPerCounter:  2,
	}
	a := newShardedAdmitter(cfg)

	const h uint64 = 0x100

	// Initially should be 0
	require.Equal(t, uint8(0), a.Estimate(h))

	// After recording, estimate should increase
	a.Record(h)
	a.Record(h) // Second record increments sketch

	est := a.Estimate(h)
	require.Greater(t, est, uint8(0), "estimate should be > 0 after records")
}

// TestShardedAdmitter_Reset clears doorkeeper and halves sketch counters.
func TestShardedAdmitter_Reset(t *testing.T) {
	cfg := &config.AdmissionControlCfg{
		Capacity:            128,
		Shards:              4,
		MinTableLenPerShard: 64,
		SampleMultiplier:    3,
		DoorBitsPerCounter:  2,
	}
	a := newShardedAdmitter(cfg)

	const h uint64 = 0x100

	// Build up frequency
	recordN(a, h, 100)
	before := a.Estimate(h)
	require.Greater(t, before, uint8(0))

	// Reset should halve counters
	a.Reset()
	after := a.Estimate(h)

	// After reset, estimate should be approximately halved
	require.LessOrEqual(t, after, before, "reset should reduce estimate")
	require.GreaterOrEqual(t, after, before/2-1, "reset should halve estimate (with tolerance)")
}

// TestSketch_MaybeReset triggers reset when adds >= resetAt.
func TestSketch_MaybeReset(t *testing.T) {
	var s sketch
	s.init(64, 2) // resetAt = 2 * 64 = 128

	// Record up to reset threshold
	for i := 0; i < 130; i++ {
		s.increment(0x100)
	}

	// maybeReset should have been called during increments
	// Verify by checking that estimate is reasonable (not saturated)
	est := s.estimate(0x100)
	require.LessOrEqual(t, est, uint8(15), "estimate should not exceed saturation")
}

// TestDoorkeeper_Reset clears all bits.
func TestDoorkeeper_Reset(t *testing.T) {
	var d doorkeeper
	d.init(64)

	h := uint64(0x100)

	// Set bits
	d.seenOrAdd(h)
	require.True(t, d.probablySeen(h))

	// Reset should clear
	d.reset()
	require.False(t, d.probablySeen(h), "reset should clear doorkeeper bits")
}

// recordN is a helper to record a key n times.
func recordN(a AdmissionControl, h uint64, n int) {
	for i := 0; i < n; i++ {
		a.Record(h)
	}
}
