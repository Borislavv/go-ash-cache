package model

import (
	"github.com/zeebo/xxh3"
	"sync"
	"unsafe"
)

type Key struct {
	v  uint64
	hi uint64
	lo uint64
}

func NewKey(key string) *Key {
	return buildKey(unsafe.Slice(unsafe.StringData(key), len(key)))
}

func (k *Key) Value() uint64 {
	return k.v
}

func (k *Key) IsTheSame(key *Key) (same bool) {
	return k.v == key.v && k.hi == key.hi && k.lo == key.lo
}

var hasherPool = sync.Pool{New: func() any { return xxh3.New() }}

func (e *Entry) Key() *Key {
	if e == nil {
		return nil
	}
	return e.key
}

func buildKey(key []byte) *Key {
	// acquire reusable hasher
	hasher := hasherPool.Get().(*xxh3.Hasher)
	hasher.Reset()

	// calculate key hash
	_, _ = hasher.Write(key)

	u128 := hasher.Sum128()

	// calculate map key
	k := &Key{
		v:  hasher.Sum64(),
		hi: u128.Hi,
		lo: u128.Lo,
	}

	// release hasher after use
	hasherPool.Put(hasher)

	return k
}

func (e *Entry) setUpNewKey(data []byte) {
	if e.key != nil {
		// already exists
		return
	}
	e.key = buildKey(data)
}
