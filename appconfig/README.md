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
| `service.conn.mtls` | `SERVICE_CONN_MTLS` |
| `service.conn.ca` | `SERVICE_CONN_CA` |
| `service.conn.cert` | `SERVICE_CONN_CERT` |
| `service.conn.key` | `SERVICE_CONN_KEY` |
| `service.conn.client.ca` | `SERVICE_CONN_CLIENT_CA` |
| `service.conn.client.cert` | `SERVICE_CONN_CLIENT_CERT` |
| `service.conn.client.key` | `SERVICE_CONN_CLIENT_KEY` |

Priority order: **CLI flag > environment variable > config file > default**.

## Secrets

Passwords, tokens and DSNs do not have to sit in plain text in
`/etc/default/<service>` or the YAML config. A key marked as a secret accepts,
besides a literal, a reference to a systemd credential or a file, and the
service still gets a plain `string` in its config struct. The implementation
lives in [`appconfig/secret`](secret) (stdlib only); this section covers the
`Loader` side. The full guide — systemd encryption, unit drop-ins, Docker /
Kubernetes, security limits — is [secret/README.md](secret/README.md).

### Which keys are secrets

Marked automatically for the enabled sections — bumping this module is enough:

| Section | Key | Credential name | `_FILE` env |
|---|---|---|---|
| `Postgres` | `postgres.dsn` | `postgres.dsn` | `POSTGRES_DSN_FILE` |
| `Redis` | `redis.password` | `redis.password` | `REDIS_PASSWORD_FILE` |
| `Pubsub` | `pubsub.url` | `pubsub.url` | `PUBSUB_URL_FILE` |

A service marks its own keys before `Load`:

```go
loader.
    MarkSecret("service.secret_key").                                // value keys
    MarkSecretPath(appconfig.GRPCConnSecretPaths("service.conn")...) // TLS private-key paths
```

- `MarkSecret` — the key ends up holding the secret's **contents**.
- `MarkSecretPath` — for settings that take a **file path** (TLS keys): the key
  ends up holding the path of the source, e.g. `cred:service.key` →
  `$CREDENTIALS_DIRECTORY/service.key`. `GRPCConnSecretPaths(prefix)` returns
  `<prefix>.key` and `<prefix>.client.key`.

### Where a secret can come from

| Configured value (flag / env / YAML) | Result |
|---|---|
| `cred:<name>` | contents of systemd credential `$CREDENTIALS_DIRECTORY/<name>` |
| `file:<path>` | contents of `<path>` |
| other literal | used as is — existing deployments keep working |
| unset or only a default | 1. file named by `<ENV>_FILE` (`POSTGRES_DSN_FILE=/run/secrets/pg_dsn`) |
| | 2. systemd credential named like the key (`postgres.dsn`) |
| | 3. the default, if any; otherwise empty |

So a systemd credential named after the key needs **no config change at all**:

```ini
# systemctl edit webitel-im-account
[Service]
LoadCredentialEncrypted=postgres.dsn:/etc/credstore.encrypted/webitel-im-account.postgres.dsn
```

and a Docker / Kubernetes secret is one env var: `POSTGRES_DSN_FILE=/run/secrets/pg_dsn`.

### Behaviour

- **Resolved once, in `Load`**, before unmarshalling. Resolved values are
  stored as viper overrides, so `Watch` reloads (`loader.Viper().Unmarshal`)
  keep them. A changed reference or a rotated secret takes effect on restart —
  systemd loads credentials only at service start anyway.
- **Conflicts fail `Load`.** A key set explicitly (flag, env var, config file)
  *and* present as a conventional source (`POSTGRES_DSN=...` plus a
  `postgres.dsn` credential) returns `secret.ErrConflict`, so a forgotten
  plain-text value cannot silently win over the credential that replaced it.
  `postgres.dsn: cred:postgres.dsn` is the same source and is fine. Defaults
  never conflict.
- **Errors:** `secret.ErrNoCredentialsDir` for `cred:` outside systemd,
  `secret.ErrEmpty` for an empty file, and an error for a missing file or one
  over 1 MiB. They are wrapped as `appconfig: ...`; the sentinels (and
  `fs.ErrNotExist`) still match with `errors.Is`.
  Messages name the key, credential and path, never the secret itself.
- **File contents:** trailing `\n` / `\r\n` is trimmed; other whitespace is kept.
- **Keys are case-insensitive**; credential names are the lower-case key.
- Only marked keys are touched. A literal secret that really starts with
  `cred:` or `file:` has to be supplied through a file.

### Testing without systemd

`CREDENTIALS_DIRECTORY` is just an env var, so a test can point it at
`t.TempDir()`:

```go
dir := t.TempDir()
os.WriteFile(filepath.Join(dir, "postgres.dsn"), []byte("postgres://db.example.com/test\n"), 0o400)
t.Setenv("CREDENTIALS_DIRECTORY", dir)
// loader.Load(...) now reads postgres.dsn from the "credential"
```

On a Linux box with systemd, a one-off run gets real credentials:

```sh
sudo systemd-run --pty --uid=$(id -u) \
  -p LoadCredential=postgres.dsn:/path/to/pg_dsn ./webitel-im-account server
```

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
    SecretKey  string             `mapstructure:"secret_key"`
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
    pflag.String("service.secret_key", "", "Token encryption key (required)")
    appconfig.RegisterGRPCConnFlags(pflag.CommandLine, "service.conn", true)

    // postgres.dsn, redis.password and pubsub.url are already marked.
    loader.
        MarkSecret("service.secret_key").
        MarkSecretPath(appconfig.GRPCConnSecretPaths("service.conn")...)

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

The same file with secrets kept out of it — values come from systemd
credentials named after the keys (`postgres.dsn`, `service.secret_key`) or
from explicit references:

```yaml
postgres:
  max_open_conns: 25          # dsn: systemd credential "postgres.dsn"

pubsub:
  url: cred:amqp              # credential with a different name

service:
  conn:
    verify_certs: true
    ca: /etc/webitel/tls/ca.crt
    cert: /etc/webitel/tls/service.crt
    key: cred:service.key     # resolves to the credential's path
```
