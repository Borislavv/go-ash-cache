package tests

import (
	"context"
	"github.com/Borislavv/go-ash-cache"
	"github.com/Borislavv/go-ash-cache/config"
	"github.com/Borislavv/go-ash-cache/model"
	"log/slog"
	"math/rand"
	"sync"
	"testing"
	"time"
)

var (
	benchCache     *ashcache.Cache
	benchCacheOnce sync.Once
	benchKeys      []string
)

func initBenchCache() {
	ctx := context.Background()
	logger := slog.Default()

	cfg := &config.Cache{
		DB: config.DBCfg{
			SizeBytes:        100 * 1024 * 1024, // 100MB
			CacheTimeEnabled: true,
		},
		Eviction: &config.EvictionCfg{
			LRUMode:              config.LRUModeListing,
			SoftLimitCoefficient: 0.8,
			CallsPerSec:          10,
			BackoffSpinsPerCall:  2048,
		},
		AdmissionControl: &config.AdmissionControlCfg{
			Capacity:            10000,
			Shards:              16,
			MinTableLenPerShard: 64,
			SampleMultiplier:    10,
			DoorBitsPerCounter:  2,
		},
	}
	cfg.AdjustConfig()

	benchCache = ashcache.New(ctx, cfg, logger)

	// Pre-populate with test data
	benchKeys = make([]string, 1000)
	testData := make([]byte, 1024) // 1KB payload
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	for i := 0; i < 1000; i++ {
		key := string(rune('a'+(i%26))) + string(rune('0'+(i/26)))
		benchKeys[i] = key
		_, _ = benchCache.Get(key, func(item model.Item) ([]byte, error) {
			return testData, nil
		})
	}
}

func getBenchCache() *ashcache.Cache {
	benchCacheOnce.Do(initBenchCache)
	return benchCache
}

// BenchmarkGetHit measures Get() performance on cache hits
func BenchmarkGetHit(b *testing.B) {
	cache := getBenchCache()
	key := benchKeys[0]
	testData := make([]byte, 1024)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		data, err := cache.Get(key, func(item model.Item) ([]byte, error) {
			return testData, nil
		})
		if err != nil {
			b.Fatal(err)
		}
		if len(data) == 0 {
			b.Fatal("empty data")
		}
	}
}

// BenchmarkGetMiss measures Get() performance on cache misses
func BenchmarkGetMiss(b *testing.B) {
	cache := getBenchCache()
	testData := make([]byte, 1024)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := string(rune('z')) + string(rune('0'+(i%10)))
		data, err := cache.Get(key, func(item model.Item) ([]byte, error) {
			return testData, nil
		})
		if err != nil {
			b.Fatal(err)
		}
		if len(data) == 0 {
			b.Fatal("empty data")
		}
	}
}

// BenchmarkSet measures Set() performance (via Get on miss)
func BenchmarkSet(b *testing.B) {
	cache := getBenchCache()
	testData := make([]byte, 1024)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := string(rune('z')) + string(rune('1'+(i%10)))
		_, err := cache.Get(key, func(item model.Item) ([]byte, error) {
			return testData, nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetHitParallel measures concurrent Get() performance on hits
func BenchmarkGetHitParallel(b *testing.B) {
	cache := getBenchCache()
	key := benchKeys[0]
	testData := make([]byte, 1024)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			data, err := cache.Get(key, func(item model.Item) ([]byte, error) {
				return testData, nil
			})
			if err != nil {
				b.Fatal(err)
			}
			if len(data) == 0 {
				b.Fatal("empty data")
			}
		}
	})
}

// BenchmarkGetMissParallel measures concurrent Get() performance on misses
func BenchmarkGetMissParallel(b *testing.B) {
	cache := getBenchCache()
	testData := make([]byte, 1024)
	counter := int64(0)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := int(counter) % 10000
			counter++
			key := string(rune('z')) + string(rune('a'+(idx%26))) + string(rune('0'+(idx/26)))
			data, err := cache.Get(key, func(item model.Item) ([]byte, error) {
				return testData, nil
			})
			if err != nil {
				b.Fatal(err)
			}
			if len(data) == 0 {
				b.Fatal("empty data")
			}
		}
	})
}

// BenchmarkGetMixed measures Get() with mixed hit/miss ratio (80% hits)
func BenchmarkGetMixed(b *testing.B) {
	cache := getBenchCache()
	testData := make([]byte, 1024)
	rng := rand.New(rand.NewSource(42))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var key string
		if rng.Float32() < 0.8 {
			// 80% hits
			key = benchKeys[rng.Intn(len(benchKeys))]
		} else {
			// 20% misses
			key = string(rune('z')) + string(rune('0'+(i%10)))
		}

		data, err := cache.Get(key, func(item model.Item) ([]byte, error) {
			return testData, nil
		})
		if err != nil {
			b.Fatal(err)
		}
		if len(data) == 0 {
			b.Fatal("empty data")
		}
	}
}

// BenchmarkGetMixedParallel measures concurrent Get() with mixed hit/miss ratio
func BenchmarkGetMixedParallel(b *testing.B) {
	cache := getBenchCache()
	testData := make([]byte, 1024)
	counter := int64(0)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			var key string
			if rng.Float32() < 0.8 {
				// 80% hits
				key = benchKeys[rng.Intn(len(benchKeys))]
			} else {
				// 20% misses
				idx := int(counter) % 10000
				counter++
				key = string(rune('z')) + string(rune('a'+(idx%26))) + string(rune('0'+(idx/26)))
			}

			data, err := cache.Get(key, func(item model.Item) ([]byte, error) {
				return testData, nil
			})
			if err != nil {
				b.Fatal(err)
			}
			if len(data) == 0 {
				b.Fatal("empty data")
			}
		}
	})
}

// BenchmarkDel measures Delete() performance
func BenchmarkDel(b *testing.B) {
	cache := getBenchCache()
	testData := make([]byte, 1024)

	// Pre-populate keys to delete
	keys := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		key := string(rune('d')) + string(rune('0'+(i%10)))
		keys[i] = key
		_, _ = cache.Get(key, func(item model.Item) ([]byte, error) {
			return testData, nil
		})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Del(keys[i])
	}
}

// BenchmarkConcurrentThroughput measures overall throughput with concurrent operations
func BenchmarkConcurrentThroughput(b *testing.B) {
	cache := getBenchCache()
	testData := make([]byte, 1024)

	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	numGoroutines := 10
	opsPerGoroutine := b.N / numGoroutines

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				key := benchKeys[(goroutineID*opsPerGoroutine+i)%len(benchKeys)]
				data, err := cache.Get(key, func(item model.Item) ([]byte, error) {
					return testData, nil
				})
				if err != nil {
					b.Fatal(err)
				}
				if len(data) == 0 {
					b.Fatal("empty data")
				}
			}
		}(g)
	}

	wg.Wait()
}
