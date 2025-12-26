package model

func (e *Entry) SetMapKeyForTests(key uint64) *Entry {
	e.key = &Key{v: key}
	return e
}
