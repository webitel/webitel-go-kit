# pkg/depenlog examples

Runnable examples for the kit's unified logger (`github.com/webitel/webitel-go-kit/pkg/depenlog`).
Every example is self-terminating, so `go run` prints a few lines and exits.

The point of `pkg/depenlog` is **one record schema everywhere**: field names come from
`pkg/semconv` (`date` / `level` / `message`, `trace_id` / `span_id`, `error`, `component`,
…), `trace_id`/`span_id` are attached automatically from the active span, and third-party
logs (grpc-go, fx, HTTP, stdlib `log`) are funnelled through the same handler — so a single
Loki/ELK query works across all services.

## Examples

| Dir | Run | Shows |
|-----|-----|-------|
| `basic/` | `go run ./basic` | `New`, structured logging with canonical keys, `WithComponent`, `trace_id` correlation via `*Context`, `err`→`error` normalization, grpc-go logs joining the schema |
| `fx/`    | `go run ./fx`    | `fx.WithLogger(FxLogger(l))` — fx's lifecycle events logged as `component=fx`; injecting `logger.Logger` into constructors |
| `http/`  | `go run ./http`  | `Middleware` (per-request access log with `trace_id`) and `ErrorLog` (net/http internal errors) |
| `otel/`  | `cd otel && go run .` | OTel-pipeline mode: `WithHandler` plugs the otelslog bridge so all logs flow through the OTel `LoggerProvider`/exporter, with `trace_id`/`span_id` from real spans (separate module — see below) |

## Typical wiring in a service

```go
// Map your appconfig.Log onto depenlog.Config and build the logger once at startup.
l := depenlog.New(depenlog.Config{
    Level:   cfg.Log.Level,   // debug|info|warn|error
    JSON:    cfg.Log.JSON,
    File:    cfg.Log.File,    // rotated via lumberjack when set
    Console: cfg.Log.Console,
})
// New already did: slog.SetDefault(l) + grpc-go global logger.

// fx:
fx.WithLogger(func() fxevent.Logger { return depenlog.FxLogger(l) })

// http:
srv := &http.Server{
    Handler:  depenlog.Middleware(l)(mux),
    ErrorLog: depenlog.ErrorLog(l),
}

// scope by component:
db := depenlog.WithComponent(l, "postgres")
```

Always log with the `*Context` methods (`InfoContext(ctx, …)`) so `trace_id`/`span_id`
from the request's span land in the record.

## OTel mode (`otel/`)

`otel/` is its **own module** because it pulls the full OpenTelemetry SDK (via
`infra/otel`). Two notes:

- The SDK's exporters are **plugins** — blank-import the ones your `OTEL_*_EXPORTER`
  env vars name (`_ ".../infra/otel/sdk/log/stdout"`), or nothing is registered.
- It pins a newer `go.opentelemetry.io/contrib/bridges/otelzap` than `pkg/logger` drags
  in transitively (via `wlog`); the older one doesn't compile against the SDK's newer
  `otel/log`. This is only relevant when one module combines `pkg/depenlog` with `infra/otel`.

In OTel mode the **exporter** owns the output schema (the kit's stdout codec already emits
`date`/`level`/`message`/`trace_id`/`span_id`), so `pkg/depenlog`'s own JSON/text/ReplaceAttr
path is bypassed — that's expected.

## Note on module resolution

These examples use `replace` directives to point at the sibling modules in this repo (no
`go.work` checked in). On release those are swapped for version tags.
