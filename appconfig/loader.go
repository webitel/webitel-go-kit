package appconfig

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Loader wires pflag → viper → struct unmarshalling for a specific set of
// config sections. Each CLI command constructs its own Loader so that only
// the flags relevant to that command are registered.
type Loader struct {
	v        *viper.Viper
	sections Sections
}

// NewLoader creates a Loader for the given sections.
func NewLoader(s Sections) *Loader {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	return &Loader{v: v, sections: s}
}

// RegisterFlags registers a --config_file flag plus the flags for every
// enabled section into fs. Call this before pflag.Parse().
func (l *Loader) RegisterFlags(fs *pflag.FlagSet) {
	fs.String("config_file", "", "Configuration file (YAML, JSON, etc.)")

	if l.sections.Log {
		fs.String("log.level", "info", "Log level (debug|info|warn|error)")
		fs.Bool("log.json", false, "Emit logs as JSON")
		fs.Bool("log.otel", false, "Bridge logs to OpenTelemetry")
		fs.String("log.file", "", "Write logs to this file path")
		fs.Bool("log.console", true, "Write logs to stdout")
	}

	if l.sections.Postgres {
		fs.String("postgres.dsn", "", "PostgreSQL DSN (required)")
	}

	if l.sections.Redis {
		fs.String("redis.addr", "localhost:6379", "Redis address")
		fs.String("redis.password", "", "Redis password")
		fs.Int("redis.db", 0, "Redis database index")
	}

	if l.sections.Consul {
		fs.String("consul.addr", "localhost:8500", "Consul address")
	}

	if l.sections.Pubsub {
		fs.String("pubsub.url", "", "AMQP broker URL, e.g. amqp://user:pass@host/ (required)")
		fs.String("pubsub.driver", "", "Broker driver, e.g. rabbitmq (required)")
	}

	if l.sections.Profiler {
		fs.String("profiler.addr", "", "pprof listen address (disabled when empty)")
		fs.Int("profiler.mutex_fraction", 1, "runtime.SetMutexProfileFraction value")
		fs.Int("profiler.block_rate", 1, "runtime.SetBlockProfileRate value")
	}
}

// Load binds fs to the internal viper instance, reads the config file named
// by the --config_file flag (if set), and unmarshals the result into target.
// Call after pflag.Parse().
func (l *Loader) Load(fs *pflag.FlagSet, target any) error {
	if err := l.v.BindPFlags(fs); err != nil {
		return fmt.Errorf("appconfig: bind flags: %w", err)
	}

	if f := l.v.GetString("config_file"); f != "" {
		l.v.SetConfigFile(f)
		if err := l.v.ReadInConfig(); err != nil {
			return fmt.Errorf("appconfig: read config file %q: %w", f, err)
		}
	}

	if err := l.v.Unmarshal(target); err != nil {
		return fmt.Errorf("appconfig: unmarshal: %w", err)
	}

	return nil
}

// Watch registers fn to be called whenever the config file changes, then
// starts the fsnotify watcher. Has no effect if no config file was loaded.
func (l *Loader) Watch(fn func(fsnotify.Event)) {
	l.v.OnConfigChange(fn)
	l.v.WatchConfig()
}

// Viper returns the underlying viper instance for advanced use.
func (l *Loader) Viper() *viper.Viper {
	return l.v
}

// configError is a sentinel error type for invalid configuration values.
type configErr string

func (e configErr) Error() string { return "config: " + string(e) }

func configError(msg string) error { return configErr(msg) }

// IsConfigError reports whether err is a configuration validation error.
func IsConfigError(err error) bool {
	var e configErr
	return errors.As(err, &e)
}
