# discoveryfx

Service discovery for an fx application. The module creates the Consul provider,
registers the service instance on start and deregisters it on stop. It provides
`discovery.DiscoveryProvider`, which can be taken from the graph for
`GetService`, `KV` and so on.

The graph must contain a `*slog.Logger`.

## Usage

```go
import (
    discoveryfx "github.com/webitel/webitel-go-kit/infra/discovery/fx"
    healthfx "github.com/webitel/webitel-go-kit/infra/health/fx"
)

fx.New(
    healthfx.Module(healthfx.Config{HTTPAddr: cfg.Health.Addr}),
    discoveryfx.Module(discoveryfx.Config{
        Addr:      cfg.Consul.Addr,
        Name:      model.ServiceName,
        Version:   model.Version,
        Endpoints: []string{"grpc://" + cfg.Service.Addr},
        Metadata: map[string]string{
            "commit":         model.Commit,
            "commitDate":     model.CommitDate,
            "branch":         model.Branch,
            "buildTimestamp": model.BuildTimestamp,
        },
    }),
    // ... servers and dependencies ...
    healthfx.Shutdown(),
)
```

`Name` is required: the instance ID is built from it (`discovery.GenerateInstanceID`).
The first endpoint is the address Consul checks. `Timeout` for Consul calls
defaults to `DefaultTimeout` (30s). No separate `fx.Invoke` is needed to build
the provider — the module does it itself.

## Readiness

With a `*health.Registry` in the graph (for example, from `healthfx.Module`), the
TTL heartbeat reports its verdict to Consul through
`consul.WithReadiness(h.ReadyFunc())`: while the node is not ready, the check is
`critical`. Without a registry the heartbeat always reports `passing`.

## Shutdown order

`healthfx.Shutdown()` goes last, at the root of `fx.New`. Then on stop the drain
runs first (`DrainHold`, Consul sees `critical`), and `Deregister` after it. The
position of `discoveryfx.Module` relative to `Shutdown()` does not matter. Do not
wrap `healthfx.Shutdown()` in its own `fx.Module` — `Deregister` would then run
before the drain.

## Testing

```bash
cd infra/discovery/fx
go test -race ./...
```
