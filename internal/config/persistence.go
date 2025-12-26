package config

type PersistenceCfg struct {
	// Dir specifies the directory where cache dump files are stored.
	// The directory must exist and be writable.
	Dir string `yaml:"dump_dir"`

	// Name defines the base name of the cache dump file.
	// The final file name may include extensions depending on configuration
	// (e.g., ".gz" when Gzip is enabled).
	Name string `yaml:"dump_name"`

	// Gzip enables gzip compression for cache dump files.
	// When enabled, dumps are written and read in compressed form,
	// reducing disk usage at the cost of additional CPU overhead.
	Gzip bool `yaml:"gzip"`
}

func (cfg *PersistenceCfg) Enabled() bool {
	return cfg != nil
}
