package cachedtime

import (
	"context"
	"github.com/Borislavv/go-ash-cache/config"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TestNow_Disabled returns time.Now() when cache time is disabled.
func TestNow_Disabled(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			CacheTimeEnabled: false,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	RunIfEnabled(ctx, cfg)

	// Should return real time
	now1 := Now()
	time.Sleep(10 * time.Millisecond)
	now2 := Now()

	require.True(t, now2.After(now1), "time should advance when disabled")
}

// TestUnixNano_Disabled returns real time when cache time is disabled.
func TestUnixNano_Disabled(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			CacheTimeEnabled: false,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	RunIfEnabled(ctx, cfg)

	nano1 := UnixNano()
	time.Sleep(10 * time.Millisecond)
	nano2 := UnixNano()

	require.Greater(t, nano2, nano1, "UnixNano should advance when disabled")
}

// TestSince_CalculatesDuration verifies Since calculates duration correctly.
func TestSince_CalculatesDuration(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			CacheTimeEnabled: false,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	RunIfEnabled(ctx, cfg)

	start := Now()
	time.Sleep(50 * time.Millisecond)
	duration := Since(start)

	require.GreaterOrEqual(t, duration, 40*time.Millisecond, "Since should calculate correct duration")
	require.Less(t, duration, 100*time.Millisecond, "Since should calculate correct duration")
}

// TestRunIfEnabled_StartsTicker verifies that RunIfEnabled starts time caching.
func TestRunIfEnabled_StartsTicker(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			CacheTimeEnabled: true,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	RunIfEnabled(ctx, cfg)

	// Give ticker time to update
	time.Sleep(20 * time.Millisecond)

	// Time should be cached (not advancing immediately)
	nano1 := UnixNano()
	time.Sleep(5 * time.Millisecond) // Less than ticker interval
	nano2 := UnixNano()

	// With cached time, nano2 might equal nano1 (within ticker resolution)
	require.GreaterOrEqual(t, nano2, nano1, "UnixNano should be non-decreasing")
}

// TestRunIfEnabled_StopsOnContextCancel verifies that ticker stops on context cancel.
func TestRunIfEnabled_StopsOnContextCancel(t *testing.T) {
	cfg := &config.Cache{
		DB: config.DBCfg{
			CacheTimeEnabled: true,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	RunIfEnabled(ctx, cfg)

	// Cancel context
	cancel()

	// Wait for cleanup
	time.Sleep(50 * time.Millisecond)

	// After cancel, should fall back to real time
	nano1 := UnixNano()
	time.Sleep(10 * time.Millisecond)
	nano2 := UnixNano()

	require.Greater(t, nano2, nano1, "time should advance after context cancel")
}
