package help

import (
	"github.com/Borislavv/go-ash-cache/config"
	"time"
)

func Cfg() *config.Cache {
	c := &config.Cache{
		Lifetime: &config.LifetimerCfg{
			OnTTL:         config.TTLModeRefresh,
			TTL:           time.Minute * 5,
			Rate:          1000,
			Beta:          0.5,
			Coefficient:   0.5,
			IsRemoveOnTTL: false,
		},
		Eviction: &config.EvictionCfg{
			LRUMode:              config.LRUModeListing,
			SoftLimitCoefficient: 0.8,
			SoftMemoryLimitBytes: 1024 * 1024 * 800,
			CallsPerSec:          5,
			BackoffSpinsPerCall:  1024,
			IsListing:            true,
		},
		DB: config.DBCfg{
			SizeBytes:              1024 * 1024 * 1024,
			IsTelemetryLogsEnabled: true,
			TelemetryLogsInterval:  time.Second * 5,
			CacheTimeEnabled:       true,
		},
	}
	c.AdjustConfig()
	return c
}

func EvictionCfg() *config.Cache {
	c := Cfg()
	c.Eviction = &config.EvictionCfg{
		LRUMode:              config.LRUModeListing,
		SoftLimitCoefficient: 0.8,
		SoftMemoryLimitBytes: 1024 * 1024 * 8,
		CallsPerSec:          5,
		BackoffSpinsPerCall:  1024,
	}
	c.Lifetime = nil
	return c
}

func LifetimerRefreshStochasticCfg() *config.Cache {
	c := Cfg()
	c.Lifetime = &config.LifetimerCfg{
		OnTTL:                        config.TTLModeRefresh,
		TTL:                          5 * time.Second,
		Rate:                         1_000_000,
		Beta:                         0.5,
		StochasticBetaRefreshEnabled: true,
		Coefficient:                  0.5,
		IsRemoveOnTTL:                false,
	}
	c.Eviction = nil
	return c
}

func LifetimerRefreshCfg() *config.Cache {
	c := Cfg()
	c.Lifetime = &config.LifetimerCfg{
		OnTTL:                        config.TTLModeRefresh,
		TTL:                          5 * time.Second,
		Rate:                         1_000_000,
		Beta:                         0.5,
		StochasticBetaRefreshEnabled: false,
		Coefficient:                  0.5,
		IsRemoveOnTTL:                false,
	}
	c.Eviction = nil
	return c
}

func LifetimerRemoveStochasticCfg() *config.Cache {
	c := Cfg()
	c.Lifetime = &config.LifetimerCfg{
		OnTTL:                        config.TTLModeRemove,
		TTL:                          6 * time.Second,
		Rate:                         1_000_000,
		Beta:                         0.5,
		StochasticBetaRefreshEnabled: true,
		Coefficient:                  0.5,
		IsRemoveOnTTL:                true,
	}
	c.Eviction = nil
	return c
}

func LifetimerRemoveCfg() *config.Cache {
	c := Cfg()
	c.Lifetime = &config.LifetimerCfg{
		OnTTL:                        config.TTLModeRemove,
		TTL:                          5 * time.Second,
		Rate:                         1_000_000,
		Beta:                         0.5,
		StochasticBetaRefreshEnabled: false,
		Coefficient:                  0.5,
		IsRemoveOnTTL:                true,
	}
	c.Eviction = nil
	return c
}

func TinyLFUCfg() *config.AdmissionControlCfg {
	return &config.AdmissionControlCfg{
		Capacity:            128,
		Shards:              4,
		MinTableLenPerShard: 64,
		SampleMultiplier:    3,
		DoorBitsPerCounter:  2,
	}
}
