# pkg/depenlog

```
import depenlog "github.com/webitel/webitel-go-kit/pkg/depenlog"
```

The kit's **unified logger**: one `slog`-based handler that

- emits **one record schema** for every service (field names from
  [`pkg/semconv`](../semconv) — `date` / `level` / `message`, `trace_id` / `span_id`,
  `error`, `component`, …);
- **auto-attaches** the active span's `trace_id` / `span_id` from `context`;
- **funnels third-party logs** — grpc-go, fx, `net/http`, the stdlib `log` package —
  through that same handler.

The result: a single `trace_id` query in Loki/ELK/Tempo joins log lines across
every service and every framework.

> **The one rule:** log through the `*Context` methods (`InfoContext(ctx, …)`),
> passing a real request `ctx`. That's how `trace_id`/`span_id` reach the record.
> `Info(…)` (no ctx) works but cannot correlate.

## Build it once, at startup

`New` builds the logger from `Config` and installs it process-wide
(`slog.SetDefault`, so `slog.*` and the stdlib `log` package share it; plus
grpc-go's global logger). The returned `logger.Logger` is the handle you inject
everywhere else.

```go
l := depenlog.New(depenlog.Config{
    Level:   cfg.Log.Level,   // "debug" | "info" | "warn" | "error"  (default info)
    JSON:    cfg.Log.JSON,    // JSON when true, human-readable text otherwise
    Console: cfg.Log.Console, // write to stdout
    File:    cfg.Log.File,    // also write to this path (rotated), optional

    // File rotation (only when File != ""; zero = lumberjack defaults):
    MaxSizeMB:  100,   // rotate after N MB
    MaxBackups: 7,     // keep N rotated files (0 = keep all)
    MaxAgeDays: 30,    // retain for N days (0 = forever)
    Compress:   true,  // gzip rotated files
})
```

`Config` deliberately mirrors `appconfig.Log` without importing it — map your
app's config fields onto it. If neither `Console` nor `File` is set, output falls
back to stdout (logs are never silently dropped).

### `logger.Logger` — the logging surface

```go
type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
    Debug(msg string, args ...any)
    Warn(msg string, args ...any)

    InfoContext(ctx context.Context, msg string, args ...any)   // use these
    ErrorContext(ctx context.Context, msg string, args ...any)
    DebugContext(ctx context.Context, msg string, args ...any)
    WarnContext(ctx context.Context, msg string, args ...any)

    With(args ...any) Logger   // child logger that tags every record
}
```

```go
l.InfoContext(ctx, "user authenticated", semconv.UserIDKey, uid)
l.ErrorContext(ctx, "fetch failed", semconv.ErrorKey, err)
```

## Scope a sub-logger by component

`WithComponent` tags every record from a sub-logger with `semconv.ComponentKey`,
so you can filter logs to one subsystem.

```go
db := depenlog.WithComponent(l, "postgres")
db.InfoContext(ctx, "connected", "host", host)   // adds component=postgres
```

## Wire in the frameworks

`New` already routes `slog`, stdlib `log`, and grpc-go. The rest are explicit
because they're per-app:

```go
// fx — its provide/invoke/lifecycle events become component=fx records:
app := fx.New(
    fx.Provide(func() logger.Logger { return l }),     // inject the kit logger
    fx.WithLogger(func() fxevent.Logger { return depenlog.FxLogger(l) }),
    // ...
)

// net/http — access log + internal server errors, both component=http:
srv := &http.Server{
    Handler:  depenlog.Middleware(l)(mux),  // logs method/path/status/duration_ms
    ErrorLog: depenlog.ErrorLog(l),         // net/http's own errors → error level
}

// grpc-go is wired by New automatically; call UseGRPC(l) only to rebind it.
```

`Middleware` uses the request context, so its access-log lines carry
`trace_id`/`span_id` when a span is active. grpc-go's global logger is
context-free, so its framework lines carry no `trace_id` — per-RPC correlation
comes from your server interceptors, which do have a context.

## OTel-pipeline mode

To hand logging to an OpenTelemetry `LoggerProvider`/exporter instead of writing
JSON to stdout yourself, replace the base handler with `WithHandler`. Every
adapter log (grpc-go/fx/http) then flows into the OTel pipeline too.

```go
otelHandler := bridgeslog.New("my-service").Handler()
l := depenlog.New(depenlog.Config{}, depenlog.WithHandler(otelHandler))
```

When `WithHandler` is set, `Config`'s `JSON`/`File`/`Console` fields and the
built-in trace/semconv decorators are **bypassed** — the exporter owns the
output schema (the kit's stdout codec already emits the canonical
`date`/`level`/`message`/`trace_id` shape). This is expected.

## API at a glance

| Symbol | Purpose |
|--------|---------|
| `New(cfg, opts...) logger.Logger` | Build the logger; install it process-wide. |
| `Config` | Level / JSON / Console / File + rotation knobs. |
| `WithHandler(h slog.Handler) Option` | Route through a custom handler (e.g. OTel bridge). |
| `WithComponent(l, name) logger.Logger` | Sub-logger tagged `component=name`. |
| `FxLogger(l) fxevent.Logger` | Adapter for `fx.WithLogger`. |
| `Middleware(l) func(http.Handler) http.Handler` | HTTP access-log middleware. |
| `ErrorLog(l) *log.Logger` | For `http.Server.ErrorLog`. |
| `UseGRPC(l)` | (Re)bind grpc-go's global logger; `New` calls this for you. |

## Examples

Runnable, self-terminating examples live in [`example/`](./example):

| Run | Shows |
|-----|-------|
| `go run ./example/basic` | `New`, canonical keys, `WithComponent`, `trace_id` via `*Context`, `err`→`error`, grpc-go logs |
| `go run ./example/fx`    | `fx.WithLogger(FxLogger(l))`; injecting `logger.Logger` |
| `go run ./example/http`  | `Middleware` + `ErrorLog` |
| `cd example/otel && go run .` | OTel-pipeline mode via `WithHandler` (separate module) |

See [`example/README.md`](./example/README.md) for details and the OTel-module caveats.
