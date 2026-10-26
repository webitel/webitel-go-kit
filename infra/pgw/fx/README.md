# pgwfx

PostgreSQL pool manager for an fx application. The module opens a `pgw.PoolManager`
on start, closes it on stop, and optionally registers an informational health
check from the primary pool state (no `SELECT 1`).

The graph receives `*pgwfx.Pool`. Call `Manager()` for `*pgw.PoolManager`, or
`Primary()` / `Ready()` directly — the same shell pattern as the former
service-local `postgres.Module`.

## Usage

```go
import (
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
    fx.Provide(func(p *pgwfx.Pool) *Store {
        return NewStore(p) // hold *Pool; call Manager() / Primary() per request
    }),
    healthfx.Shutdown(),
)
```

Hold `*pgwfx.Pool` in the store — `Manager()` is nil until `OnStart` finishes.
Call `Manager()` / `Primary()` per request, not in the constructor.

`DSN` is required. Zero `MaxConns` / `MinConns` leave pgw defaults. Pass extra
`pgw.ConfigOption` values after the config for tracer, standby, migration
verifier, and so on:

```go
pgwfx.Module(cfg, pgw.WithTracer(myTracer))
```

## Health

With `RegisterHealth: true` and a `*health.Registry` in the graph (for example
from `healthfx.Module`), the module registers
`Informational(SystemPostgreSQL)` using `Pool.Ready` — primary `GetState` /
`Primary()`, not a round-trip ping. Without a registry the flag is a no-op.

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
fx.Provide(func(p *pgwfx.Pool) *Store {
    return NewStore(p)
}),
```

## Testing

```bash
cd infra/pgw/fx
go test -race ./...
```
