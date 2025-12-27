package tests

import (
	"fmt"
	"github.com/Borislavv/go-ash-cache"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"github.com/Borislavv/go-ash-cache/internal/config"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func defaultCfg() *config.Cache {
	return &config.Cache{
		Lifetime: &config.LifetimerCfg{
			OnTTL:         config.TTLModeRefresh,
			TTL:           time.Minute * 5,
			Rate:          1000,
			Beta:          0.5,
			Coefficient:   0.5,
			IsRemoveOnTTL: false,
		},
		Eviction: &config.EvictionCfg{
			LRUMode:              config.LRUModeListing,
			SoftLimitCoefficient: 0.8,
			SoftMemoryLimitBytes: 1024 * 1024 * 800,
			CallsPerSec:          5,
			BackoffSpinsPerCall:  1024,
			IsListing:            true,
		},
		DB: config.DBCfg{
			SizeBytes:              1024 * 1024 * 1024,
			IsTelemetryLogsEnabled: true,
			TelemetryLogsInterval:  time.Second * 5,
		},
	}
}

func defaultLogger() *slog.Logger {
	// Level can come from config/env; Info is a good production default.
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		// AddSource: false, // keep off in prod unless you need it
	}

	h := slog.NewJSONHandler(os.Stdout, opts)

	log := slog.New(h).With(
		slog.String("service", "ashCache"),
		slog.String("env", "test"),
	)

	// Optional: make it the default logger used by slog.Info/Debug/etc.
	slog.SetDefault(log)

	return log
}

func TestCache(t *testing.T) {
	cache := ashcache.New(t.Context(), defaultCfg(), defaultLogger())

	var (
		err      error
		payload  []byte
		invokes  uint64
		testResp = []byte("test response")
	)
	for i := 0; i < 1000; i++ {
		payload, err = cache.Get("hello_world", func(item model.AshItem) (resp []byte, respErr error) {
			atomic.AddUint64(&invokes, 1)
			return testResp, nil
		})
		require.NoError(t, err)
	}

	require.Equal(t, testResp, payload)
	require.Equal(t, uint64(1), atomic.LoadUint64(&invokes))
}

func TestCacheKeyRespected(t *testing.T) {
	cache := ashcache.New(t.Context(), defaultCfg(), defaultLogger())

	var (
		err      error
		payload  []byte
		invokes  uint64
		testResp = "test response: #%d"
	)
	for i := 0; i < 1000; i++ {
		payload, err = cache.Get(fmt.Sprintf("hello_world_%d", i), func(item model.AshItem) (resp []byte, respErr error) {
			atomic.AddUint64(&invokes, 1)
			return []byte(fmt.Sprintf(testResp, i)), nil
		})
		require.NoError(t, err)
	}

	require.Equal(t, []byte(fmt.Sprintf(testResp, 999)), payload)
	require.Equal(t, uint64(1000), atomic.LoadUint64(&invokes))
}

func TestCacheErrPropagates(t *testing.T) {
	cache := ashcache.New(t.Context(), defaultCfg(), defaultLogger())

	var invokes uint64
	for i := 0; i < 1000; i++ {
		_, err := cache.Get(fmt.Sprintf("hello_world_%d", i), func(item model.AshItem) (resp []byte, respErr error) {
			atomic.AddUint64(&invokes, 1)
			return nil, fmt.Errorf("error #%d", i)
		})
		require.Errorf(t, err, "error #%d", i)
	}

	require.Equal(t, uint64(1000), atomic.LoadUint64(&invokes), fmt.Sprintf("expected: 999, actual: %d", invokes))
}
