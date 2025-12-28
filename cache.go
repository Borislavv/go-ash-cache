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

type TTLMode int32

const (
	Refresh TTLMode = iota
	Remove
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
	context.CancelFunc
}

// New - make a new AshCache instance. Respects context, as well as each component does.
func New(ctx context.Context, cfg *config.Cache, logger *slog.Logger) *Cache {
	ctx, cancel := context.WithCancel(ctx)
	cachedtime.RunIfEnabled(ctx, cfg)
	cacher := cache.New(ctx, cfg, logger)
	eviction := evictor.New(ctx, cfg.Eviction, logger, cacher)
	lifetime := lifetimer.New(ctx, cfg.Lifetime, logger, cacher)
	telemeter := telemetry.New(ctx, cfg, logger, cacher, eviction, lifetime, cfg.DB.TelemetryLogsInterval)
	return &Cache{CancelFunc: cancel, Cacher: cacher, Evictor: eviction, Lifetimer: lifetime, Logger: telemeter}
}

// Close - force close before the main context is not done yet. Otherwise, it does not necessary.
func (c *Cache) Close() error {
	c.CancelFunc()
	return nil
}
