package cache

import (
	"context"
	"github.com/Borislavv/go-ash-cache/internal/cache/db"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/bloom"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"github.com/Borislavv/go-ash-cache/internal/config"
	"log/slog"
	"runtime"
)

const shardsSample, keysSample, spinsBackoff = 2, 8, 32

type Cacher interface {
	// Cache API
	Get(key uint64) (*model.Entry, bool)
	Set(new *model.Entry) (persisted bool)
	Remove(entry *model.Entry) (int64, bool)
	Clear()
	Len() int64
	Mem() int64
	// Evictor API
	SoftMemoryLimitOvercome() bool
	SoftEvictUntilWithinLimit(backoff int64) (freed, evicted int64)
	// Lifetimer API
	PeekExpiredTTL() (*model.Entry, bool)
	// Public Additionl Access API
	MakeEntry(key *model.Key, callback func(entry model.AshItem) ([]byte, error)) *model.Entry
	WalkShards(ctx context.Context, fn func(key uint64, shard *db.Shard))
	Metrics() (admissionAllowed, admissionNotAllowed, hardEvictedItems, hardEvictedBytes int64)
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

func (c *Cache) Get(key uint64) (*model.Entry, bool) {
	if ptr, found := c.db.Get(key); found {
		return c.touch(ptr), true
	}
	return nil, false
}

func (c *Cache) Set(new *model.Entry) (persisted bool) {
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
	if existing.IsExpired(c.cfg) && existing.QueueExpired() {
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

func (c *Cache) OnTTL(entry *model.Entry) error {
	if c.cfg.Lifetime.IsRemoveOnTTL {
		c.Remove(entry)
		return nil
	} else {
		return entry.Update()
	}
}

func (c *Cache) Len() int64                              { return c.db.Len() }
func (c *Cache) Mem() int64                              { return c.db.Mem() }
func (c *Cache) Clear()                                  { c.db.Clear() }
func (c *Cache) Remove(entry *model.Entry) (int64, bool) { return c.db.Remove(entry.Key().Value()) }

func (c *Cache) Metrics() (admissionAllowed, admissionNotAllowed, hardEvictedItems, hardEvictedBytes int64) {
	return c.counters.snapshot()
}

func (c *Cache) MakeEntry(key *model.Key, callback func(entry model.AshItem) ([]byte, error)) *model.Entry {
	return model.NewEmptyEntry(key, c.cfg.Lifetime.TTL.Nanoseconds(), callback)
}

func (c *Cache) Around(ctx context.Context, fn func(item model.AshItem) bool, rw bool) {
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
	return c.cfg.Eviction.Enabled() && c.db.Len() > 0 && c.db.Mem()-c.cfg.Eviction.SoftMemoryLimitBytes > 0
}

func (c *Cache) PeekExpiredTTL() (*model.Entry, bool) {
	return c.db.PeekExpiredTTL()
}

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
