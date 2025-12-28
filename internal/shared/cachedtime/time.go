package cachedtime

import (
	"context"
	"github.com/Borislavv/go-ash-cache/config"
	"sync/atomic"
	"time"
)

const cacheTimeEach = 10 * time.Millisecond

var (
	nowUnix atomic.Int64
	closed  atomic.Bool
	doneCh  = make(chan struct{})
)

func RunIfEnabled(ctx context.Context, cfg *config.Cache) {
	if !cfg.DB.CacheTimeEnabled {
		return
	}
	cancelDeferredByCtx(ctx)

	nowUnix.Store(time.Now().UnixNano())
	ticker := time.NewTicker(cacheTimeEach)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case tt := <-ticker.C:
				nowUnix.Store(tt.UnixNano())
			case <-doneCh:
				return
			}
		}
	}()
}

func Now() time.Time {
	if closed.Load() {
		return time.Now()
	}
	return time.Unix(0, nowUnix.Load())
}

func UnixNano() int64 {
	if closed.Load() {
		return time.Now().UnixNano()
	}
	return nowUnix.Load()
}

func Since(t time.Time) time.Duration {
	return Now().Sub(t)
}

func cancelDeferredByCtx(ctx context.Context) {
	go func() {
		<-ctx.Done()
		if closed.CompareAndSwap(false, true) {
			// we are only one how closing the channel
			close(doneCh)
		}
	}()
}
