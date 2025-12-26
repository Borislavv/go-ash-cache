package cachedtime

import (
	"context"
	"sync/atomic"
	"time"
)

var (
	cacheTimeEach = time.Millisecond * 10
	nowUnix       atomic.Int64
	doneCh        = make(chan struct{})
)

func init() {
	nowUnix.Store(time.Now().UnixNano())
	t := time.NewTicker(cacheTimeEach)
	doneCh = make(chan struct{})
	go func() {
		for {
			select {
			case tt := <-t.C:
				nowUnix.Store(tt.UnixNano())
			case <-doneCh:
				t.Stop()
				return
			}
		}
	}()
}
func Now() time.Time {
	if doneCh == nil {
		return time.Now()
	}
	return time.Unix(0, nowUnix.Load())
}
func UnixNano() int64 {
	if doneCh == nil {
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
		if doneCh != nil {
			close(doneCh)
			doneCh = nil
		}
	}()
}
