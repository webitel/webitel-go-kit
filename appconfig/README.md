# appconfig

Shared configuration primitives for Webitel services.

Provides canonical struct definitions for common infrastructure sections (Log, Postgres, Redis, Consul, Pubsub, Profiler, TLS) and a section-based `Loader` that wires `pflag → env → YAML → struct` with a single call.

## Env var naming convention

Dots and hyphens in flag names are replaced with underscores for the env var equivalent:

| Flag | Env |
|---|---|
| `log.level` | `LOG_LEVEL` |
| `postgres.dsn` | `POSTGRES_DSN` |
| `postgres.max_open_conns` | `POSTGRES_MAX_OPEN_CONNS` |
| `postgres.max_idle_conns` | `POSTGRES_MAX_IDLE_CONNS` |
| `postgres.conn_max_idle_time` | `POSTGRES_CONN_MAX_IDLE_TIME` |
| `postgres.conn_max_lifetime` | `POSTGRES_CONN_MAX_LIFETIME` |
| `redis.addr` | `REDIS_ADDR` |
| `redis.password` | `REDIS_PASSWORD` |
| `redis.db` | `REDIS_DB` |
| `consul.addr` | `CONSUL_ADDR` |
| `pubsub.url` | `PUBSUB_URL` |
| `pubsub.driver` | `PUBSUB_DRIVER` (default: `rabbitmq`) |
| `profiler.addr` | `PROFILER_ADDR` |
| `profiler.mutex_fraction` | `PROFILER_MUTEX_FRACTION` |
| `profiler.block_rate` | `PROFILER_BLOCK_RATE` |
| `service.conn.verify_certs` | `SERVICE_CONN_VERIFY_CERTS` |
| `service.conn.ca` | `SERVICE_CONN_CA` |
| `service.conn.cert` | `SERVICE_CONN_CERT` |
| `service.conn.key` | `SERVICE_CONN_KEY` |
| `service.conn.client.ca` | `SERVICE_CONN_CLIENT_CA` |
| `service.conn.client.cert` | `SERVICE_CONN_CLIENT_CERT` |
| `service.conn.client.key` | `SERVICE_CONN_CLIENT_KEY` |

Priority order: **CLI flag > environment variable > config file > default**.

## Integration example

### 1. Define the service config struct

```go
// config/config.go
package config

import (
    "fmt"
    "log/slog"
    "strings"

    "github.com/fsnotify/fsnotify"
    "github.com/spf13/pflag"
    "github.com/webitel/webitel-go-kit/appconfig"
)

type Config struct {
    Service  ServiceConfig      `mapstructure:"service"`
    Log      appconfig.Log      `mapstructure:"log"`
    Postgres appconfig.Postgres `mapstructure:"postgres"`
    Redis    appconfig.Redis    `mapstructure:"redis"`
    Consul   appconfig.Consul   `mapstructure:"consul"`
    Pubsub   appconfig.Pubsub   `mapstructure:"pubsub"`
    Profiler appconfig.Profiler `mapstructure:"profiler"`
}

type ServiceConfig struct {
    Addr       string             `mapstructure:"addr"`
    Connection appconfig.GRPCConn `mapstructure:"conn"`
}
```

### 2. server command — full config

```go
func LoadServerConfig() (*Config, error) {
    loader := appconfig.NewLoader(appconfig.Sections{
        Log: true, Postgres: true, Redis: true,
        Consul: true, Pubsub: true, Profiler: true,
    })
    loader.RegisterFlags(pflag.CommandLine)

    pflag.String("service.addr", "localhost:8080", "gRPC listen address")
    appconfig.RegisterGRPCConnFlags(pflag.CommandLine, "service.conn", true)

    pflag.Parse()

    cfg := &Config{}
    if err := loader.Load(pflag.CommandLine, cfg); err != nil {
        return nil, err
    }

    loader.Watch(func(e fsnotify.Event) {
        slog.Info("config file changed", "name", e.Name)
        newCfg := &Config{}
        if err := loader.Viper().Unmarshal(newCfg); err != nil {
            slog.Error("config reload failed", "error", err)
            return
        }
        *cfg = *newCfg
    })

    return cfg, cfg.validate()
}
```

### 3. migrate command — minimal config

```go
func LoadMigrateConfig() (*Config, error) {
    loader := appconfig.NewLoader(appconfig.Sections{
        Log: true, Postgres: true,
    })
    loader.RegisterFlags(pflag.CommandLine)
    pflag.Parse()

    cfg := &Config{}
    if err := loader.Load(pflag.CommandLine, cfg); err != nil {
        return nil, err
    }
    if cfg.Postgres.DSN == "" {
        return nil, fmt.Errorf("config: postgres.dsn is required")
    }
    return cfg, nil
}
```

### 4. Apply Postgres pool settings

```go
db, err := sql.Open("postgres", cfg.Postgres.DSN)
if err != nil { ... }
cfg.Postgres.ApplyToSQLDB(db)
```

### 5. Example config file

```yaml
log:
  level: info
  json: true
  console: true

postgres:
  dsn: postgres://user:pass@localhost:5432/webitel?sslmode=disable
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_idle_time: 10m
  conn_max_lifetime: 1h

redis:
  addr: localhost:6379
  db: 0

consul:
  addr: localhost:8500

pubsub:
  url: amqp://user:pass@localhost:5672/

profiler:
  addr: 0.0.0.0:6060
  mutex_fraction: 1
  block_rate: 1
```
