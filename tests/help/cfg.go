package help

import (
	"github.com/Borislavv/go-ash-cache/internal/config"
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
