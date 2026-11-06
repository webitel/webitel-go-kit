package throttle

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Config sets the quota of each key. A field that is not positive takes the value of [DefaultConfig].
type Config struct {
	// Window is the period after which all quotas are reset.
	Window time.Duration
	// First is the number of records of a key that pass in each window.
	First int
	// Every passes each Every-th record of a key after the First records.
	Every int
	// MaxKeys bounds the keys tracked in one window. Records with new keys
	// above the bound pass without throttling.
	MaxKeys int
}

// DefaultConfig returns a window of one minute, 100 first records, then every
// 100th, and at most 10000 keys.
func DefaultConfig() Config {
	return Config{
		Window:  time.Minute,
		First:   100,
		Every:   100,
		MaxKeys: 10000,
	}
}

func (c Config) withDefaults() Config {
	def := DefaultConfig()
	if c.Window <= 0 {
		c.Window = def.Window
	}
	if c.First <= 0 {
		c.First = def.First
	}
	if c.Every <= 0 {
		c.Every = def.Every
	}
	if c.MaxKeys <= 0 {
		c.MaxKeys = def.MaxKeys
	}
	return c
}

// ParseConfig parses "off", or comma-separated fields over [DefaultConfig]:
// "window=1m,first=100,every=100,keys=10000". An empty string or "on" gives
// [DefaultConfig]. It returns nil for "off".
func ParseConfig(s string) (*Config, error) {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "off":
		return nil, nil
	case "", "on":
		cfg := DefaultConfig()
		return &cfg, nil
	}

	cfg := DefaultConfig()
	var errs []error
	for field := range strings.SplitSeq(s, ",") {
		name, value, ok := strings.Cut(strings.TrimSpace(field), "=")
		if !ok {
			errs = append(errs, fmt.Errorf("%q: want name=value", field))
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.TrimSpace(value)
		if err := cfg.set(name, value); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("throttle: %w", err)
	}
	return &cfg, nil
}

func (c *Config) set(name, value string) error {
	if name == "window" {
		d, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		if d <= 0 {
			return errors.New("must be positive")
		}
		c.Window = d
		return nil
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	if n <= 0 {
		return errors.New("must be positive")
	}
	switch name {
	case "first":
		c.First = n
	case "every":
		c.Every = n
	case "keys":
		c.MaxKeys = n
	default:
		return errors.New("unknown field")
	}
	return nil
}
