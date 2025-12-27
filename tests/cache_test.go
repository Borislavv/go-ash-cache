package tests

import (
	"fmt"
	"github.com/Borislavv/go-ash-cache"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"github.com/Borislavv/go-ash-cache/tests/help"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
)

func TestCache(t *testing.T) {
	cache := ashcache.New(t.Context(), help.Cfg(), help.Logger())

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
	cache := ashcache.New(t.Context(), help.Cfg(), help.Logger())

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
	cache := ashcache.New(t.Context(), help.Cfg(), help.Logger())

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
