package cache

import (
	"context"
	"github.com/Borislavv/go-ash-cache"
	"github.com/Borislavv/go-ash-cache/config"
	"github.com/Borislavv/go-ash-cache/internal/cache/db"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/bloom"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"log/slog"
	"runtime"
)

const shardsSample, keysSample, spinsBackoff = 2, 8, 32

type Cacher interface {
	Get(key string, callback func(item ashcache.Item) ([]byte, error)) (data []byte, err error)
	CacheMetrics() (admissionAllowed, admissionNotAllowed, hardEvictedItems, hardEvictedBytes int64)
	Around(ctx context.Context, fn func(item ashcache.CacheItem) bool, rw bool)
	Del(key string) (ok bool)
	Clear()
	Len() int64
	Mem() int64
}

// Cache respects given ctx.
type Cache struct {
	admitter bloom.AdmissionControl
	cfg      *config.Cache
	db       *db.Map
	logger   *slog.Logger
	counters *counters
}

func New(ctx context.Context, cfg *config.Cache, logger *slog.Logger) *Cache {
	return &Cache{
		cfg:      cfg,
		logger:   logger,
		counters: newCounters(),
		db:       db.NewMap(ctx, cfg),
		admitter: bloom.NewAdmissionControl(cfg.AdmissionControl),
	}
}

func (c *Cache) Get(key string, callback func(item ashcache.Item) ([]byte, error)) (data []byte, err error) {
	k := model.NewKey(key)
	if entry, ok := c.get(k.Value()); ok {
		if entry.Key().IsTheSame(k) {
			return entry.PayloadBytes(), nil
		}
		// hash collision
	}

	entry := c.makeEntry(k, callback)
	resp, respErr := callback(entry)
	if respErr != nil {
		return nil, respErr
	}
	entry.SetPayload(resp)
	c.set(entry)

	return resp, nil
}

func (c *Cache) Del(key string) bool {
	k := model.NewKey(key)

	if entry, ok := c.get(k.Value()); ok {
		if entry.Key().IsTheSame(k) {
			_, ok = c.remove(entry)
			return ok
		}
		// hash collision
	}

	return true
}

func (c *Cache) OnTTL(entry *model.Entry) error {
	if c.cfg.Lifetime.IsRemoveOnTTL {
		c.remove(entry)
		return nil
	} else {
		return entry.Update()
	}
}

func (c *Cache) Len() int64 { return c.db.Len() }
func (c *Cache) Mem() int64 { return c.db.Mem() }
func (c *Cache) Clear()     { c.db.Clear() }

func (c *Cache) CacheMetrics() (admissionAllowed, admissionNotAllowed, hardEvictedItems, hardEvictedBytes int64) {
	return c.counters.snapshot()
}

func (c *Cache) Around(ctx context.Context, fn func(item ashcache.CacheItem) bool, rw bool) {
	c.db.WalkShardsConcurrent(ctx, runtime.GOMAXPROCS(0), func(key uint64, shard *db.Shard) {
		shard.Walk(ctx, fn, rw)
	})
}

func (c *Cache) WalkShards(ctx context.Context, fn func(key uint64, shard *db.Shard)) {
	c.db.WalkShardsConcurrent(ctx, runtime.GOMAXPROCS(0), fn)
}

func (c *Cache) SoftEvictUntilWithinLimit(backoff int64) (freed, evicted int64) {
	if c.cfg.Eviction.Enabled() {
		freed, evicted = c.db.EvictUntilWithinLimit(c.cfg.Eviction.SoftMemoryLimitBytes, backoff)
	}
	return
}

func (c *Cache) SoftMemoryLimitOvercome() bool {
	return c.cfg.Eviction.Enabled() && c.db.Len() > 0 && c.db.Mem() > c.cfg.Eviction.SoftMemoryLimitBytes
}

func (c *Cache) PeekExpiredTTL() (*model.Entry, bool) {
	return c.db.PeekExpiredTTL()
}

/**
 * Private API.
 */

func (c *Cache) get(key uint64) (*model.Entry, bool) {
	if ptr, found := c.db.Get(key); found {
		return c.touch(ptr), true
	}
	return nil, false
}

func (c *Cache) set(new *model.Entry) (persisted bool) {
	key := new.Key().Value()
	c.admitter.Record(key)

	if old, found := c.db.Get(key); found {
		if old.IsTheSamePayload(new) {
			c.touch(old)
		} else {
			c.update(old, new)
		}
		return true
	}

	if c.isAdmissionControlAllowed() {
		_, victim, found := c.db.PickVictim(shardsSample, keysSample)
		if !found || !c.admitter.Allow(key, victim.Key().Value()) {
			c.counters.admissionNotAllowed.Add(1)
			return false
		} else {
			c.counters.admissionAllowed.Add(1)
		}
	}

	if c.hardMemoryLimitOvercome() {
		freedBytes, items := c.hardEvictUntilWithinLimit()
		if freedBytes > 0 || items > 0 {
			c.counters.evictedHardLimitItems.Add(items)
			c.counters.evictedHardLimitBytes.Add(freedBytes)
		}
	}

	c.db.Set(key, new)

	return true
}

func (c *Cache) touch(existing *model.Entry) *model.Entry {
	existing.RenewTouchedAt()
	// move to front in LRU list
	c.db.Touch(existing.Key().Value())
	// check the entry exists and expired, if so then push it to the per-shard refresh queue
	if existing.IsExpired(c.cfg) && existing.EnqueueExpired() {
		if !c.db.EnqueueExpired(existing.Key().Value()) {
			existing.DequeueExpired()
		}
	}
	return existing
}

func (c *Cache) update(existing, in *model.Entry) {
	c.db.AddMem(existing.Key().Value(), existing.SwapPayloads(in))
	existing.RenewTouchedAt()
	existing.RenewUpdatedAt()
	existing.DequeueExpired()
	c.db.Touch(existing.Key().Value())
}

func (c *Cache) makeEntry(key *model.Key, callback func(entry ashcache.Item) ([]byte, error)) *model.Entry {
	return model.NewEmptyEntry(key, c.cfgTTLNanoseconds(), callback)
}

func (c *Cache) cfgTTLNanoseconds() int64 {
	if c.cfg.Lifetime.Enabled() {
		return c.cfg.Lifetime.TTL.Nanoseconds()
	}
	return 0
}

func (c *Cache) remove(entry *model.Entry) (int64, bool) { return c.db.Remove(entry.Key().Value()) }

func (c *Cache) hardEvictUntilWithinLimit() (freed, evicted int64) {
	if c.cfg.Eviction.Enabled() {
		freed, evicted = c.db.EvictUntilWithinLimit(c.cfg.DB.SizeBytes, spinsBackoff)
	}
	return
}

func (c *Cache) hardMemoryLimitOvercome() bool {
	return c.cfg.Eviction.Enabled() && c.db.Len() > 0 && c.db.Mem()-c.cfg.DB.SizeBytes > 0
}

func (c *Cache) isAdmissionControlAllowed() bool {
	return c.cfg.AdmissionControl.Enabled() && c.db.Len() > 0 && c.db.Mem() > 0
}
