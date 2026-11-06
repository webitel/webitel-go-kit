# auth

Authentication and authorization of gRPC calls for Webitel services.

The caller's access token (`x-webitel-access`) is exchanged for a `Session`
through `Auth.UserInfo` of go.webitel.app. Two parts, both shaped like
go-grpc-middleware's `AuthFunc` (`func(ctx) (context.Context, error)`):

- `Client`: `Session(ctx, token)` and `Authenticate(ctx)`. It takes the token from
  the metadata, requires a live session and puts it into the context. It knows
  nothing about `WebitelAPI`.
- `Authorizer`: `NewAuthorizer(client, pb.WebitelAPI)` and `Authorize(ctx)`.
  On top of `Authenticate` it checks the call against the `WebitelAPI` map that
  `protoc-gen-go-webitel` generates. `WebitelAPI` is passed as `any` and
  converted by field name, so `NewAuthorizer` fails at once if its shape changes.

Order of checks in `Authorize`:

1. the method (`grpc.Method(ctx)`) must be in the map, otherwise it is denied
   (`PermissionDenied`), also outside a server context;
2. the token must be present and the session not expired (`Authenticate`);
3. every license in the service's `additional_license` must be valid now
   (`IssuedAt <= now < ExpiresAt`, matched by `License.Prod`);
4. obac: the user needs the action on the object class.

## Usage

Interceptors live in `pkg/interceptors`; this package does not import them.

```go
import authgrpc "buf.build/gen/go/webitel/webitel-go/grpc/go/_gogrpc"

client := auth.New(authgrpc.NewAuthClient(conn)) // conn: gRPC connection to go.webitel.app
authorizer, err := auth.NewAuthorizer(client, auditpb.WebitelAPI)
if err != nil {
    return err
}

srv := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        interceptors.UnaryServerErrorInterceptor(),
        interceptors.AuthUnaryServerInterceptor(authorizer.Authorize),
    ),
    grpc.ChainStreamInterceptor(interceptors.AuthStreamServerInterceptor(authorizer.Authorize)),
)
```

A service without `WebitelAPI` passes `client.Authenticate` instead of
`authorizer.Authorize`.

Public methods, such as health checks, are the service's job: leave them out
with the `selector` of go-grpc-middleware. Reflection stays closed unless the
service opts in the same way. Service-internal APIs with their own credentials
are handled the same way.

`UnaryServerErrorInterceptor` maps the errors below to gRPC codes and rewrites
the IDs to `api.process.*` for clients. For streams the error format comes from
the server (WTEL-10860); nothing is formatted here.

In a handler:

```go
s, ok := auth.SessionFromContext(ctx)
if !ok {
    // only on methods the interceptor skipped
    return nil, errors.Unauthenticated("no session")
}
if s.RequiresRBAC("forms", auth.ActionRead) {
    // restrict records to those reachable by s.ACLRoles()
}
```

Record-based access control (rbac) is not applied here; that is the handler's
job through `Session.RequiresRBAC` and `Session.ACLRoles`.
In tests build the handler context with `auth.ContextWithSession`.

A stream is authorized once, when it opens. A token that is revoked or expires
later does not close a stream that is already running.

## Access and permissions

| `access` in proto | `Action` | Flag in `Scope.Access` | Super permission |
|---|---|---|---|
| create (also when unset) | `ActionCreate` | `x` | `add` |
| read | `ActionRead` | `r` | `read` |
| update | `ActionUpdate` | `w` | `write` |
| delete | `ActionDelete` | `d` | `delete` |

A method without an `access` option is generated as `ActionCreate`, so it needs `x`.
A class missing from the user's scope (go.webitel.app omits classes without
read) is denied. With `OBAC` off for the class the action is allowed; with it
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
| missing license (all missing are in the `licenses` value) | `PermissionDenied` | `auth.license` |
| obac denied | `PermissionDenied` | `auth.permission` |

Each row is an exported `Err*` variable, so callers check with `errors.Is`;
`ErrLicense` and `ErrPermission` carry `ValueLicenses` and `ValueClass`, read
with `errors.Value`.

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
