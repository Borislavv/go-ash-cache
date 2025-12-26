package bloom

import (
	"github.com/Borislavv/go-ash-cache/internal/db"
	"github.com/Borislavv/go-ash-cache/internal/db/config"
	"github.com/Borislavv/go-ash-cache/internal/db/model"
	"math/rand"
	"testing"
	"time"
)

var cfg = &config.AdmissionControlCfg{
	Capacity:            1_000_000,
	Shards:              db.NumOfShards,
	MinTableLenPerShard: 8192,
	DoorBitsPerCounter:  16,
	SampleMultiplier:    10,
}

func BenchmarkTinyLFUIncrement(b *testing.B) {
	tlfu := newShardedAdmitter(cfg)

	keys := make([]uint64, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = rand.Uint64()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tlfu.Record(keys[i])
	}
}

func BenchmarkTinyLFUAdmit(b *testing.B) {
	tlfu := newShardedAdmitter(cfg)

	// simulate some initial frequencies
	for i := 0; i < 100000; i++ {
		tlfu.Record(uint64(i))
	}
	time.Sleep(time.Second) // wait for run()

	newEntry := (&model.Entry{}).SetMapKeyForTests(rand.Uint64())
	oldEntry := (&model.Entry{}).SetMapKeyForTests(rand.Uint64())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tlfu.Allow(newEntry.Key().Value(), oldEntry.Key().Value())
	}
}
