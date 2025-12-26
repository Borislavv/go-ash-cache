package model

import (
	"github.com/Borislavv/go-ash-cache/internal/config"
	"sync/atomic"
	"time"
)

type AshItemCache interface {
	AshItem
	Key() *Key
	Update() error
	PayloadBytes() []byte
	SetPayload([]byte)
	IsExpired(cfg *config.Cache) bool
	UpdatedAt() int64
	TouchedAt() int64
	Weight() int64
}

type AshItem interface {
	SetTTL(ttl time.Duration)
}

type Entry struct {
	key               *Key                    // 64 bit xxh + hi + lo for manage collisions
	touchedAt         int64                   // atomic: unix nano (used in LRU algo.)
	updatedAt         int64                   // atomic: unix nano (used for refresh entry)
	ttl               int64                   // atomic: unix nano (used for refresh/remove entry)
	isQueuedOnRefresh int64                   // atomic: int as bool; whether an item is queued on update
	payload           *atomic.Pointer[[]byte] // atomic: payload ([]byte)
	callback          func(entry AshItem) ([]byte, error)
}

func NewEmptyEntry(key *Key, cfgTTL int64, callback func(entry AshItem) ([]byte, error)) *Entry {
	return &Entry{
		key:      key,
		ttl:      cfgTTL,
		callback: callback,
		payload:  &atomic.Pointer[[]byte]{},
	}
}

func (e *Entry) Update() error {
	payload, err := e.callback(e)
	if err != nil {
		return err
	}
	e.payload.Store(&payload)
	return nil
}
