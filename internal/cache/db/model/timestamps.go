package model

import (
	"github.com/Borislavv/go-ash-cache/internal/shared/cachedtime"
	"github.com/Borislavv/go-ash-cache/model"
	"sync/atomic"
	"time"
)

var trueI32 = int32(1)

func (e *Entry) SetTTL(ttl time.Duration) {
	atomic.StoreInt64(&e.ttl, ttl.Nanoseconds())
}

func (e *Entry) SetTTLMode(mode model.TTLMode) {
	atomic.StoreInt32(&e.isRemoveOnTTL, int32(mode))
}

func (e *Entry) IsRemoveByTTL() bool {
	return atomic.LoadInt32(&e.isRemoveOnTTL) == trueI32
}

func (e *Entry) UpdatedAt() int64 {
	return atomic.LoadInt64(&e.updatedAt)
}

func (e *Entry) RenewTouchedAt() {
	atomic.StoreInt64(&e.touchedAt, cachedtime.UnixNano())
}

func (e *Entry) TouchedAt() int64 {
	return atomic.LoadInt64(&e.touchedAt)
}

func (e *Entry) RenewUpdatedAt() {
	atomic.StoreInt64(&e.updatedAt, cachedtime.UnixNano())
}

func (e *Entry) UntouchRefreshedAt() {
	atomic.StoreInt64(&e.updatedAt, cachedtime.UnixNano()-e.ttl)
}
