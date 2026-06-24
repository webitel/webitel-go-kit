# pkg/semconv

```
import "github.com/webitel/webitel-go-kit/pkg/semconv"
```

Canonical attribute and field **names** shared across every Webitel service. It is
a tiny, dependency-free package of `const` string keys — nothing else. The point
is that producers and consumers of structured logs and telemetry agree on the same
field names, so a single Loki/ELK/Tempo query works across all services.

> **Rule of thumb:** never hand-write a log/attribute key as a string literal.
> If the field is cross-service, it belongs here. Use the constant; if it's
> missing, add it here rather than typing `"user_id"` again somewhere.

## The keys

Three groups, by where they appear in a record.

### Core log fields — `core.go`
The JSON field names of every emitted log record.

| Constant | Value |
|----------|-------|
| `TimestampKey`  | `date` |
| `LevelKey`      | `level` |
| `MessageKey`    | `message` |
| `TraceIDKey`    | `trace_id` |
| `SpanIDKey`     | `span_id` |
| `TraceFlagsKey` | `trace_flags` |

These rename slog's built-ins (`time`→`date`, `msg`→`message`) and are what
`pkg/depenlog` and the OTel stdout codec emit. You rarely set these yourself —
the logger does — but you query and parse on them.

### Application attributes — `semconv.go`
Keys you attach to records to correlate by request, identity, and origin.

| Constant | Value |
|----------|-------|
| `RequestIDKey` | `request_id` |
| `UserIDKey`    | `user_id` |
| `DomainIDKey`  | `domain_id` |
| `ComponentKey` | `component` |
| `ErrorKey`     | `error` |

### Resource attributes — `resource.go`
OTel `service.*` resource conventions, for describing the service that produces
telemetry.

| Constant | Value |
|----------|-------|
| `ServiceNameKey`       | `service.name` |
| `ServiceVersionKey`    | `service.version` |
| `ServiceInstanceIDKey` | `service.instance.id` |
| `ServiceNamespaceKey`  | `service.namespace` |

## Usage

As structured-log attribute keys:

```go
log.InfoContext(ctx, "request accepted",
    semconv.RequestIDKey, reqID,
    semconv.UserIDKey, userID,
    semconv.DomainIDKey, domainID,
)
```

As OTel resource attributes:

```go
res := resource.NewSchemaless(
    attribute.String(semconv.ServiceNameKey, "im-account-service"),
    attribute.String(semconv.ServiceVersionKey, version),
)
```

To parse/query the canonical record fields downstream:

```go
ts := record[semconv.TimestampKey]   // "date"
msg := record[semconv.MessageKey]    // "message"
tid := record[semconv.TraceIDKey]    // "trace_id"
```

## Notes

- **Logger-managed vs. caller-managed.** The core fields (`core.go`) and
  `component` / `error` are written for you by `pkg/depenlog`. The request /
  identity attributes are yours to attach. See [pkg/depenlog](../depenlog) for
  the logger that emits this schema and auto-attaches `trace_id`/`span_id`.
- **`error`, not `err`.** The canonical key is `ErrorKey` (`error`). `pkg/depenlog`
  normalizes the common `"err"` misspelling to it, but prefer `semconv.ErrorKey`
  (or the literal `"error"`) when logging errors directly.
- **Adding a key.** Put it in the file matching its group (core field / app
  attribute / resource attribute), give it a `…Key` name and a doc comment, and
  use the constant everywhere. One name, one place.
