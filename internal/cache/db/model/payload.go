package model

import (
	"github.com/Borislavv/go-ash-cache/internal/shared/bytes"
	"github.com/Borislavv/go-ash-cache/internal/shared/cachedtime"
	"sync/atomic"
	"unsafe"
)

func (e *Entry) Weight() int64 { return int64(unsafe.Sizeof(*e)) + int64(cap(e.PayloadBytes())) }

func (e *Entry) PayloadBytes() []byte {
	if ptr := e.payload.Load(); ptr != nil {
		return *ptr
	}
	return nil
}

func (e *Entry) IsTheSamePayload(another *Entry) bool {
	a := e.PayloadBytes()
	b := another.PayloadBytes()
	if a == nil {
		return b == nil
	}
	if b != nil {
		return bytes.IsBytesAreEquals(a, b)
	}
	return false
}

func (e *Entry) SwapPayloads(another *Entry) (weightDiff int64) {
	newWeight := another.Weight()
	oldWeight := e.Weight()
	e.payload.Swap(another.payload.Load())
	return newWeight - oldWeight
}

func (e *Entry) SetPayload(p []byte) {
	now := cachedtime.Now().UnixNano()
	atomic.StoreInt64(&e.touchedAt, now)
	atomic.StoreInt64(&e.updatedAt, now)
	atomic.StoreInt32(&e.isQueuedOnRefresh, 0)
	e.setUpNewKey(p)
	e.payload.Store(&p)
}
