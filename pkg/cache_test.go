package pkg

import (
	"fmt"
	"github.com/Borislavv/go-ash-cache/internal/db/config"
	"github.com/Borislavv/go-ash-cache/internal/db/model"
	"github.com/stretchr/testify/require"
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
			SoftMemoryLimitBytes: 1024 * 1024 * 800,
			HardMemoryLimitBytes: 1024 * 1024 * 950,
			CallsPerSec:          5,
		},
		DB: &config.DbCfg{
			IsTelemetryLogsEnabled: true,
			Mode:                   "listing",
			Size:                   1024 * 1024 * 1024,
			IsListing:              true,
			SoftMemoryLimit:        1024 * 1024 * 800,
			HardMemoryLimit:        1024 * 1024 * 950,
		},
	}
}

func TestCache(t *testing.T) {
	cache := NewAshCache(t.Context(), defaultCfg())

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
	cache := NewAshCache(t.Context(), defaultCfg())

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
	cache := NewAshCache(t.Context(), defaultCfg())

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
