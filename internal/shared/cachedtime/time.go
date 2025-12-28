package cachedtime

import (
	"context"
	"sync/atomic"
	"time"
)

const cacheTimeEach = 10 * time.Millisecond

var (
	nowUnix atomic.Int64
	closed  atomic.Bool
	doneCh  = make(chan struct{})
)

func init() {
	nowUnix.Store(time.Now().UnixNano())
	ticker := time.NewTicker(cacheTimeEach)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case tt, ok := <-ticker.C:
				if !ok {
					// never, but for robust behavior if the go Ticker will be changed in further versions
					// don't cache nil value of time.Time never
					return
				}
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

func CloseByCtx(ctx context.Context) {
	go func() {
		<-ctx.Done()
		if closed.CompareAndSwap(false, true) {
			// we are only one how closing the channel
			close(doneCh)
		}
	}()
}
