package cachedtime

import (
	"context"
	"sync/atomic"
	"time"
)

var (
	cacheTimeEach = time.Millisecond * 10
	nowUnix       atomic.Int64
	doneChPtr     = atomic.Pointer[chan struct{}]{}
)

func init() {
	nowUnix.Store(time.Now().UnixNano())
	t := time.NewTicker(cacheTimeEach)
	doneCh := make(chan struct{})
	doneChPtr.Store(&doneCh)
	go func() {
		for {
			select {
			case tt := <-t.C:
				nowUnix.Store(tt.UnixNano())
			case <-*doneChPtr.Load():
				t.Stop()
				return
			}
		}
	}()
}
func Now() time.Time {
	if doneChPtr.Load() == nil {
		return time.Now()
	}
	return time.Unix(0, nowUnix.Load())
}
func UnixNano() int64 {
	if doneChPtr.Load() == nil {
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
		if doneChPtr.Load() != nil {
			close(*doneChPtr.Load())
			doneChPtr.Store(nil)
		}
	}()
}
