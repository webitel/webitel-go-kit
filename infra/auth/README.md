# auth

Authentication and authorization of gRPC calls for Webitel services.

The access token (`x-webitel-access`) is exchanged for a `Session` through
`Auth.UserInfo` of go.webitel.app. Both entry points have the shape of
go-grpc-middleware's `AuthFunc` and plug into `pkg/interceptors`:

- `Client.Authenticate`: puts the session into the context. For services without `WebitelAPI`.
- `Authorizer.Authorize`: also checks licenses and obac against the generated `WebitelAPI`.

## Usage

```go
import authgrpc "buf.build/gen/go/webitel/webitel-go/grpc/go/_gogrpc"

client := auth.New(authgrpc.NewAuthClient(conn)) // conn: gRPC connection to go.webitel.app
defer client.Close()

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

Public methods (health checks) and reflection are the service's choice: leave them out
with the `selector` of go-grpc-middleware. A method without an `access` option
in the proto needs the create permission.

In a handler:

```go
s, _ := auth.SessionFromContext(ctx)
if !s.HasPermission("some_permission") { /* ... */ }
if auth.RequiresRBAC(ctx) {
    // restrict records to s.ACLRoles()
}
```

`RequiresRBAC(ctx)` answers for the method being served and returns `true` when there
is no rule in the context. For a parent object use `s.RequiresRBAC(class, action)`.

## Errors

Exported `Err*` variables (see `errors.go`) with a gRPC code and an ID each. Check with
`errors.Is`; `ErrLicense` and `ErrPermission` carry `ValueLicenses` and `ValueClass`.

## Caching

A session is cached for `WithCacheTTL` (default 10s, `0` disables the cache), up to
`WithCacheSize` entries. A revoked token stays valid for up to the TTL. Call `Close` when done.
