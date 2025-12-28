package ashcache

import (
	"github.com/Borislavv/go-ash-cache/config"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"time"
)

type Item interface {
	SetTTL(ttl time.Duration)
	SetTTLMode(mode TTLMode)
}

type CacheItem interface {
	Item
	Key() *model.Key
	Update() error
	PayloadBytes() []byte
	SetPayload([]byte)
	IsExpired(cfg *config.Cache) bool
	UpdatedAt() int64
	TouchedAt() int64
	Weight() int64
}
