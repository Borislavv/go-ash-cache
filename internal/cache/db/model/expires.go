package model

import (
	"github.com/Borislavv/go-ash-cache/config"
	"github.com/Borislavv/go-ash-cache/internal/shared/cachedtime"
	"github.com/Borislavv/go-ash-cache/internal/shared/random"
	"math"
	"sync/atomic"
)

// IsExpired - checks that elapsed time greater than TTL.
func (e *Entry) IsExpired(cfg *config.Cache) bool {
	if e == nil {
		return false
	}

	if cfg.Lifetime.Enabled() && cfg.Lifetime.StochasticBetaRefreshEnabled {
		return e.isProbablyExpired(cfg.Lifetime.Beta, cfg.Lifetime.Coefficient)
	}

	return e.isExpired()
}

func (e *Entry) isExpired() bool {
	if e == nil {
		return false
	}
	ttl := atomic.LoadInt64(&e.ttl)
	if ttl == 0 {
		return false
	}

	// Time since the last successful refresh.
	elapsed := cachedtime.UnixNano() - atomic.LoadInt64(&e.updatedAt)
	return elapsed > ttl
}

// isProbablyExpired implements probabilistic refresh logic ("beta" algorithm) and used while background refresh.
// Returns true if the entry is stale and, with a probability proportional to its staleness, should be refreshed now.
func (e *Entry) isProbablyExpired(beta, coefficient float64) bool {
	if e == nil {
		return false
	}
	i64TTL := atomic.LoadInt64(&e.ttl)
	if i64TTL == 0 {
		return false
	}

	var ttl = float64(i64TTL)
	// Time since the last successful refresh.
	elapsed := cachedtime.UnixNano() - atomic.LoadInt64(&e.updatedAt)
	// Hard floor: do nothing until elapsed >= coefficient * ttl.
	minStale := int64(ttl * coefficient)

	if minStale > elapsed {
		return false
	}

	// Lifetime probability via the exponential CDF:
	// p = 1 - exp(-beta * x). Larger beta -> steeper growth.
	probability := 1 - math.Exp(-beta*(float64(elapsed)/ttl))
	return random.Float64() < probability
}

func (e *Entry) EnqueueExpired() bool {
	return atomic.CompareAndSwapInt64(&e.isQueuedOnRefresh, 0, 1)
}

func (e *Entry) DequeueExpired() {
	atomic.StoreInt64(&e.isQueuedOnRefresh, 0)
}
