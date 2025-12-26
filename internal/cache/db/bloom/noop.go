package bloom

type noopBloomFilter struct{}

func newNoOp() *noopBloomFilter {
	return &noopBloomFilter{}
}

func (f *noopBloomFilter) Record(h uint64)                     {}
func (f *noopBloomFilter) Allow(candidate, victim uint64) bool { return true }
func (f *noopBloomFilter) Estimate(h uint64) uint8             { return 0 }
func (f *noopBloomFilter) Reset()                              {}
