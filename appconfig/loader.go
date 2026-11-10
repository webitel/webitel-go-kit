package appconfig

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/webitel/webitel-go-kit/appconfig/secret"
)

// Loader wires pflag → viper → struct unmarshalling for a specific set of
// config sections. Each CLI command constructs its own Loader so that only
// the flags relevant to that command are registered.
type Loader struct {
	v           *viper.Viper
	sections    Sections
	secrets     []string
	secretPaths []string
}

// NewLoader creates a Loader for the given sections. The secret keys of the
// enabled sections (postgres.dsn, postgres.standby_dsn, redis.password,
// pubsub.url) are marked automatically; see MarkSecret.
func NewLoader(s Sections) *Loader {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	l := &Loader{v: v, sections: s}
	if s.Postgres {
		l.MarkSecret("postgres.dsn", "postgres.standby_dsn")
	}
	if s.Redis {
		l.MarkSecret("redis.password")
	}
	if s.Pubsub {
		l.MarkSecret("pubsub.url")
	}
	return l
}

// MarkSecret declares config keys whose values are secrets. On Load, besides
// a literal, each of them accepts:
//
//	cred:<name>                     systemd credential $CREDENTIALS_DIRECTORY/<name>
//	file:<path>                     contents of a file
//	<ENV>_FILE=<path>               e.g. SERVICE_SECRET_KEY_FILE (when the key is unset)
//	credential named <key>          e.g. service.secret_key (when the key is unset)
//
// A flag or SetDefault default counts as unset, so a default never conflicts
// with a credential. Keys are case-insensitive, as in viper; the conventional
// credential name is the lower-case key. See package secret for the exact
// rules. Call before Load.
func (l *Loader) MarkSecret(keys ...string) *Loader {
	l.secrets = appendKeys(l.secrets, keys)
	return l
}

// MarkSecretPath declares config keys that hold the path of a secret file,
// such as a TLS private key. They accept the same sources as MarkSecret, but
// resolve to the file's path instead of its contents, so cred:service.key
// becomes $CREDENTIALS_DIRECTORY/service.key. Call before Load.
func (l *Loader) MarkSecretPath(keys ...string) *Loader {
	l.secretPaths = appendKeys(l.secretPaths, keys)
	return l
}

func appendKeys(dst, keys []string) []string {
	for _, key := range keys {
		dst = append(dst, strings.ToLower(key))
	}
	return dst
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
		fs.StringSlice("postgres.standby_dsn", nil, "Optional PostgreSQL read-replica DSNs")
		fs.Int("postgres.max_open_conns", 0, "Max open connections to the database (0 = unlimited)")
		fs.Int("postgres.max_idle_conns", 0, "Max idle connections in the pool (0 = driver default)")
		fs.Duration("postgres.conn_max_idle_time", 0, "Max time a connection may be idle (0 = no limit)")
		fs.Duration("postgres.conn_max_lifetime", 0, "Max time a connection may be reused (0 = no limit)")
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
		fs.String("pubsub.driver", "rabbitmq", "Broker driver (default: rabbitmq)")
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
	// Sync pflag defaults → viper defaults for flags that were not explicitly
	// set. Viper only reads pflag values when the flag is Changed; for unchanged
	// flags it falls through to env → config file → viper default.
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			return
		}
		l.v.SetDefault(f.Name, flagDefault(f))
	})

	if err := l.v.BindPFlags(fs); err != nil {
		return fmt.Errorf("appconfig: bind flags: %w", err)
	}

	if f := l.v.GetString("config_file"); f != "" {
		l.v.SetConfigFile(f)
		if err := l.v.ReadInConfig(); err != nil {
			return fmt.Errorf("appconfig: read config file %q: %w", f, err)
		}
	}

	if err := l.resolveSecrets(fs, secret.FromEnv()); err != nil {
		return err
	}

	if err := l.v.Unmarshal(target); err != nil {
		return fmt.Errorf("appconfig: unmarshal: %w", err)
	}

	return nil
}

// resolveSecrets replaces every secret key's reference with its resolved
// value. The result is stored as a viper override, so it survives config
// file reloads: secrets are resolved once per process, as systemd
// credentials are fixed for the lifetime of the unit anyway.
func (l *Loader) resolveSecrets(fs *pflag.FlagSet, r *secret.Resolver) error {
	for _, key := range l.secrets {
		if isStringSliceKey(fs, key) {
			if err := l.resolveSecretSlice(fs, r, key); err != nil {
				return err
			}
			continue
		}
		if err := l.resolveSecretScalar(fs, r.Value, key); err != nil {
			return err
		}
	}
	for _, key := range l.secretPaths {
		if err := l.resolveSecretScalar(fs, r.PathValue, key); err != nil {
			return err
		}
	}
	return nil
}

func (l *Loader) resolveSecretScalar(fs *pflag.FlagSet, fn func(key, v string) (string, error), key string) error {
	v := l.v.GetString(key)
	in := v
	if !secret.IsRef(v) && !l.isExplicit(fs, key) {
		// A default literal is a fallback, not a value to
		// conflict with: let a credential or _FILE replace it.
		in = ""
	}
	out, err := fn(key, in)
	if err != nil {
		return fmt.Errorf("appconfig: %w", err)
	}
	if out != "" && out != v {
		l.v.Set(key, out)
	}
	return nil
}

func (l *Loader) resolveSecretSlice(fs *pflag.FlagSet, r *secret.Resolver, key string) error {
	vals := l.v.GetStringSlice(key)
	if s, ok := l.v.Get(key).(string); ok {
		vals = splitComma(s)
	}
	explicit := l.isExplicit(fs, key)

	if len(vals) == 0 || !explicit {
		out, err := r.Value(key, "")
		if err != nil {
			return fmt.Errorf("appconfig: %w", err)
		}
		if out != "" {
			l.v.Set(key, []string{out})
		}
		return nil
	}

	resolved := make([]string, 0, len(vals))
	changed := false
	for _, raw := range vals {
		elem := strings.TrimSpace(raw)
		if elem == "" {
			changed = true
			continue
		}
		out, err := r.Value(key, elem)
		if err != nil {
			return fmt.Errorf("appconfig: %w", err)
		}
		if out == "" {
			changed = true
			continue
		}
		if out != raw {
			changed = true
		}
		resolved = append(resolved, out)
	}
	if changed {
		l.v.Set(key, resolved)
	}
	return nil
}

func splitComma(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

func isStringSliceKey(fs *pflag.FlagSet, key string) bool {
	f := fs.Lookup(key)
	if f == nil {
		return false
	}
	_, ok := f.Value.(pflag.SliceValue)
	return ok
}

// isExplicit reports whether key was set by a flag, an env var or the config
// file, as opposed to coming from a default.
func (l *Loader) isExplicit(fs *pflag.FlagSet, key string) bool {
	if f := fs.Lookup(key); f != nil && f.Changed {
		return true
	}
	if env := os.Getenv(secret.EnvName(key)); env != "" {
		return true
	}
	return l.v.InConfig(key)
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

func flagDefault(f *pflag.Flag) any {
	sv, ok := f.Value.(pflag.SliceValue)
	if !ok {
		return f.DefValue
	}
	return sv.GetSlice()
}
