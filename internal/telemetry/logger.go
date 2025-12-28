package telemetry

import (
	"context"
	"github.com/Borislavv/go-ash-cache/config"
	"log/slog"
	"time"

	"github.com/Borislavv/go-ash-cache/internal/cache"
	"github.com/Borislavv/go-ash-cache/internal/evictor"
	"github.com/Borislavv/go-ash-cache/internal/lifetimer"
	"github.com/Borislavv/go-ash-cache/internal/shared/bytes"
)

type Logger interface {
	Interval() time.Duration
	Close() error
}

type Logs struct {
	ctx       context.Context
	cancel    context.CancelFunc
	cfg       *config.Cache
	logger    *slog.Logger
	cache     cache.Cacher
	evictor   evictor.Evictor
	lifetimer lifetimer.Lifetimer
	interval  time.Duration
}

func New(
	ctx context.Context,
	cfg *config.Cache,
	logger *slog.Logger,
	cache cache.Cacher,
	evictor evictor.Evictor,
	lifetimer lifetimer.Lifetimer,
	interval time.Duration,
) *Logs {
	ctx, cancel := context.WithCancel(ctx)
	return (&Logs{
		ctx:       ctx,
		cancel:    cancel,
		cfg:       cfg,
		logger:    logger,
		cache:     cache,
		evictor:   evictor,
		lifetimer: lifetimer,
		interval:  interval,
	}).run()
}

func (l *Logs) Interval() time.Duration {
	return l.interval
}

func (l *Logs) Close() error {
	l.cancel()
	return nil
}

func (l *Logs) run() *Logs {
	if l.cfg != nil && l.cfg.DB.IsTelemetryLogsEnabled {
		go l.loop()
	}
	return l
}

func (l *Logs) loop() {
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()

	var softLimit = "INF"
	if l.cfg.Eviction.Enabled() {
		softLimit = bytes.FmtMem(uint64(l.cfg.Eviction.SoftMemoryLimitBytes))
	}
	hardLimit := bytes.FmtMem(uint64(l.cfg.DB.SizeBytes))

	s := newSampler(l.cache, l.evictor, l.lifetimer)
	prev := s.snapshot()

	for {
		select {
		case <-l.ctx.Done():
			return

		case <-ticker.C:
			cur := s.snapshot()
			d := deltaSnapshot(prev, cur)
			prev = cur

			common := []any{"interval", l.interval.String()}
			memBytes := uint64(l.cache.Mem())
			items := l.cache.Len()

			if l.cfg.Lifetime.Enabled() {
				l.logger.Info("lifetime_manager",
					append(common,
						"affected", int64(d.lifetimeAffected),
						"errors", int64(d.lifetimeErrors),
						"scans", int64(d.lifetimeScans),
						"hits", int64(d.lifetimeHits),
						"misses", int64(d.lifetimeMisses),
					)...,
				)
			}

			if l.cfg.AdmissionControl.Enabled() {
				l.logger.Info("admission_controller",
					append(common,
						"allowed", int64(d.admissionAllowed),
						"not_allowed", int64(d.admissionNotAllowed),
					)...,
				)
			}

			if l.cfg.Eviction.Enabled() {
				l.logger.Info("soft_evictor",
					append(common,
						"scans", int64(d.softScans),
						"hits", int64(d.softHits),
						"freed_items", int64(d.softEvictedItems),
						"freed_bytes", bytes.FmtMem(d.softEvictedBytes),
					)...,
				)
			}

			if d.hardEvictedItems > 0 || d.hardEvictedBytes > 0 {
				l.logger.Info("hard_evictor",
					append(common,
						"freed_items", int64(d.hardEvictedItems),
						"freed_bytes", bytes.FmtMem(d.hardEvictedBytes),
					)...,
				)
			}

			l.logger.Info("storage",
				append(common,
					"size", bytes.FmtMem(memBytes),
					"entries", items,
					"soft_limit", softLimit,
					"hard_limit", hardLimit,
				)...,
			)
		}
	}
}
