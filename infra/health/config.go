package health

import "time"

// Config holds the registry's timings. A zero field falls back to DefaultConfig.
type Config struct {
	Interval      time.Duration // how often each check runs
	Timeout       time.Duration // bounds one run; must be smaller than Interval
	FailThreshold int           // failures in a row before a check is StatusFail
	MinUnready    time.Duration // shortest time a check stays StatusFail
	StaleAfter    time.Duration // when a result stops counting and reads unknown
	DrainHold     time.Duration // how long Stop waits after Drain before letting go
}

// DefaultConfig returns a coherent set of timings, tuned together.
func DefaultConfig() Config {
	return Config{
		Interval:      5 * time.Second,
		Timeout:       2 * time.Second,
		FailThreshold: 3,
		MinUnready:    15 * time.Second,
		StaleAfter:    15 * time.Second,
		DrainHold:     10 * time.Second,
	}
}

func (c Config) withDefaults() Config {
	def := DefaultConfig()

	if c.Interval <= 0 {
		c.Interval = def.Interval
	}
	if c.Timeout <= 0 {
		c.Timeout = def.Timeout
	}
	if c.FailThreshold <= 0 {
		c.FailThreshold = def.FailThreshold
	}
	if c.MinUnready <= 0 {
		c.MinUnready = def.MinUnready
	}
	if c.StaleAfter <= 0 {
		c.StaleAfter = def.StaleAfter
	}
	if c.DrainHold <= 0 {
		c.DrainHold = def.DrainHold
	}

	return c
}
