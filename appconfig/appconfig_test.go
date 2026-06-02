package appconfig_test

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/webitel/webitel-go-kit/appconfig"
)

// testConfig mirrors a typical service config using all shared sections.
type testConfig struct {
	Log      appconfig.Log      `mapstructure:"log"`
	Postgres appconfig.Postgres `mapstructure:"postgres"`
	Redis    appconfig.Redis    `mapstructure:"redis"`
	Consul   appconfig.Consul   `mapstructure:"consul"`
	Pubsub   appconfig.Pubsub   `mapstructure:"pubsub"`
	Profiler appconfig.Profiler `mapstructure:"profiler"`
}

func newFlagSet() *pflag.FlagSet {
	return pflag.NewFlagSet("test", pflag.ContinueOnError)
}

// ── Defaults ─────────────────────────────────────────────────────────────────

func TestDefaults(t *testing.T) {
	// Clear env vars so defaults come from pflag, not the shell environment.
	// CONSUL must also be cleared: if CONSUL=<addr> is set (e.g. by the
	// Consul CLI), viper maps the key "consul" to a string which shadows the
	// nested "consul.addr" key.
	for _, env := range []string{
		"LOG_LEVEL", "LOG_JSON", "LOG_OTEL", "LOG_FILE", "LOG_CONSOLE",
		"REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB",
		"CONSUL", "CONSUL_ADDR",
		"PUBSUB_URL", "PUBSUB_DRIVER",
		"PROFILER_ADDR", "PROFILER_MUTEX_FRACTION", "PROFILER_BLOCK_RATE",
	} {
		t.Setenv(env, "")
		os.Unsetenv(env)
	}

	loader := appconfig.NewLoader(appconfig.Sections{
		Log: true, Postgres: true, Redis: true,
		Consul: true, Pubsub: true, Profiler: true,
	})
	fs := newFlagSet()
	loader.RegisterFlags(fs)
	_ = fs.Parse([]string{})

	var cfg testConfig
	if err := loader.Load(fs, &cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Log.Level != "info" {
		t.Errorf("log.level: want info, got %q", cfg.Log.Level)
	}
	if !cfg.Log.Console {
		t.Error("log.console: want true by default")
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Errorf("redis.addr: want localhost:6379, got %q", cfg.Redis.Addr)
	}
	if cfg.Consul.Addr != "localhost:8500" {
		t.Errorf("consul.addr: want localhost:8500, got %q", cfg.Consul.Addr)
	}
	if cfg.Pubsub.Driver != "rabbitmq" {
		t.Errorf("pubsub.driver: want rabbitmq by default, got %q", cfg.Pubsub.Driver)
	}
	if cfg.Profiler.MutexFraction != 1 {
		t.Errorf("profiler.mutex_fraction: want 1, got %d", cfg.Profiler.MutexFraction)
	}
}

// ── Flags ─────────────────────────────────────────────────────────────────────

func TestFlagOverride(t *testing.T) {
	loader := appconfig.NewLoader(appconfig.Sections{
		Log: true, Postgres: true, Redis: true,
	})
	fs := newFlagSet()
	loader.RegisterFlags(fs)
	_ = fs.Parse([]string{
		"--log.level=debug",
		"--log.json=true",
		"--postgres.dsn=postgres://user:pass@localhost/db",
		"--postgres.max_open_conns=10",
		"--postgres.conn_max_lifetime=1h",
		"--redis.addr=redis:6380",
		"--redis.db=3",
	})

	var cfg testConfig
	if err := loader.Load(fs, &cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Log.Level != "debug" {
		t.Errorf("log.level: want debug, got %q", cfg.Log.Level)
	}
	if !cfg.Log.JSON {
		t.Error("log.json: want true")
	}
	if cfg.Postgres.DSN != "postgres://user:pass@localhost/db" {
		t.Errorf("postgres.dsn: got %q", cfg.Postgres.DSN)
	}
	if cfg.Postgres.MaxOpenConns != 10 {
		t.Errorf("postgres.max_open_conns: want 10, got %d", cfg.Postgres.MaxOpenConns)
	}
	if cfg.Postgres.ConnMaxLifetime != time.Hour {
		t.Errorf("postgres.conn_max_lifetime: want 1h, got %v", cfg.Postgres.ConnMaxLifetime)
	}
	if cfg.Redis.Addr != "redis:6380" {
		t.Errorf("redis.addr: got %q", cfg.Redis.Addr)
	}
	if cfg.Redis.DB != 3 {
		t.Errorf("redis.db: want 3, got %d", cfg.Redis.DB)
	}
}

// ── Env vars ─────────────────────────────────────────────────────────────────

func TestEnvOverride(t *testing.T) {
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("POSTGRES_DSN", "postgres://env:secret@host/db")
	t.Setenv("POSTGRES_MAX_OPEN_CONNS", "25")
	t.Setenv("REDIS_ADDR", "envredis:6379")
	t.Setenv("CONSUL_ADDR", "envconsul:8500")
	t.Setenv("PUBSUB_URL", "amqp://env:pass@broker/")
	t.Setenv("PUBSUB_DRIVER", "rabbitmq")

	loader := appconfig.NewLoader(appconfig.Sections{
		Log: true, Postgres: true, Redis: true, Consul: true, Pubsub: true,
	})
	fs := newFlagSet()
	loader.RegisterFlags(fs)
	_ = fs.Parse([]string{})

	var cfg testConfig
	if err := loader.Load(fs, &cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Log.Level != "warn" {
		t.Errorf("LOG_LEVEL: want warn, got %q", cfg.Log.Level)
	}
	if cfg.Postgres.DSN != "postgres://env:secret@host/db" {
		t.Errorf("POSTGRES_DSN: got %q", cfg.Postgres.DSN)
	}
	if cfg.Postgres.MaxOpenConns != 25 {
		t.Errorf("POSTGRES_MAX_OPEN_CONNS: want 25, got %d", cfg.Postgres.MaxOpenConns)
	}
	if cfg.Redis.Addr != "envredis:6379" {
		t.Errorf("REDIS_ADDR: got %q", cfg.Redis.Addr)
	}
	if cfg.Consul.Addr != "envconsul:8500" {
		t.Errorf("CONSUL_ADDR: got %q", cfg.Consul.Addr)
	}
	if cfg.Pubsub.URL != "amqp://env:pass@broker/" {
		t.Errorf("PUBSUB_URL: got %q", cfg.Pubsub.URL)
	}
}

// ── YAML config file ─────────────────────────────────────────────────────────

func TestYAMLConfigFile(t *testing.T) {
	yaml := `
log:
  level: error
  json: true
postgres:
  dsn: postgres://yaml:pass@localhost/db
  max_open_conns: 5
  max_idle_conns: 2
  conn_max_idle_time: 10m
  conn_max_lifetime: 1h
redis:
  addr: yamlredis:6379
  db: 1
consul:
  addr: yamlconsul:8500
pubsub:
  url: amqp://yaml:pass@broker/
  driver: rabbitmq
profiler:
  addr: "0.0.0.0:6060"
  mutex_fraction: 2
  block_rate: 3
`
	f, err := os.CreateTemp(t.TempDir(), "config-*.yml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(yaml); err != nil {
		t.Fatal(err)
	}
	f.Close()

	loader := appconfig.NewLoader(appconfig.Sections{
		Log: true, Postgres: true, Redis: true,
		Consul: true, Pubsub: true, Profiler: true,
	})
	fs := newFlagSet()
	loader.RegisterFlags(fs)
	_ = fs.Parse([]string{"--config_file=" + f.Name()})

	var cfg testConfig
	if err := loader.Load(fs, &cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Log.Level != "error" {
		t.Errorf("log.level: want error, got %q", cfg.Log.Level)
	}
	if cfg.Postgres.DSN != "postgres://yaml:pass@localhost/db" {
		t.Errorf("postgres.dsn: got %q", cfg.Postgres.DSN)
	}
	if cfg.Postgres.MaxOpenConns != 5 {
		t.Errorf("postgres.max_open_conns: want 5, got %d", cfg.Postgres.MaxOpenConns)
	}
	if cfg.Postgres.MaxIdleConns != 2 {
		t.Errorf("postgres.max_idle_conns: want 2, got %d", cfg.Postgres.MaxIdleConns)
	}
	if cfg.Postgres.ConnMaxIdleTime != 10*time.Minute {
		t.Errorf("postgres.conn_max_idle_time: want 10m, got %v", cfg.Postgres.ConnMaxIdleTime)
	}
	if cfg.Postgres.ConnMaxLifetime != time.Hour {
		t.Errorf("postgres.conn_max_lifetime: want 1h, got %v", cfg.Postgres.ConnMaxLifetime)
	}
	if cfg.Redis.DB != 1 {
		t.Errorf("redis.db: want 1, got %d", cfg.Redis.DB)
	}
	if cfg.Pubsub.URL != "amqp://yaml:pass@broker/" {
		t.Errorf("pubsub.url: got %q", cfg.Pubsub.URL)
	}
	if cfg.Profiler.MutexFraction != 2 {
		t.Errorf("profiler.mutex_fraction: want 2, got %d", cfg.Profiler.MutexFraction)
	}
	if cfg.Profiler.BlockRate != 3 {
		t.Errorf("profiler.block_rate: want 3, got %d", cfg.Profiler.BlockRate)
	}
}

// ── Priority: flags > env > file ─────────────────────────────────────────────

func TestFlagBeatsEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "warn")

	loader := appconfig.NewLoader(appconfig.Sections{Log: true})
	fs := newFlagSet()
	loader.RegisterFlags(fs)
	_ = fs.Parse([]string{"--log.level=error"})

	var cfg testConfig
	if err := loader.Load(fs, &cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Log.Level != "error" {
		t.Errorf("flag should beat env: want error, got %q", cfg.Log.Level)
	}
}

// ── TLS validation ────────────────────────────────────────────────────────────

func TestValidateTLS(t *testing.T) {
	cases := []struct {
		name    string
		tls     appconfig.TLS
		wantErr bool
	}{
		{"all set", appconfig.TLS{CA: "ca.pem", Cert: "cert.pem", Key: "key.pem"}, false},
		{"missing CA", appconfig.TLS{Cert: "cert.pem", Key: "key.pem"}, true},
		{"missing Cert", appconfig.TLS{CA: "ca.pem", Key: "key.pem"}, true},
		{"missing Key", appconfig.TLS{CA: "ca.pem", Cert: "cert.pem"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := appconfig.ValidateTLS("service.conn", c.tls)
			if (err != nil) != c.wantErr {
				t.Errorf("wantErr=%v, got err=%v", c.wantErr, err)
			}
		})
	}
}

func TestValidateGRPCConn(t *testing.T) {
	t.Run("verify_certs=false skips validation", func(t *testing.T) {
		conn := appconfig.GRPCConn{VerifyCerts: false} // no certs set
		if err := appconfig.ValidateGRPCConn("service.conn", conn); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("verify_certs=true requires certs", func(t *testing.T) {
		conn := appconfig.GRPCConn{VerifyCerts: true} // missing certs
		if err := appconfig.ValidateGRPCConn("service.conn", conn); err == nil {
			t.Error("expected error for missing certs")
		}
	})
}

// ── RegisterGRPCConnFlags ─────────────────────────────────────────────────────

func TestRegisterGRPCConnFlags(t *testing.T) {
	fs := newFlagSet()
	appconfig.RegisterGRPCConnFlags(fs, "service.conn", true)

	expected := []string{
		"service.conn.verify_certs",
		"service.conn.ca",
		"service.conn.cert",
		"service.conn.key",
		"service.conn.client.ca",
		"service.conn.client.cert",
		"service.conn.client.key",
	}
	for _, name := range expected {
		if fs.Lookup(name) == nil {
			t.Errorf("flag %q not registered", name)
		}
	}

	// verify_certs default should be true
	f := fs.Lookup("service.conn.verify_certs")
	if f.DefValue != "true" {
		t.Errorf("verify_certs default: want true, got %q", f.DefValue)
	}
}

// ── Sections isolation ───────────────────────────────────────────────────────

func TestSectionsIsolation(t *testing.T) {
	// Loader with only Log should NOT register pubsub/redis flags.
	loader := appconfig.NewLoader(appconfig.Sections{Log: true})
	fs := newFlagSet()
	loader.RegisterFlags(fs)

	for _, name := range []string{"pubsub.url", "redis.addr", "postgres.dsn", "consul.addr"} {
		if fs.Lookup(name) != nil {
			t.Errorf("flag %q should not be registered for Sections{Log:true}", name)
		}
	}
	if fs.Lookup("log.level") == nil {
		t.Error("flag log.level should be registered")
	}
}
