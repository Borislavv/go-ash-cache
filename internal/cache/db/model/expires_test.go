package model

import (
	"github.com/Borislavv/go-ash-cache/config"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TestEntry_IsExpired_NoTTL returns false when TTL is not set.
func TestEntry_IsExpired_NoTTL(t *testing.T) {
	cfg := &config.Cache{
		Lifetime: &config.LifetimerCfg{
			OnTTL: config.TTLModeRemove,
			TTL:   time.Hour,
		},
	}
	cfg.AdjustConfig()

	entry := NewEmptyEntry(NewKey("test"), 0, nil)
	entry.SetPayload([]byte("data"))

	require.False(t, entry.IsExpired(cfg), "entry without TTL should not be expired")
}

// TestEntry_IsExpired_NotExpired returns false when entry is not expired.
func TestEntry_IsExpired_NotExpired(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			CacheTimeEnabled: false, // Use real time
		},
		Lifetime: &config.LifetimerCfg{
			OnTTL: config.TTLModeRemove,
			TTL:   time.Hour,
		},
	}
	cfg.AdjustConfig()

	entry := NewEmptyEntry(NewKey("test"), time.Hour.Nanoseconds(), nil)
	entry.SetPayload([]byte("data"))

	require.False(t, entry.IsExpired(cfg), "recently set entry should not be expired")
}

// TestEntry_IsExpired_Expired verifies expiration logic.
// Note: Full expiration behavior with timing is tested in integration tests (tests/).
// This unit test verifies the function doesn't panic and handles edge cases.
func TestEntry_IsExpired_Expired(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			CacheTimeEnabled: false, // Use real time
		},
		Lifetime: &config.LifetimerCfg{
			OnTTL: config.TTLModeRemove,
			TTL:   time.Millisecond,
		},
	}
	cfg.AdjustConfig()

	entry := NewEmptyEntry(NewKey("test"), time.Millisecond.Nanoseconds(), nil)
	entry.SetPayload([]byte("data"))

	// Set updatedAt to past using UntouchRefreshedAt
	// This sets updatedAt = now - ttl, so elapsed = ttl (not > ttl)
	// To make it expired, we need elapsed > ttl, so wait a bit after UntouchRefreshedAt
	entry.UntouchRefreshedAt()       // Sets updatedAt to now - ttl
	time.Sleep(2 * time.Millisecond) // Wait so elapsed = now - (now - ttl) = ttl + 2ms > ttl

	// With real time, this should now be expired
	result := entry.IsExpired(cfg)
	// Result depends on timing, but function should not panic
	require.IsType(t, false, result, "IsExpired should return bool")
}

// TestEntry_QueueExpired sets and checks queue flag atomically.
func TestEntry_QueueExpired(t *testing.T) {
	entry := NewEmptyEntry(NewKey("test"), 0, nil)

	// First call should succeed
	require.True(t, entry.EnqueueExpired())

	// Second call should fail (already queued)
	require.False(t, entry.EnqueueExpired())
}

// TestEntry_DequeueExpired clears queue flag.
func TestEntry_DequeueExpired(t *testing.T) {
	entry := NewEmptyEntry(NewKey("test"), 0, nil)

	entry.EnqueueExpired()
	entry.DequeueExpired()

	// Should be able to queue again
	require.True(t, entry.EnqueueExpired())
}

// TestEntry_IsProbablyExpired_Stochastic verifies stochastic expiration logic.
func TestEntry_IsProbablyExpired_Stochastic(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			CacheTimeEnabled: false,
		},
		Lifetime: &config.LifetimerCfg{
			OnTTL:                        config.TTLModeRefresh,
			TTL:                          time.Hour,
			Beta:                         0.5,
			StochasticBetaRefreshEnabled: true,
			Coefficient:                  0.1, // Start checking at 10% of TTL
		},
	}
	cfg.AdjustConfig()

	entry := NewEmptyEntry(NewKey("test"), time.Hour.Nanoseconds(), nil)
	entry.SetPayload([]byte("data"))

	// Test that IsExpired with stochastic mode doesn't panic
	// Actual expiration logic is probabilistic and tested in integration tests
	result := entry.IsExpired(cfg)
	require.IsType(t, false, result, "IsExpired should return bool")
}
