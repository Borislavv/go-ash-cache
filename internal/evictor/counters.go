package evictor

import "sync/atomic"

type evictorCounters struct {
	scans        atomic.Int64
	scanHits     atomic.Int64
	evictedItems atomic.Int64
	evictedBytes atomic.Int64
}

func (c *evictorCounters) snapshot() (scans, hits, evictedItems, evictedBytes int64) {
	return c.scans.Load(), c.scanHits.Load(), c.evictedItems.Load(), c.evictedBytes.Load()
}

func newEvictorCounters() *evictorCounters {
	return &evictorCounters{
		scans:        atomic.Int64{},
		scanHits:     atomic.Int64{},
		evictedItems: atomic.Int64{},
		evictedBytes: atomic.Int64{},
	}
}
