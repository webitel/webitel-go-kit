package appconfig

// Profiler holds pprof profiler configuration.
// Env vars: PROFILER_ADDR, PROFILER_MUTEX_FRACTION, PROFILER_BLOCK_RATE.
//
// Canonical field names — supersedes per-service variants:
//   - mutex_profile_fraction (contact-service) → mutex_fraction
//   - block_profile_rate     (contact-service) → block_rate
type Profiler struct {
	Addr          string `mapstructure:"addr"`
	MutexFraction int    `mapstructure:"mutex_fraction"`
	BlockRate     int    `mapstructure:"block_rate"`
}
