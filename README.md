# AshCache

[![Go Version](https://img.shields.io/static/v1?label=Go&message=1.25%2B&logo=go&color=00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/codecov/c/github/Borislavv/go-ash-cache?label=coverage)](https://codecov.io/gh/Borislavv/go-ash-cache)
[![License](https://img.shields.io/badge/License-Apache--2.0-green.svg)](./LICENCE)
[![Go Reference](https://pkg.go.dev/badge/github.com/Borislavv/go-ash-cache.svg)](https://pkg.go.dev/github.com/Borislavv/go-ash-cache)

**AshCache** is a production-grade, high-performance in-memory cache library for Go. Built for applications that demand predictable latency, intelligent admission control, and efficient memory management at scale.

## Features

### 🚀 Performance & Scalability

- **Sharded Architecture**: 1024 independent shards minimize lock contention, enabling linear scalability across CPU cores
- **Zero-Allocation Hot Paths**: Critical operations (Get/Set) avoid heap allocations for maximum throughput
- **Lock-Free Operations**: Atomic counters and CAS-based algorithms reduce synchronization overhead

### 🧠 Intelligent Admission Control

- **TinyLFU Algorithm**: Count-Min Sketch with Doorkeeper prevents one-hit wonders from polluting the cache
- **Frequency Estimation**: 4-bit counters track access patterns with minimal memory overhead
- **Adaptive Aging**: Automatic counter decay maintains relevance over time

### 📊 Flexible Eviction Policies

- **LRU with Two Modes**:
  - **Listing Mode**: Precise LRU ordering for predictable eviction
  - **Sampling Mode**: Redis-inspired sampling for lower overhead on large caches
- **Soft & Hard Limits**: Proactive eviction at soft threshold, guaranteed enforcement at hard limit
- **Configurable Backoff**: Tune eviction aggressiveness based on workload

### ⏱️ Advanced TTL Management

- **Background Refresh**: Automatic cache warming before expiration
- **Stochastic Refresh**: Beta-distributed refresh times prevent thundering herd problems
- **Rate Limiting**: Configurable refresh rate to protect backend systems
- **Remove or Refresh**: Choose between automatic removal or background refresh on TTL expiry

### 📈 Production-Ready Observability

- **Delta-Based Telemetry**: Interval-based metrics suitable for Grafana dashboards
- **Per-Component Logging**: Separate metrics for admission, eviction, and lifetime management
- **Structured Logging**: JSON logs with `slog` for easy integration

## Installation

```bash
go get github.com/Borislavv/go-ash-cache
```

## Quick Start

```go
package main

import (
    "context"
    "log/slog"
    
    "github.com/Borislavv/go-ash-cache"
    "github.com/Borislavv/go-ash-cache/config"
    "github.com/Borislavv/go-ash-cache/internal/cache/db/model"
)

func main() {
    ctx := context.Background()
    logger := slog.Default()
    
    // Create configuration
    cfg := &config.Cache{
        DB: config.DBCfg{
            SizeBytes: 100 * 1024 * 1024, // 100MB cache
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
    
    // Initialize cache
    cache := ashcache.New(ctx, cfg, logger)
    defer cache.Close()
    
    // Use the cache
    data, err := cache.Get("user:123", func(item model.AshItem) ([]byte, error) {
        // This callback is only executed on cache miss
        // Fetch data from your data source
        return []byte("cached data"), nil
    })
    
    if err != nil {
        panic(err)
    }
    
    // data is now cached and will be returned on subsequent calls
    _ = data
}
```

## Configuration

AshCache uses YAML configuration files for production deployments. All components are optional and can be enabled independently.

### Basic Configuration

```yaml
db:
  size: 1073741824  # 1GB hard limit
  stat_logs_enabled: true
  telemetry_logs_interval: 5s
  cache_time_enabled: true
```

### With Eviction

```yaml
eviction:
  mode: listing  # or "sampling"
  soft_limit_coefficient: 0.8  # Start evicting at 80% capacity
  calls_per_sec: 10
  backoff_spins_per_call: 4096
```

### With Admission Control

```yaml
admission_control:
  capacity: 100000
  shards: 32
  min_table_len_per_shard: 128
  sample_multiplier: 10
  door_bits_per_counter: 2
```

### With TTL and Refresh

```yaml
lifetime:
  on_ttl: refresh  # or "remove"
  ttl: 24h
  rate: 100  # Max 100 refreshes per second
  coefficient: 0.5  # Start refreshing at 50% of TTL
  beta: 0.4
  stochastic_refresh_enabled: true
```

### Loading Configuration

```go
cfg, err := config.LoadConfig("cache.yaml")
if err != nil {
    log.Fatal(err)
}

cache := ashcache.New(ctx, cfg, logger)
```

## API Overview

### Core Operations

```go
// Get retrieves a value, calling callback on miss
data, err := cache.Get("key", func(item model.AshItem) ([]byte, error) {
    // Fetch and return data
    return fetchData(), nil
})

// Delete removes an entry
ok := cache.Del("key")

// Clear removes all entries
cache.Clear()

// Get cache statistics
len := cache.Len()      // Number of entries
mem := cache.Mem()      // Memory usage in bytes
```

### Metrics

```go
// Cache metrics (admission control)
allowed, notAllowed, hardEvictedItems, hardEvictedBytes := cache.CacheMetrics()

// Eviction metrics
scans, hits, evictedItems, evictedBytes := cache.EvictorMetrics()

// Lifetime metrics
affected, errors, scans, hits, misses := cache.LifetimerMetrics()
```

### TTL Management

```go
// Set TTL per item in callback
cache.Get("key", func(item model.AshItem) ([]byte, error) {
    item.SetTTL(1 * time.Hour)  // Custom TTL for this item
    return data, nil
})
```

## Performance Considerations

### Memory Usage

- Entry weight = `sizeof(Entry)` + `cap(payload)`
- Soft limit triggers proactive eviction
- Hard limit enforces strict memory bounds

### Eviction Modes

**Listing Mode** (Recommended for most cases):
- Precise LRU ordering
- Better hit rate prediction
- Slightly higher CPU overhead

**Sampling Mode** (For very large caches):
- Lower CPU overhead
- Good enough for most workloads
- Scales better with cache size

### Admission Control Tuning

- **Capacity**: Should match expected cache size
- **Shards**: More shards = less contention, more memory
- **SampleMultiplier**: Higher = less frequent aging, more memory per counter

### Concurrency

AshCache is fully thread-safe. All operations can be called concurrently from multiple goroutines without additional synchronization.

## Use Cases

### API Response Caching

Cache expensive API responses with automatic refresh:

```go
response, err := cache.Get("api:user:123", func(item model.AshItem) ([]byte, error) {
    item.SetTTL(5 * time.Minute)
    return fetchUserFromAPI(123)
})
```

### Database Query Results

Cache database queries with admission control to avoid cache pollution:

```go
results, err := cache.Get("query:popular:products", func(item model.AshItem) ([]byte, error) {
    return db.QueryPopularProducts()
})
```

### Computed Values

Cache expensive computations:

```go
result, err := cache.Get("compute:fibonacci:1000", func(item model.AshItem) ([]byte, error) {
    return computeFibonacci(1000), nil
})
```

## Architecture

### Sharding

The cache uses 1024 independent shards, each with its own lock. Keys are distributed across shards using a hash function, ensuring even distribution and minimal contention.

### Admission Control Flow

1. **Doorkeeper**: First access sets a bit (Bloom-like filter)
2. **Sketch**: Subsequent accesses increment frequency counters
3. **Decision**: New items are admitted only if their frequency estimate exceeds the victim's

### Eviction Flow

1. **Soft Limit**: Background evictor starts when memory exceeds soft threshold
2. **Hard Limit**: Immediate eviction when memory exceeds hard limit
3. **Victim Selection**: LRU-based (listing) or sampled (sampling mode)

### Refresh Flow

1. **Detection**: Background scanner finds expired entries
2. **Rate Limiting**: Refresh rate is capped to protect backends
3. **Stochastic Timing**: Beta distribution prevents synchronized refreshes

## Benchmarks

AshCache is designed for high-throughput scenarios. Typical performance characteristics:

- **Get (hit)**: < 100ns (zero allocations)
- **Get (miss)**: Depends on callback execution time
- **Set**: < 200ns (zero allocations on hot path)
- **Concurrent throughput**: Scales linearly with CPU cores

*Note: Actual performance depends on workload, cache size, and system configuration.*

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup

```bash
# Clone the repository
git clone https://github.com/Borislavv/go-ash-cache.git
cd go-ash-cache

# Run tests
go test ./...

# Run tests with race detector
go test -race ./...

# Run benchmarks
go test -bench=. ./...
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENCE](LICENCE) file for details.

## Acknowledgments

- TinyLFU algorithm by [Gil Einziger](https://github.com/dgryski) et al.
- Count-Min Sketch implementation inspired by academic research
- Stochastic cache expiration based on Google's Staleness paper (RFC 5861)

## Related Projects

- [groupcache](https://github.com/golang/groupcache) - Distributed caching library
- [bigcache](https://github.com/allegro/bigcache) - Fast, concurrent cache
- [freecache](https://github.com/coocood/freecache) - Zero GC cache

---

**Made with ❤️ for the Go community**
