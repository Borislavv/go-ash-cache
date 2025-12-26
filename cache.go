package ashcache

import (
	"context"
	"github.com/Borislavv/go-ash-cache/internal/cache"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"github.com/Borislavv/go-ash-cache/internal/config"
	"github.com/Borislavv/go-ash-cache/internal/evictor"
	"github.com/Borislavv/go-ash-cache/internal/lifetimer"
	"github.com/Borislavv/go-ash-cache/internal/shared/cachedtime"
	"github.com/Borislavv/go-ash-cache/internal/telemetry"
	"log/slog"
)

type AshCache interface {
	Get(key string, callback func(item model.AshItem) ([]byte, error)) (data []byte, err error)
	Del(key string) (ok bool)
}

type Cache struct {
	cacher    cache.Cacher
	eviction  evictor.Evictor
	lifetime  lifetimer.Lifetimer
	telemeter telemetry.Logger
}

func New(ctx context.Context, cfg *config.Cache, logger *slog.Logger) *Cache {
	cachedtime.CloseByCtx(ctx)
	cacher := cache.New(ctx, cfg, logger)
	eviction := evictor.New(ctx, cfg.Eviction, logger, cacher)
	lifetime := lifetimer.New(ctx, cfg.Lifetime, logger, cacher)
	telemeter := telemetry.New(ctx, cfg, logger, cacher, eviction, lifetime, cfg.DB.TelemetryLogsInterval)
	return &Cache{cacher: cacher, eviction: eviction, lifetime: lifetime, telemeter: telemeter}
}

func (c *Cache) Get(key string, callback func(item model.AshItem) ([]byte, error)) (data []byte, err error) {
	k := model.NewKey(key)

	if entry, ok := c.cacher.Get(k.Value()); ok {
		if entry.Key().IsTheSame(k) {
			return entry.PayloadBytes(), nil
		}
		// hash collision; rewrite, because it's really rare
	}

	entry := c.cacher.MakeEntry(k, callback)
	resp, respErr := callback(entry)
	if respErr != nil {
		return nil, respErr
	}
	entry.SetPayload(resp)
	c.cacher.Set(entry)

	return resp, nil
}
