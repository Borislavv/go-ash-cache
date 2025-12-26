package evictor

import (
	"context"
	"errors"
	"github.com/Borislavv/go-ash-cache/internal/cache"
	"github.com/Borislavv/go-ash-cache/internal/config"
	"log/slog"
	"runtime"
	"sync"
	"time"
)

var ErrEvictorNotResponded = errors.New("evictor not responded")

type Evictor interface {
	ForceCall(timeout time.Duration) error
	Metrics() (scans, hits, evictedItems, evictedBytes int64)
	Close() error
}

type EvictionWorker struct {
	ctx      context.Context
	cancel   context.CancelFunc
	cfg      *config.EvictionCfg
	logger   *slog.Logger
	cache    *cache.Cache
	counters *evictorCounters
	invokeCh chan struct{}
}

func New(
	ctx context.Context,
	cfg *config.EvictionCfg,
	logger *slog.Logger,
	cache *cache.Cache,
) Evictor {
	if !cfg.Enabled() {
		return &NoOpEvictor{}
	}

	ctx, cancel := context.WithCancel(ctx)
	return (&EvictionWorker{
		ctx:      ctx,
		cancel:   cancel,
		cfg:      cfg,
		logger:   logger,
		cache:    cache,
		counters: newEvictorCounters(),
		invokeCh: make(chan struct{}),
	}).run()
}

func (w *EvictionWorker) ForceCall(timeout time.Duration) error {
	after := time.NewTimer(timeout)
	defer after.Stop()

	select {
	case <-w.ctx.Done():
	case w.invokeCh <- struct{}{}:
	case <-after.C:
		return ErrEvictorNotResponded
	}
	return nil
}

func (w *EvictionWorker) Metrics() (scans, hits, evictedItems, evictedBytes int64) {
	return w.counters.snapshot()
}

func (w *EvictionWorker) Close() error {
	w.cancel()
	return nil
}

func (w *EvictionWorker) run() *EvictionWorker {
	w.logger.Info("evictor is running", "calls_per_sec", w.cfg.CallsPerSec, "backoff_spins", w.cfg.BackoffSpinsPerCall)

	go func() {
		defer w.logger.Info("evictor is stopped")
		var wg sync.WaitGroup
		for i := 0; i <= runtime.GOMAXPROCS(0); i++ {
			wg.Go(w.consumer)
		}
		wg.Go(w.provider)
		wg.Wait()
	}()

	return w
}

// provider - calls one of evictor workers when the memory overcome limit.
func (w *EvictionWorker) provider() {
	var evictionCallsPerSec = w.cfg.CallsPerSec
	if w.cfg.CallsPerSec <= 0 {
		evictionCallsPerSec = 1
	}

	each := time.Second / time.Duration(evictionCallsPerSec)
	tick := time.NewTicker(each)
	defer tick.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-tick.C:
			if w.cache.Len() > 0 && w.cache.Mem() > 0 {
				w.counters.scans.Add(1)
				if w.cache.SoftMemoryLimitOvercome() {
					select {
					case <-w.ctx.Done():
						return
					case w.invokeCh <- struct{}{}:
						w.counters.scanHits.Add(1)
					}
				}
			}
		}
	}
}

// consumer - evicts item from the cache until within limit or backoff by spins.
func (w *EvictionWorker) consumer() {
	var evictionSpinsBackoff = w.cfg.BackoffSpinsPerCall
	if w.cfg.BackoffSpinsPerCall <= 0 {
		const defaultEvictionSpinsBackoff = 2048
		evictionSpinsBackoff = defaultEvictionSpinsBackoff
	}

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.invokeCh:
			if w.cache.Len() > 0 && w.cache.Mem() > 0 {
				freedBytes, items := w.cache.SoftEvictUntilWithinLimit(evictionSpinsBackoff)
				if items > 0 || freedBytes > 0 {
					w.counters.evictedItems.Add(items)
					w.counters.evictedBytes.Add(freedBytes)
				}
			}
		}
	}
}
