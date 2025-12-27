package evictor

import "time"

// NoOpEvictor is a no-op implementation of Evictor.
// It performs no eviction and reports zero metrics.
type NoOpEvictor struct{}

// ForceCall does nothing and returns nil immediately.
func (NoOpEvictor) ForceCall(timeout time.Duration) error {
	return nil
}

// LifetimerMetrics always returns zero values.
func (NoOpEvictor) EvictorMetrics() (scans, hits, evictedItems, evictedBytes int64) {
	return 0, 0, 0, 0
}

// Close does nothing and returns nil.
func (NoOpEvictor) Close() error {
	return nil
}
