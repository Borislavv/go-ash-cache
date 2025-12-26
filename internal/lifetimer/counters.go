package lifetimer

import "sync/atomic"

type lifetimerCounters struct {
	affected   atomic.Int64 // successful refresh/remove operations
	errors     atomic.Int64 // error refresh operations (remove don't return an error)
	scans      atomic.Int64 // total scans number
	scanHits   atomic.Int64 // scan hits
	scanMisses atomic.Int64 // scan misses
}

func newLifetimerCounters() *lifetimerCounters {
	return &lifetimerCounters{
		affected:   atomic.Int64{},
		errors:     atomic.Int64{},
		scans:      atomic.Int64{},
		scanHits:   atomic.Int64{},
		scanMisses: atomic.Int64{},
	}
}

func (c *lifetimerCounters) snapshot() (affected, errors, scans, hits, misses int64) {
	affected = c.affected.Load()
	errors = c.errors.Load()
	scans = c.scans.Load()
	hits = c.scanHits.Load()
	misses = c.scanMisses.Load()
	return
}
