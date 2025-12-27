package tests

import (
	"fmt"
	ashcache "github.com/Borislavv/go-ash-cache"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"github.com/Borislavv/go-ash-cache/internal/shared/bytes"
	"github.com/Borislavv/go-ash-cache/tests/help"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestEvictorEviction(t *testing.T) {
	evictionTestCfg := help.EvictionCfg()
	cache := ashcache.New(t.Context(), evictionTestCfg, help.Logger())

	// attempt to load 10mb in cache
	for i := 0; i < 100; i++ {
		const KB = 1024
		data, err := cache.Get(fmt.Sprintf("key-%d", i), func(item model.AshItem) ([]byte, error) {
			data := make([]byte, 100*KB)
			return data, nil
		})
		require.NoError(t, err)
		require.Len(t, data, 100*KB)
	}

	time.Sleep(3 * time.Second)
	length := cache.Len()
	memory := cache.Mem()
	require.Equal(t, int64(80), length, fmt.Sprintf("cache length - %d, memory - %s (expected length = 80)", length, bytes.FmtMem(uint64(memory))))
	require.LessOrEqual(t, evictionTestCfg.Eviction.SoftMemoryLimitBytes, memory, fmt.Sprintf("cache length - %d, memory - %s (expected memory = %s)", length, bytes.FmtMem(uint64(memory)), bytes.FmtMem(uint64(evictionTestCfg.Eviction.SoftMemoryLimitBytes))))
}
