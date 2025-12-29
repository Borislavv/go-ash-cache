package model

import (
	"github.com/Borislavv/go-ash-cache/model"
	"sync/atomic"
)

type TTLCallback func(entry model.Item) ([]byte, error)

type Entry struct {
	key               *model.Key              // 64 bit xxh + hi + lo for manage collisions
	ttl               int64                   // atomic: unix nano (used for refresh/remove entry)
	isQueuedOnRefresh int32                   // atomic: int as bool; whether an item is queued on update
	isRemoveOnTTL     int32                   // atomic: int as bool; whether an item should be removed on TTL exceeded
	payload           *atomic.Pointer[[]byte] // atomic: payload ([]byte)
	callback          TTLCallback
	touchedAt         int64 // atomic: unix nano (used in LRU algo.)
	updatedAt         int64 // atomic: unix nano (used for refresh entry)
}

func NewEntry(key *model.Key, ttl int64, isRemoveOnTTL bool) *Entry {
	e := &Entry{
		key:     key,
		ttl:     ttl,
		payload: &atomic.Pointer[[]byte]{},
	}
	if isRemoveOnTTL {
		e.isRemoveOnTTL = 1
	}
	return e
}

func (e *Entry) OnTTL() ([]byte, error) {
	return e.callback(e)
}

func (e *Entry) Update() error {
	payload, err := e.callback(e)
	if err != nil {
		return err
	}
	e.payload.Store(&payload)
	return nil
}

// SetCallback is NOT CONCURRENT SAFE!
func (e *Entry) SetCallback(callback TTLCallback) {
	e.callback = callback
}
