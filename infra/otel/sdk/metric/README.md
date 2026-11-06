# metric

Builds a `go.opentelemetry.io/otel/sdk/metric` provider from a DSN.
Blank-import the exporters you ship, pick one with `OTEL_METRICS_EXPORTER`.

```go
import (
    otelsdk "github.com/webitel/webitel-go-kit/infra/otel/sdk"
    _ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/prometheus"
)

shutdown, err := otelsdk.Configure(ctx)
defer shutdown(ctx) // usable even when err != nil
```

```sh
OTEL_METRICS_EXPORTER=prometheus ./service
```

| DSN | Import | Configured by |
|---|---|---|
| `stdout`, `stderr`, `file:/path;max-size=100;max-age=30;backups=3;localtime=true;compress=false` | `metric/stdout` | the DSN |
| `otlphttp`, `otlpgrpc` | `metric/otlp` | `OTEL_EXPORTER_OTLP_*`; TLS unless the endpoint is `http://` |
| `prometheus` | `metric/prometheus` | `OTEL_EXPORTER_PROMETHEUS_HOST`/`_PORT`, default `localhost:9464` |

Unset means no metrics; an unknown scheme fails `Configure`. Push exporters
export every `OTEL_METRIC_EXPORT_INTERVAL` ms (60000).
`OTEL_METRICS_RUNTIME=true` adds the Go runtime metrics.

To add an exporter, call `metric.Register("scheme", Options)` from `init()`.

All variables: [`../README.md`](../README.md).
