package config

// LRUMode defines the LRU eviction strategy.
type LRUMode string

const (
	// LRUModeSampling evicts entries using Redis-like sampling for victim selection.
	LRUModeSampling LRUMode = "sampling"

	// LRUModeListing evicts entries by iterating over the LRU list directly.
	LRUModeListing LRUMode = "listing"
)

type EvictionCfg struct {
	// LRUMode defines the LRU eviction mode.
	// Supported values:
	//   - "sampling": eviction is based on sampling a subset of entries
	//   - "listing":  eviction iterates over the LRU list directly
	LRUMode LRUMode `yaml:"mode"`

	// SoftLimitCoefficient defines the soft memory usage threshold as a fraction of cfg.DB.SizeBytes.
	// When memory usage exceeds this limit, eviction may start proactively.
	//
	// Example:
	//   SoftLimitCoefficient: 0.80 // start evicting after reaching 80% of cfg.DB.SizeBytes
	SoftLimitCoefficient float64 `yaml:"soft_limit_coefficient"`

	// SoftMemoryLimitBytes is derived during initialization from cfg.DB.SizeBytes and SoftLimitCoefficient.
	// It is not read from YAML.
	SoftMemoryLimitBytes int64 // virtual: computed during init (bytes)

	// CallsPerSec defines how many eviction scan cycles the evictor performs per second.
	// Increasing this value makes eviction more responsive but increases CPU usage.
	CallsPerSec int64 `yaml:"calls_per_sec"`

	// BackoffSpinsPerCall defines how many cache entries are checked during a single eviction scan.
	// The total number of entries scanned per second is:
	//
	//   CallsPerSec * BackoffSpinsPerCall
	//
	// Example:
	//   CallsPerSec:         10
	//   BackoffSpinsPerCall: 4096
	//   => ~40,960 cache entries scanned per second
	//
	// Tune this value based on cache size and workload characteristics.
	BackoffSpinsPerCall int64 `yaml:"backoff_spins_per_call"`

	// IsListing is derived from LRUMode during initialization.
	// It is used internally as a fast-path flag to avoid repeated comparisons.
	// This field is not read from YAML.
	IsListing bool // virtual: computed during init
}

func (cfg *EvictionCfg) Enabled() bool {
	return cfg != nil
}
