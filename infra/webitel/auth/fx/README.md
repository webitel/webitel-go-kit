# authfx

Authorization of gRPC calls for an fx application. The module opens the
connection to go.webitel.app with `auth.Dial` (`discovery:///go.webitel.app`,
round-robin, with otelgrpc), closes it on stop, and provides `auth.Client` and
`*auth.Authorizer` from `infra/webitel/auth`; a service without `WebitelAPI` passes `nil`
and gets only the `auth.Client`, whose `Authenticate` goes to the interceptors.

The graph must contain a `discovery.DiscoveryProvider`, for example from
`discoveryfx.Module`.

## Usage

```go
import (
    authfx "github.com/webitel/webitel-go-kit/infra/webitel/auth/fx"
    discoveryfx "github.com/webitel/webitel-go-kit/infra/discovery/fx"
)

fx.New(
    discoveryfx.Module(discoveryfx.Config{ /* ... */ }),
    authfx.Module(authfx.Config{}, auditpb.WebitelAPI),
    fx.Provide(newServer),
)

func newServer(a *auth.Authorizer) *grpc.Server {
    return grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            interceptors.UnaryServerErrorInterceptor(),
            interceptors.AuthUnaryServerInterceptor(a.Authorize),
        ),
        grpc.ChainStreamInterceptor(interceptors.AuthStreamServerInterceptor(a.Authorize)),
    )
}
```

`auditpb.WebitelAPI` is the variable that `protoc-gen-go-webitel` generates in
the service package. An invalid value makes `fx.New` fail at startup. Public
methods such as health checks are left out with the `selector` of go-grpc-middleware.

## Config

| Field | Default | Meaning |
|---|---|---|
| `Service` | `go.webitel.app` | discovery name of the auth service |
| `TLS` | nil (no TLS) | TLS config of the connection, for example from `ClientTLSConfig(appconfig.GRPCConn)` of `infra/transport/gRPC` |
| `CacheTTL` | 10s | session cache TTL |
| `DisableCache` | false | turns the session cache off |
| `CacheSize` | 10000 | max cached sessions |
| `CallTimeout` | 5s | timeout of one `UserInfo` call |

## Testing

```bash
cd infra/webitel/auth/fx
GOWORK=off go vet ./...
GOWORK=off go test -race ./...
```

The tests use `fxtest` with a discovery provider that is never called: gRPC
builds the resolver on the first RPC, and the tests make none.
