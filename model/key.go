package model

type Key struct {
	v  uint64
	hi uint64
	lo uint64
}

func NewKey(v, hi, lo uint64) *Key {
	return &Key{
		v:  v,
		hi: hi,
		lo: lo,
	}
}

func (k *Key) Value() uint64 {
	return k.v
}

func (k *Key) IsTheSame(key *Key) (same bool) {
	return k.v == key.v && k.hi == key.hi && k.lo == key.lo
}
