# auth

Authentication and authorization of gRPC calls for Webitel services. It replaces
the copies of `internal/auth` and `interceptors/auth.go` that lived in every
service.

The caller's access token (`x-webitel-access`) is exchanged for a `Session`
through `Auth.UserInfo` of go.webitel.app. The interceptor then checks the call
against the `WebitelAPI` map that `protoc-gen-go-webitel` generates:

1. public methods (health checks, reflection) pass without a token;
2. the method must be in the map, otherwise it is denied (`PermissionDenied`);
3. the token must be present and the session not expired;
4. every license in the service's `additional_license` must be valid now
   (`IssuedAt <= now < ExpiresAt`, matched by `License.Prod`);
5. obac: the user needs the action on the object class.

## Usage

```go
import authgrpc "buf.build/gen/go/webitel/webitel-go/grpc/go/_gogrpc"

authorizer := auth.New(authgrpc.NewAuthClient(conn)) // conn: gRPC connection to go.webitel.app
interceptor, err := auth.NewInterceptor(authorizer, auditpb.WebitelAPI)
if err != nil {
    return err
}

srv := grpc.NewServer(
    grpc.ChainUnaryInterceptor(errorInterceptor, recoveryInterceptor, interceptor.Unary()),
    grpc.ChainStreamInterceptor(interceptor.Stream()),
)
```

`WebitelAPI` is passed as `any` and converted by field name at startup, so
`NewInterceptor` fails at once if its shape ever changes.

`errorInterceptor` is `UnaryServerErrorInterceptor` of `pkg/interceptors`; without
it unary errors reach clients as `Unknown`. It also rewrites the IDs below to
`api.process.*` for clients. `Stream()` sets the gRPC code itself, as plain text.

In a handler:

```go
s, ok := auth.SessionFromContext(ctx)
if !ok {
    // only on public methods, which the interceptor lets through without a session
    return nil, errors.Unauthenticated("no session")
}
if s.RequiresRBAC("forms", auth.Read) {
    // restrict records to those reachable by s.ACLRoles()
}
```

The interceptor does not apply record-based access control (rbac); that is the
handler's job through `Session.RequiresRBAC` and `Session.ACLRoles`.
In tests build the handler context with `auth.ContextWithSession`.

A stream is authorized once, when it opens. A token that is revoked or expires
later does not close a stream that is already running.

Health checks and server reflection need no token. Add more with
`auth.WithPublicMethods("/pkg.Service/")`; any other prefix shape, such as
`"/pkg.Pub"` or `"/"`, makes `NewInterceptor` fail. Service-internal APIs that use their
own credentials belong to the service: make them public here and put your own
guard in the chain before this interceptor.

## Access and permissions

| `access` in proto | `Access` | Flag in `Scope.Access` | Super permission |
|---|---|---|---|
| create (also when unset) | `Create` | `x` | `add` |
| read | `Read` | `r` | `read` |
| update | `Update` | `w` | `write` |
| delete | `Delete` | `d` | `delete` |

A method without an `access` option is generated as `Create`, so it needs `x`.
A class missing from the user's scope (go.webitel.app omits classes without
read) is denied. With `Obac` off for the class the action is allowed; with it
on the user needs the flag or the super permission.

## Errors

Errors come from `pkg/errors`, so `UnaryServerErrorInterceptor` maps them to
HTTP statuses.

| Situation | gRPC code | ID |
|---|---|---|
| method not in `WebitelAPI`, no object class, bad access | `PermissionDenied` | `auth.method.unknown` |
| no token | `Unauthenticated` | `auth.token.missing` |
| go.webitel.app rejects the token | `Unauthenticated` | `auth.token.invalid` |
| session expired | `Unauthenticated` | `auth.session.expired` |
| go.webitel.app unavailable, timeout or other error | `Unavailable` | `auth.provider.unavailable` |
| caller gave up while waiting | `Canceled` / `DeadlineExceeded` | `auth.request.canceled` |
| missing license (all missing are listed) | `PermissionDenied` | `auth.license` |
| obac denied | `PermissionDenied` | `auth.permission` |

An outage of go.webitel.app answers `Unavailable`, not `Unauthenticated`, so
clients are not logged out by it.

## Caching

- Concurrent requests with one token share a single `UserInfo` call; it is not
  cancelled when the first caller gives up.
- The session is cached for the lesser of `WithCacheTTL` (default 10s) and the
  token's own expiry. Errors are never cached.
- Only `x-webitel-access` is sent to go.webitel.app, so a cached session depends
  on the token alone.
- **Revocation delay:** a revoked token or removed permission stays effective for
  up to the cache TTL, on top of go.webitel.app's own session cache. Use
  `WithCacheTTL(0)` to disable the cache.
- The cache holds at most `WithCacheSize` sessions (default 10000). When full it
  drops expired entries and stops caching new tokens until some expire.

Options of `New`: `WithCacheTTL`, `WithCacheSize`, `WithCallTimeout` (default 5s).

## Testing

```bash
cd infra/auth
GOWORK=off go vet ./...
GOWORK=off go test -race ./...
```

Tests use a hand-written `UserInfoClient` fake and `testing/synctest` for TTL
and concurrency.
