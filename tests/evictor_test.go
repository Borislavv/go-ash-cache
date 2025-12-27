package tests

import (
	"context"
	"fmt"
	ashcache "github.com/Borislavv/go-ash-cache"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"github.com/Borislavv/go-ash-cache/internal/shared/bytes"
	"github.com/Borislavv/go-ash-cache/tests/help"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestEvictorListingEviction(t *testing.T) {
	evictionTestCfg := help.EvictionCfg()
	evictionTestCfg.Eviction.IsListing = true
	cache := ashcache.New(t.Context(), evictionTestCfg, help.Logger())

	// attempt to load 10mb in cache
	const wightKB = 100 * 1024
	for i := 0; i < 100; i++ {
		data, err := cache.Get(fmt.Sprintf("key-%d", i), func(item model.AshItem) ([]byte, error) {
			data := make([]byte, wightKB)
			return data, nil
		})
		require.NoError(t, err)
		require.Len(t, data, wightKB)
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second*30)
	defer cancel()

	checkEach := time.NewTicker(time.Millisecond * 100)
	defer checkEach.Stop()

	expectedMemory := evictionTestCfg.Eviction.SoftMemoryLimitBytes
	expectedLength := evictionTestCfg.Eviction.SoftMemoryLimitBytes / wightKB

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("context deadline exceeded; test failed")
		case <-checkEach.C:
			memory := cache.Mem()
			length := cache.Len()
			if length <= expectedLength && memory <= expectedMemory {
				require.LessOrEqual(t, length, expectedLength, fmt.Sprintf("cache length - %d, memory - %s (expected length = 80)", length, bytes.FmtMem(uint64(memory))))
				require.LessOrEqual(t, memory, expectedMemory, fmt.Sprintf("cache length - %d, memory - %s (expected memory = %s)", length, bytes.FmtMem(uint64(memory)), bytes.FmtMem(uint64(evictionTestCfg.Eviction.SoftMemoryLimitBytes))))
				return
			}
		}
	}
}

func TestEvictorSamplingEviction(t *testing.T) {
	evictionTestCfg := help.EvictionCfg()
	evictionTestCfg.Eviction.IsListing = false
	cache := ashcache.New(t.Context(), evictionTestCfg, help.Logger())

	// attempt to load 10mb in cache when threshold is 8mb
	const wightKB = 100 * 1024
	for i := 0; i < 100; i++ {
		data, err := cache.Get(fmt.Sprintf("key-%d", i), func(item model.AshItem) ([]byte, error) {
			data := make([]byte, wightKB)
			return data, nil
		})
		require.NoError(t, err)
		require.Len(t, data, wightKB)
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second*30)
	defer cancel()

	checkEach := time.NewTicker(time.Millisecond * 100)
	defer checkEach.Stop()

	expectedMemory := evictionTestCfg.Eviction.SoftMemoryLimitBytes
	expectedLength := evictionTestCfg.Eviction.SoftMemoryLimitBytes / wightKB

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("context deadline exceeded; test failed")
		case <-checkEach.C:
			memory := cache.Mem()
			length := cache.Len()
			if length <= expectedLength && memory <= expectedMemory {
				require.LessOrEqual(t, length, expectedLength, fmt.Sprintf("cache length - %d, memory - %s (expected length = 80)", length, bytes.FmtMem(uint64(memory))))
				require.LessOrEqual(t, memory, expectedMemory, fmt.Sprintf("cache length - %d, memory - %s (expected memory = %s)", length, bytes.FmtMem(uint64(memory)), bytes.FmtMem(uint64(evictionTestCfg.Eviction.SoftMemoryLimitBytes))))
				return
			}
		}
	}
}
