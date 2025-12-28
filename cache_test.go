package ashcache

import (
	"context"
	"github.com/Borislavv/go-ash-cache/config"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
)

// TestCache_Close cancels context and stops background workers.
func TestCache_Close(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Cache{
		DB: config.DBCfg{
			SizeBytes:        10 * 1024 * 1024,
			CacheTimeEnabled: true,
		},
		Eviction: &config.EvictionCfg{
			LRUMode:              config.LRUModeListing,
			SoftLimitCoefficient: 0.8,
			CallsPerSec:          10,
			BackoffSpinsPerCall:  1024,
		},
	}
	cfg.AdjustConfig()

	logger := slog.Default()
	cache := New(ctx, cfg, logger)

	// Close should not panic
	err := cache.Close()
	require.NoError(t, err)

	// Close should be idempotent
	err = cache.Close()
	require.NoError(t, err)
}
