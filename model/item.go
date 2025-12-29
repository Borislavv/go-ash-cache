package model

import (
	"github.com/Borislavv/go-ash-cache/config"
	"time"
)

type TTLMode int32

const (
	TTLModeRefresh TTLMode = iota
	TTLModeRemove
)

type Item interface {
	Key() *Key
	SetTTL(ttl time.Duration)
	SetTTLMode(mode TTLMode)
}

type CacheItem interface {
	Item
	Update() error
	PayloadBytes() []byte
	SetPayload([]byte)
	IsExpired(cfg *config.Cache) bool
	UpdatedAt() int64
	TouchedAt() int64
	Weight() int64
}
