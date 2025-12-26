package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

func (cfg *Cache) AdjustConfig() {
	if cfg.Eviction.Enabled() {
		cfg.Eviction.IsListing = cfg.Eviction.LRUMode == LRUModeListing
		cfg.Eviction.SoftMemoryLimitBytes = int64(float64(cfg.DB.SizeBytes) * cfg.Eviction.SoftLimitCoefficient)
	}

	if cfg.Lifetime.Enabled() {
		if cfg.Lifetime.OnTTL == TTLModeRefresh {
			cfg.Lifetime.IsRemoveOnTTL = false
		} else {
			cfg.Lifetime.IsRemoveOnTTL = true
		}
	}
}

func LoadConfig(path string) (*Cache, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("stat config path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config yaml file %s: %w", path, err)
	}

	var cfg *Cache
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal yaml from %s: %w", path, err)
	}
	cfg.AdjustConfig()

	return cfg, nil
}
