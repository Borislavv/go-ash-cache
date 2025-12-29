package model

import (
	"github.com/Borislavv/go-ash-cache/model"
)

func (e *Entry) SetMapKeyForTests(key uint64) *Entry {
	e.key = model.NewKey(key, 0, 0)
	return e
}
