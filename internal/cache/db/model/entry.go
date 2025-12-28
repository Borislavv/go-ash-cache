package model

import (
	"github.com/Borislavv/go-ash-cache"
	"sync/atomic"
)

type Entry struct {
	key               *Key                    // 64 bit xxh + hi + lo for manage collisions
	ttl               int64                   // atomic: unix nano (used for refresh/remove entry)
	isQueuedOnRefresh int32                   // atomic: int as bool; whether an item is queued on update
	isRemoveOnTTL     int32                   // atomic: int as bool; whether an item should be removed on TTL exceeded
	payload           *atomic.Pointer[[]byte] // atomic: payload ([]byte)
	callback          func(entry ashcache.Item) ([]byte, error)
	touchedAt         int64 // atomic: unix nano (used in LRU algo.)
	updatedAt         int64 // atomic: unix nano (used for refresh entry)
}

func NewEmptyEntry(key *Key, cfgTTL int64, callback func(entry ashcache.Item) ([]byte, error)) *Entry {
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
