package config

// AdmissionControlCfg configures TinyLFU-style admission control.
// It estimates item popularity (e.g., via a sketch + doorkeeper) to decide whether a new item
// should be admitted into the cache.
//
// Note: when Enabled is false, a NoOp admission controller is used (items are admitted unconditionally).
type AdmissionControlCfg struct {
	// Capacity is the logical size used to dimension admission-control data structures.
	// Typically aligned with the cache capacity (number of items).
	Capacity int `yaml:"capacity"`

	// Shards defines how many independent admission-control shards to use.
	// Sharding reduces lock contention and improves scalability on multi-core systems.
	Shards int `yaml:"shards"`

	// MinTableLenPerShard sets a lower bound for internal table length per shard.
	// This prevents undersized tables when Capacity is low or Shards is high.
	MinTableLenPerShard int `yaml:"min_table_len_per_shard"`

	// SampleMultiplier controls the admission sampling intensity.
	// Higher values usually improve admission quality at the cost of extra CPU work.
	SampleMultiplier int `yaml:"sample_multiplier"`

	// DoorBitsPerCounter configures the size/precision of the doorkeeper (Bloom-like) structure.
	// More bits reduce false positives but increase memory usage.
	DoorBitsPerCounter int `yaml:"door_bits_per_counter"`
}

func (cfg *AdmissionControlCfg) Enabled() bool {
	return cfg != nil
}
