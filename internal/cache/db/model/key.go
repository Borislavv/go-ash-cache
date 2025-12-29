package model

import (
	"github.com/Borislavv/go-ash-cache/model"
	"github.com/zeebo/xxh3"
	"sync"
	"unsafe"
)

func NewKey(key string) *model.Key {
	return buildKey(unsafe.Slice(unsafe.StringData(key), len(key)))
}

var hasherPool = sync.Pool{New: func() any { return xxh3.New() }}

func (e *Entry) Key() *model.Key {
	if e == nil {
		return nil
	}
	return e.key
}

func buildKey(key []byte) *model.Key {
	// acquire reusable hasher
	hasher := hasherPool.Get().(*xxh3.Hasher)
	hasher.Reset()

	// calculate key hash
	_, _ = hasher.Write(key)

	u128 := hasher.Sum128()

	// calculate map key
	k := model.NewKey(hasher.Sum64(), u128.Hi, u128.Lo)

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
