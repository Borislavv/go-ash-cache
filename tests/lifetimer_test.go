package tests

import (
	"context"
	"fmt"
	ashcache "github.com/Borislavv/go-ash-cache"
	"github.com/Borislavv/go-ash-cache/model"
	"github.com/Borislavv/go-ash-cache/tests/help"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"
)

func TestLifetimerRefreshingEntries(t *testing.T) {
	lifetimerStochasticTestCfg := help.LifetimerRefreshCfg()
	cache := ashcache.New(t.Context(), lifetimerStochasticTestCfg, help.Logger())

	var refreshes = &atomic.Int64{}
	for i := 0; i < 100; i++ {
		_, err := cache.Get(fmt.Sprintf("key-%d", i), func(item model.Item) ([]byte, error) {
			item.SetTTL(time.Second * 2)
			data := make([]byte, 128)
			refreshes.Add(1)
			return data, nil
		})
		require.NoError(t, err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second*30)
	defer cancel()

	checkEach := time.NewTicker(time.Millisecond * 100)
	defer checkEach.Stop()

	const expectedAtLeastRefreshes = int64(100)
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("context deadline exceeded; test failed")
		case <-checkEach.C:
			curRefreshes := refreshes.Load()
			if curRefreshes >= expectedAtLeastRefreshes {
				require.GreaterOrEqual(t, curRefreshes, expectedAtLeastRefreshes)
				t.Logf("success - refreshes: %d, expected: %d", curRefreshes, expectedAtLeastRefreshes)
				return
			}
		}
	}
}

func TestLifetimerStochasticRefreshingEntries(t *testing.T) {
	lifetimerStochasticTestCfg := help.LifetimerRefreshStochasticCfg()
	cache := ashcache.New(t.Context(), lifetimerStochasticTestCfg, help.Logger())

	var refreshes = &atomic.Int64{}
	for i := 0; i < 100; i++ {
		_, err := cache.Get(fmt.Sprintf("key-%d", i), func(item model.Item) ([]byte, error) {
			item.SetTTL(time.Second * 2)
			data := make([]byte, 128)
			refreshes.Add(1)
			return data, nil
		})
		require.NoError(t, err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second*30)
	defer cancel()

	checkEach := time.NewTicker(time.Millisecond * 100)
	defer checkEach.Stop()

	const expectedAtLeastRefreshes = int64(1000)
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("context deadline exceeded; test failed")
		case <-checkEach.C:
			curRefreshes := refreshes.Load()
			if curRefreshes >= expectedAtLeastRefreshes {
				require.GreaterOrEqual(t, curRefreshes, expectedAtLeastRefreshes)
				t.Logf("success - refreshes: %d, expected: %d", curRefreshes, expectedAtLeastRefreshes)
				return
			}
		}
	}
}

func TestLifetimerRemoveAllEntries(t *testing.T) {
	lifetimerRemoveModelTestCfg := help.LifetimerRemoveCfg()
	cache := ashcache.New(t.Context(), lifetimerRemoveModelTestCfg, help.Logger())

	for i := 0; i < 100; i++ {
		_, err := cache.Get(fmt.Sprintf("key-%d", i), func(item model.Item) ([]byte, error) {
			item.SetTTL(time.Second)
			data := make([]byte, 128)
			return data, nil
		})
		require.NoError(t, err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second*30)
	defer cancel()

	checkEach := time.NewTicker(time.Millisecond * 100)
	defer checkEach.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("context deadline exceeded; test failed")
		case <-checkEach.C:
			curLength := cache.Len()
			if curLength == 0 {
				require.Equal(t, int64(0), curLength)
				return
			}
		}
	}
}

func TestLifetimerDontRemoveBeforeTTLExpired(t *testing.T) {
	lifetimerRemoveModelTestCfg := help.LifetimerRemoveCfg()
	cache := ashcache.New(t.Context(), lifetimerRemoveModelTestCfg, help.Logger())

	for i := 0; i < 100; i++ {
		_, err := cache.Get(fmt.Sprintf("key-%d", i), func(item model.Item) ([]byte, error) {
			item.SetTTL(time.Second * 6)
			data := make([]byte, 128)
			return data, nil
		})
		require.NoError(t, err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second*30)
	defer cancel()

	after := time.NewTicker(time.Second * 5)
	defer after.Stop()

	const expectedLength = int64(100)
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("context deadline exceeded; test failed")
		case <-after.C:
			curLength := cache.Len()
			if curLength == expectedLength {
				require.Equal(t, expectedLength, curLength)
				return
			}
			t.Fatalf("test conditions did not match")
		}
	}
}

func TestLifetimerStochasticRemoveBeforeTTLExpired(t *testing.T) {
	lifetimerRemoveModelTestCfg := help.LifetimerRemoveStochasticCfg()
	cache := ashcache.New(t.Context(), lifetimerRemoveModelTestCfg, help.Logger())

	for i := 0; i < 100; i++ {
		_, err := cache.Get(fmt.Sprintf("key-%d", i), func(item model.Item) ([]byte, error) {
			item.SetTTL(time.Second * 6)
			data := make([]byte, 128)
			return data, nil
		})
		require.NoError(t, err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second*30)
	defer cancel()

	after := time.NewTicker(time.Second * 5)
	defer after.Stop()

	const expectedLength = int64(0)
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("context deadline exceeded; test failed")
		case <-after.C:
			curLength := cache.Len()
			if curLength == expectedLength {
				require.Equal(t, expectedLength, curLength)
				return
			}
			t.Fatalf("test conditions did not match")
		}
	}
}
