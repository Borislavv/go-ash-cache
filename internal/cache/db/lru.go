package db

import (
	"container/list"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"sync/atomic"
)

type LRUMode int

const (
	Listing LRUMode = iota
	Sampling
)

func (sh *Shard) enableLRU() {
	sh.Lock()
	if sh.lru == nil {
		sh.lru = list.New()
		if sh.lidx == nil {
			sh.lidx = make(map[uint64]*list.Element, len(sh.items))
		}
		for k := range sh.items {
			sh.lidx[k] = sh.lru.PushFront(k)
		}
	}
	sh.lruOn = true
	sh.Unlock()
}

func (sh *Shard) disableLRU() {
	sh.Lock()
	sh.lruOn = false
	sh.lru = nil
	sh.lidx = nil
	sh.Unlock()
}

// lruOnInsertUnlocked - is unsafe without shard.Lock due to it mutates the list.
func (sh *Shard) lruOnInsertUnlocked(key uint64) {
	if !sh.lruOn || sh.lru == nil {
		return
	}
	if el := sh.lidx[key]; el != nil {
		sh.lru.MoveToFront(el)
		return
	}
	sh.lidx[key] = sh.lru.PushFront(key)
}

// lruOnAccessUnlocked - is unsafe without shard.Lock due to it mutates the list otherwise use touchLRU.
func (sh *Shard) lruOnAccessUnlocked(key uint64) {
	if !sh.lruOn || sh.lru == nil {
		return
	}
	if el := sh.lidx[key]; el != nil {
		sh.lru.MoveToFront(el)
	}
}

// lruOnDeleteUnlocked - is unsafe without shard.Lock due to it mutates the list.
func (sh *Shard) lruOnDeleteUnlocked(key uint64) {
	if !sh.lruOn || sh.lru == nil {
		return
	}
	if el := sh.lidx[key]; el != nil {
		sh.lru.Remove(el)
		delete(sh.lidx, key)
	}
}

// touchLRU - threadsafe.
func (sh *Shard) touchLRU(key uint64) {
	if !sh.lruOn || sh.lru == nil {
		return
	}
	if sh.TryLock() {
		if el := sh.lidx[key]; el != nil {
			sh.lru.MoveToFront(el)
		}
		sh.Unlock()
	}
}

// tail ops for eviction/refresh
func (sh *Shard) lruPeekTail() (key uint64, val *model.Entry, ok bool) {
	if !sh.lruOn || sh.lru == nil {
		return 0, nil, false
	}
	sh.RLock()
	defer sh.RUnlock()
	el := sh.lru.Back()
	if el == nil {
		return 0, nil, false
	}
	k := el.Value.(uint64)
	v, ok := sh.items[k]
	if !ok {
		return 0, nil, false
	}
	return k, v, true
}

func (sh *Shard) lruPopTail() (key uint64, val *model.Entry, ok bool) {
	if !sh.lruOn || sh.lru == nil {
		return 0, nil, false
	}
	sh.Lock()
	defer sh.Unlock()
	el := sh.lru.Back()
	if el == nil {
		return 0, nil, false
	}
	k := el.Value.(uint64)
	v, ok := sh.items[k]
	if !ok {
		sh.lru.Remove(el)
		delete(sh.lidx, k)
		return 0, nil, false
	}
	delete(sh.items, k)
	atomic.AddInt64(&sh.len, -1)
	atomic.AddInt64(&sh.mem, -v.Weight())
	sh.lru.Remove(el)
	delete(sh.lidx, k)
	return k, v, true
}

func (sh *Shard) lruPeekHead() (key uint64, val *model.Entry, ok bool) {
	if !sh.lruOn || sh.lru == nil {
		return 0, nil, false
	}
	sh.RLock()
	defer sh.RUnlock()
	el := sh.lru.Front()
	if el == nil {
		return 0, nil, false
	}
	k := el.Value.(uint64)
	v, ok := sh.items[k]
	if !ok {
		return 0, nil, false
	}
	return k, v, true
}

func (sh *Shard) lruPeekHeadK(k int, chooseFn func(*model.Entry) bool) (v *model.Entry, ok bool) {
	if !sh.lruOn || sh.lru == nil || k <= 0 {
		return nil, false
	}
	sh.RLock()
	defer sh.RUnlock()

	e := sh.lru.Front()
	for i := 0; i < k && e != nil; i, e = i+1, e.Next() {
		key := e.Value.(uint64)
		vv, ok2 := sh.items[key]
		if !ok2 {
			continue
		}
		if chooseFn(vv) {
			return vv, true
		}
	}
	return nil, false
}

func (sh *Shard) lruPeekTailK(k int, chooseFn func(*model.Entry) bool) (v *model.Entry, ok bool) {
	if !sh.lruOn || sh.lru == nil || k <= 0 {
		return nil, false
	}
	sh.RLock()
	defer sh.RUnlock()

	e := sh.lru.Back()
	for i := 0; i < k && e != nil; i, e = i+1, e.Prev() {
		key := e.Value.(uint64)
		vv, ok2 := sh.items[key]
		if !ok2 {
			continue
		}
		if chooseFn(vv) {
			return vv, true
		}
	}
	return nil, false
}
