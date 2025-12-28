package ashcache

import (
	"context"
	"github.com/Borislavv/go-ash-cache/config"
	"github.com/Borislavv/go-ash-cache/internal/cache"
	"github.com/Borislavv/go-ash-cache/internal/evictor"
	"github.com/Borislavv/go-ash-cache/internal/lifetimer"
	"github.com/Borislavv/go-ash-cache/internal/shared/cachedtime"
	"github.com/Borislavv/go-ash-cache/internal/telemetry"
	"io"
	"log/slog"
)

type AshCache interface {
	cache.Cacher
	evictor.Evictor
	lifetimer.Lifetimer
	telemetry.Logger
	io.Closer
}

type Cache struct {
	cache.Cacher
	evictor.Evictor
	lifetimer.Lifetimer
	telemetry.Logger
	cls context.CancelFunc
}

func New(ctx context.Context, cfg *config.Cache, logger *slog.Logger) *Cache {
	ctx, cancel := context.WithCancel(ctx)
	cachedtime.RunIfEnabled(ctx, cfg)
	cacher := cache.New(ctx, cfg, logger)
	eviction := evictor.New(ctx, cfg.Eviction, logger, cacher)
	lifetime := lifetimer.New(ctx, cfg.Lifetime, logger, cacher)
	telemeter := telemetry.New(ctx, cfg, logger, cacher, eviction, lifetime, cfg.DB.TelemetryLogsInterval)
	return &Cache{cls: cancel, Cacher: cacher, Evictor: eviction, Lifetimer: lifetime, Logger: telemeter}
}

func (c *Cache) Close() error {
	c.cls()
	return nil
}
