# pgwfx

PostgreSQL pool manager for an fx application. The module starts a
`pgw.PoolManager` on application start, closes it on stop, and optionally
registers an informational health check from the primary pool state (no
`SELECT 1`).

The graph receives `*pgw.PoolManager`. Constraint processors can be registered
in constructors before `OnStart`; call `Primary()` per request.

## Usage

```go
import (
    "github.com/webitel/webitel-go-kit/infra/pgw"
    pgwfx "github.com/webitel/webitel-go-kit/infra/pgw/fx"
    healthfx "github.com/webitel/webitel-go-kit/infra/health/fx"
)

fx.New(
    fx.StartTimeout(60*time.Second), // bounds OnStart connect / pgw retries
    healthfx.Module(healthfx.Config{HTTPAddr: cfg.Health.Addr}),
    pgwfx.Module(pgwfx.Config{
        ApplicationName: model.ServiceName,
        DSN:             cfg.Postgres.DSN,
        MaxConns:        cfg.Postgres.MaxOpenConns,
        MinConns:        cfg.Postgres.MaxIdleConns,
        RegisterHealth:  true,
    }),
    fx.Provide(func(m *pgw.PoolManager) *Store {
        _ = m.RegisterUniqueViolation("users_email_key", mapEmailTaken)
        return NewStore(m)
    }),
    healthfx.Shutdown(),
)
```

`DSN` is required. Zero `MaxConns` / `MinConns` leave pgw defaults. Pass extra
`pgw.ConfigOption` values after the config for tracer, standby, migration
verifier, and so on:

```go
pgwfx.Module(cfg, pgw.WithTracer(myTracer))
```

## Health

With `RegisterHealth: true` and a `*health.Registry` in the graph (for example
from `healthfx.Module`), the module registers
`Informational(SystemPostgreSQL)` using `Primary()` — pool state, not a
round-trip ping. Without a registry the flag is a no-op.

Separate primary vs standby checks are out of scope for this module.

## Replacing a service-local module

```go
// before
postgres.Module,

// after
pgwfx.Module(pgwfx.Config{
    ApplicationName: model.ServiceName,
    DSN:             cfg.Postgres.DSN,
    MaxConns:        cfg.Postgres.MaxOpenConns,
    MinConns:        cfg.Postgres.MaxIdleConns,
    RegisterHealth:  true,
}),
fx.Provide(func(m *pgw.PoolManager) *Store {
    return NewStore(m)
}),
```

## Testing

```bash
cd infra/pgw/fx
go test -race ./...
```
