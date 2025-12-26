package cache

import "sync/atomic"

type counters struct {
	admissionAllowed      atomic.Int64
	admissionNotAllowed   atomic.Int64
	evictedHardLimitItems atomic.Int64
	evictedHardLimitBytes atomic.Int64
}

func newCounters() *counters {
	return &counters{
		admissionAllowed:      atomic.Int64{},
		admissionNotAllowed:   atomic.Int64{},
		evictedHardLimitItems: atomic.Int64{},
		evictedHardLimitBytes: atomic.Int64{},
	}
}

func (c *counters) snapshot() (allowed, notAllowed, hardEvictedItems, hardEvictedBytes int64) {
	return c.admissionAllowed.Load(), c.admissionNotAllowed.Load(), c.evictedHardLimitItems.Load(), c.evictedHardLimitBytes.Load()
}
