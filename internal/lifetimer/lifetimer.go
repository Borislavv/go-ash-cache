package lifetimer

import (
	"context"
	"github.com/Borislavv/go-ash-cache/internal/cache"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"github.com/Borislavv/go-ash-cache/internal/config"
	"github.com/Borislavv/go-ash-cache/internal/shared/rate"
	"log/slog"
	"runtime"
	"sync"
)

const removeModeJitterRate = 100_000

type Lifetimer interface {
	LifetimerMetrics() (affected, errors, scans, hits, misses int64)
	Close() error
}

type LifetimeWorker struct {
	ctx      context.Context
	cancel   context.CancelFunc
	cfg      *config.LifetimerCfg
	cache    *cache.Cache
	logger   *slog.Logger
	jitter   *rate.Jitter
	counters *lifetimerCounters
	invokeCh chan *model.Entry
}

func New(
	ctx context.Context,
	cfg *config.LifetimerCfg,
	logger *slog.Logger,
	cache *cache.Cache,
) Lifetimer {
	if !cfg.Enabled() {
		return &NoOpLifetimer{}
	}

	ctx, cancel := context.WithCancel(ctx)

	var jitter *rate.Jitter
	if cfg.OnTTL == config.TTLModeRefresh {
		jitter = rate.NewJitter(ctx, cfg.Rate)
	} else {
		jitter = rate.NewJitter(ctx, removeModeJitterRate)
	}

	var invokeCap = cfg.Rate
	if invokeCap <= 0 {
		invokeCap = 1
	}

	return (&LifetimeWorker{
		ctx:      ctx,
		cancel:   cancel,
		cfg:      cfg,
		cache:    cache,
		logger:   logger,
		jitter:   jitter,
		counters: newLifetimerCounters(),
		invokeCh: make(chan *model.Entry, invokeCap),
	}).run()
}

func (w *LifetimeWorker) LifetimerMetrics() (affected, errors, scans, hits, misses int64) {
	return w.counters.snapshot()
}

func (w *LifetimeWorker) Close() error {
	w.cancel()
	return nil
}

func (w *LifetimeWorker) run() *LifetimeWorker {
	w.logger.Info("refresher is running", "mode", w.cfg.OnTTL, "rate", w.cfg.Rate)

	go func() {
		defer w.logger.Info("refresher is stopped")
		var wg sync.WaitGroup
		for i := 0; i <= runtime.GOMAXPROCS(0); i++ {
			wg.Go(w.consumer)
		}
		wg.Go(w.provider)
		wg.Wait()
	}()

	return w
}

func (w *LifetimeWorker) provider() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.jitter.Chan():
			if w.cache.Len() > 0 {
				w.counters.scans.Add(1)
				entry, ok := w.cache.PeekExpiredTTL()
				if !ok {
					w.counters.scanMisses.Add(1)
					continue
				}
				w.counters.scanHits.Add(1)

				select {
				case <-w.ctx.Done():
					return
				case w.invokeCh <- entry:
				}
			}
		}
	}
}

func (w *LifetimeWorker) consumer() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case entry := <-w.invokeCh:
			if err := w.cache.OnTTL(entry); err == nil {
				w.counters.affected.Add(1)
			} else {
				w.counters.errors.Add(1)
			}
		}
	}
}
