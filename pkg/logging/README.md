# pkg/logging

Structured logger for Webitel services on the OpenTelemetry Logs API. It depends only on the OpenTelemetry API modules (`otel`, `otel/log`, `contrib/bridges/otelslog`); the SDK and exporters are set up by `infra/otel/sdk`.

Requires Go 1.26 or newer (OpenTelemetry v1.47.0).

## Usage

```go
log := logging.New(logging.WithComponent("engine.dialer"))

log.InfoCtx(ctx, "user authenticated",
	logging.Str("user.id", id),
	logging.Str("phone", u.Phone).PII(),
	logging.Str("api_token", tok).Secret(),
)

log.ErrorCtx(ctx, "relay stalled",
	logging.Ev(relayFailed),
	logging.Err(err),
)
```

- Only context methods exist. Pass the request `ctx`: the SDK takes `trace_id`/`span_id` from it, and baggage processors read it.
- Arguments are typed `Field` values. `Any`, `Map` and `Attr` are the explicit escapes for untyped values; they resolve `slog.LogValuer` at any depth.
- `Ev(e)` (here `relayFailed` is any `logging.Event`) makes the record an OpenTelemetry event (`EventName` set). `Emit(ctx, e)` is `InfoCtx(ctx, "", Ev(e))`. Event types come from the generated semantic conventions packages, or implement `Event` by hand.
- `Err(err)` records the group `error` (`type`, `message`, values from `LogValue()` along the chain, `chain`) and sets `Record.Err`.
- `With` fields are rendered for every record, so sensitivity follows the current `redact` policy.
- `Component` is the instrumentation scope name. Use constants only: `<service>.<subsystem>` for application code, the framework name for adapters. Default scope: the module import path.
- The level filter is `LevelHandler` with `WithLevel` (use `*slog.LevelVar` to change it at run time). `OTEL_LOG_LEVEL` does not filter application records.

## Startup

`New` uses `otel.GetLoggerProvider()` by default. A logger created before `infra/otel/sdk.Configure` starts emitting once the provider is installed, but records written before that are dropped. For the earliest phase use `logging.Bootstrap()`: text on stderr, events degrade to an `event.name` attribute.

## Sensitive data

Package `redact` holds the classes and the process policy:

```go
redact.SetPolicy(redact.Policy{Salt: salt})
```

| Class | Recorded as | Without policy |
| --- | --- | --- |
| `.PII()` | `pii:` + 32 hex chars (HMAC-SHA256, 16 bytes) | dropped |
| `.Secret()` | `set` / `empty`, or dropped with `SecretMode: redact.Drop` | `set` / `empty` |

Without a salt, PII values are dropped and one warning goes to stderr. Pseudonyms are still personal data under GDPR. Domain types can implement `LogValue()` with `redact.Map`/`redact.Value` and `String()` with `redact.Render`. `RedactValue` renders an `attribute.Value` with the same rules as `.PII()`/`.Secret()` fields, for code that sees attributes outside the facade. `redact.Rendered` guesses from its form whether a string is already rendered for a class, so such code does not render a value twice.

## Compatibility surface

`Slog()` returns a `*slog.Logger` over the same handler for third-party code and `slog.SetDefault`. Its `...any` arguments go through otelslog reflection and are not redacted by this package.

## Testing

`logtest.NewRecorder()` is a `log.LoggerProvider` that keeps records in memory:

```go
rec := logtest.NewRecorder()
l := logging.New(logging.WithProvider(rec))
l.InfoCtx(ctx, "hello", logging.Int("n", 1))
r := rec.Records()[0] // r.Severity(), r.EventName(), r.Attributes(), r.Scope.Name
```

## Value conversion

Both emit paths convert fields with the same rules: `time.Time` becomes Unix nanoseconds, `time.Duration` nanoseconds, `uint64` above `math.MaxInt64` a decimal string; maps become attribute maps; `[]string`, `[]bool`, `[]int`, `[]int64`, `[]float64` keep their typed slice kind. `AttrToKeyValue` and `KeyValueToAttr` expose the same conversion.
