package lifetimer

// NoOpLifetimer is a no-op implementation of Lifetimer.
// It performs no lifetime management and reports zero metrics.
type NoOpLifetimer struct{}

// Metrics always returns zero values.
func (NoOpLifetimer) Metrics() (affected, errors, scans, hits, misses int64) {
	return 0, 0, 0, 0, 0
}

// Close does nothing and returns nil.
func (NoOpLifetimer) Close() error {
	return nil
}
