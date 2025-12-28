# AshCache — high-performance in-memory cache for Go

[![Go Version](https://img.shields.io/static/v1?label=Go&message=1.23%2B&logo=go&color=00ADD8)](https://go.dev/dl/) [![Coverage](https://img.shields.io/codecov/c/github/Borislavv/go-ash-cache?label=coverage)](https://codecov.io/gh/Borislavv/go-ash-cache) [![License](https://img.shields.io/badge/License-Apache--2.0-green.svg)](./LICENSE)


AshCache is a production-oriented **in-memory cache library for Go** designed for high throughput and predictable latency.
It combines:

- **Sharded storage** for scalability (reduced lock contention)
- **LRU eviction** (listing or sampling style)
- **TinyLFU-style admission control** (Count-Min Sketch + Doorkeeper)
- **TTL + background refresh** (including stochastic/beta refresh to avoid cache stampedes)
- **Telemetry logs** with interval deltas (Grafana-like “last N seconds” view)

> If you need a fast cache with admission control (avoid one-hit wonders), predictable eviction, and refresh logic — AshCache is built for that.
