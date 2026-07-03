# httpproxy

Hot-reloadable forward-proxy settings (`http_proxy` / `https_proxy` / `no_proxy`)
for outbound HTTP clients. Lets a service pick up proxy changes at runtime —
no restart — which the standard environment variables cannot do (a process
environment is fixed at startup).

## How it works

`Manager` keeps the current settings in an atomic snapshot and exposes
`Manager.Proxy`, a function assignable to `http.Transport.Proxy`. The
transport consults it **on every request**, so long-lived clients follow
settings changes without being rebuilt. After every applied change the
manager calls `CloseIdleConnections()` on all registered transports, so
pooled keep-alive connections do not keep using the previous route.

Settings come from a small watched YAML file, with the process environment
as the fallback. The watch is on the parent directory and any event there
triggers a debounced reload (unchanged settings are a no-op), so
rename-replace updates and even
replacement of the directory itself are picked up; watcher setup failures
are logged, never silent.

## Quick start

```go
mgr := httpproxy.NewManager(httpproxy.WithLogger(log))

// Make http.Get, http.DefaultClient and SDKs on the default transport dynamic.
if err := mgr.HookDefaultTransport(); err != nil {
    return err
}

// Apply and watch the settings file; empty path keeps environment settings.
go mgr.WatchFile(ctx, os.Getenv("PROXY_CONFIG_FILE"))
```

Custom transports:

```go
shared := mgr.Transport()                 // clone of http.DefaultTransport; create once and reuse
client := mgr.Client(15 * time.Second)    // safe to call per request: one shared transport inside
custom := mgr.WrapTransport(&http.Transport{TLSClientConfig: tlsCfg})
```

`Transport()`/`WrapTransport()` register the transport for the manager's
lifetime — create them once and share. Short-lived per-request transports
should set `Proxy: mgr.Proxy` directly and must **not** be registered.

## Settings file

YAML (JSON is accepted too, being a YAML subset):

```yaml
http_proxy: "http://user:pass@10.0.1.1:3128"
https_proxy: "http://10.0.1.1:3128"
no_proxy: "localhost,127.0.0.1,.svc,10.0.0.0/8"
```

| State | Effect |
|---|---|
| Key absent | Falls back to the environment variable (`HTTP_PROXY`, …) |
| Key set to `""` | Explicitly no proxy for that class of requests |
| File missing / deleted | Plain environment behavior |
| Misspelled key, malformed file, URL without host, bad `no_proxy` CIDR/IP | Logged (credentials redacted); last good settings stay in effect |

`no_proxy` matching follows `golang.org/x/net/http/httpproxy`: host names,
domain suffixes (`.example.com`), IPs and CIDRs; `*` disables proxying.
Requests to `localhost` and loopback addresses are always direct.

## Caveats

- `HookDefaultTransport` must run early in `main()`, before any outbound
  request and before any code clones `http.DefaultTransport`.
- Changing the settings re-routes **new** requests; in-flight requests finish
  on the connection they started with.
- HTTP CONNECT proxying applies to plain-HTTP and TLS (including HTTP/2 via
  CONNECT). Raw-TCP protocols (gRPC dialers, MTProto) are out of scope.
- Services using `appconfig` can wire the manager to their config reload by
  calling `mgr.Update(cfg)` from `Loader.Watch`; a canonical `appconfig.Proxy`
  section can be added once such a service needs outbound HTTP.
