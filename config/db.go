package config

import "time"

type DBCfg struct {
	SizeBytes              int64         `yaml:"size"`
	IsTelemetryLogsEnabled bool          `yaml:"stat_logs_enabled"`
	TelemetryLogsInterval  time.Duration `yaml:"5s"`
	CacheTimeEnabled       bool          `yaml:"cache_time_enabled"`
}
