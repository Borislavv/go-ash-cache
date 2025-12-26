package config

import "time"

type TTLMode string

var (
	TTLModeRemove  TTLMode = "remove"
	TTLModeRefresh TTLMode = "refresh"
)

type LifetimerCfg struct {
	// OnTTL defines what to do when an item reaches (or is close to) its TTL.
	// For example: refresh in background, keep stale, or remove.
	OnTTL TTLMode `yaml:"on_ttl"`

	// TTL is the maximum lifetime of a cached response without a successful refresh.
	// Example: "1d".
	TTL time.Duration `yaml:"ttl"`

	// Rate limits the maximum number of refresh operations (per time unit defined by the refresher),
	// used when OnTTL triggers background refreshes.
	// Example: 100.
	Rate int `yaml:"rate"`

	// Beta controls the coefficient used to compute a randomized cache refresh time.
	// Higher Beta values increase the probability of refreshing the cache before the TTL expires,
	// which reduces the risk of synchronized expirations (thundering herd).
	//
	// The formula is based on the "stochastic cache expiration" approach described in Google's
	// Staleness paper:
	//
	//   expireTime = ttl * (-Beta * ln(rand()))
	//
	// References: RFC 5861 and
	// https://web.archive.org/web/20100829170210/http://labs.google.com/papers/staleness.pdf
	//
	// Example:
	//   Beta: 0.4
	Beta float64 `yaml:"beta"` // Recommended range: (0, 1].

	// StochasticBetaRefreshEnabled enables stochastic (Beta-based) scheduling for refreshes.
	// When disabled, refresh scheduling falls back to the deterministic policy (e.g., Coefficient).
	StochasticBetaRefreshEnabled bool `yaml:"stochastic_refresh_enabled"`

	// Coefficient defines when to start refresh attempts relative to TTL.
	// The refresher starts trying to renew data after TTL * Coefficient has elapsed.
	// Example: TTL=24h and Coefficient=0.5 -> start refreshing after 12h.
	Coefficient float64 `yaml:"coefficient"` // Typical range: [0..1].

	// IsRemoveOnTTL is derived from OnTTL during initialization and is not read from YAML.
	// It is used internally as a fast path to decide whether an item should be removed at TTL.
	IsRemoveOnTTL bool // virtual: computed during init
}

func (cfg *LifetimerCfg) Enabled() bool {
	return cfg != nil
}
