package config

// Cache groups configuration of all cache subsystems.
// Each component can be configured independently or disabled by setting it to nil.
type Cache struct {
	DB DBCfg `yaml:"db"`

	// AdmissionControl configures TinyLFU-style admission control.
	// It decides whether new items should be admitted into the cache
	// based on estimated access frequency.
	// If nil, admission control is disabled and items are admitted unconditionally.
	AdmissionControl *AdmissionControlCfg `yaml:"admission_control"`

	// Compression configures on-the-fly compression of cached values.
	// If nil, compression is disabled.
	Compression *CompressionCfg `yaml:"compression"`

	// Lifetime configures TTL handling and refresh behavior for cached items.
	// It controls item expiration, background refresh, and stochastic refresh policies.
	// If nil, items rely solely on eviction and are not refreshed based on TTL.
	Lifetime *LifetimerCfg `yaml:"lifetime"`

	// Eviction configures memory-based eviction policies.
	// It defines when and how cache entries are evicted to stay within memory limits.
	// If nil, eviction is disabled and cache size is unbounded (not recommended).
	Eviction *EvictionCfg `yaml:"eviction"`
}
