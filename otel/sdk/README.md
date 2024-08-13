# [O]pen[Tel]emetry Environment Configuration

Create environment configuration file
```sh
echo "export OTEL_LOG_LEVEL=trace

### stdout ; stderr ; file ; otlphttp ; otlpgrpc ###
export OTEL_LOG_EXPORT=file:///tmp/otlp.logs.jsonl
export OTEL_LOG_EXPORT=stdout

# text ; json ; otel
export OTEL_LOG_FORMAT=json

export OTEL_LOG_FORMAT_TIMESTAMP=\"2006-01-02 15:04:05.000000Z07:00\"

### stdout ; otlphttp ; otlpgrpc ###
export OTEL_TRACE_EXPORT=

### stdout ; otlphttp ; otlpgrpc ###
export OTEL_METRIC_EXPORT=

#export OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=
#export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=
#export OTEL_EXPORTER_OTLP_METRICS_ENDPOINT=

#export OTEL_EXPORTER_OTLP_INSECURE=false
#export OTEL_EXPORTER_OTLP_ENDPOINT=https://remote.collector.otlp:4317

" > .env
```
Apply environment configuration ...
```sh
source .env
```

Go simplest example
```golang
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	otelsdk "github.com/webitel/webitel-go-kit/otel/sdk"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	// -------------------- plugin(s) -------------------- //
	_ "github.com/webitel/webitel-go-kit/otel/sdk/log/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/log/stdout"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/metric/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/metric/stdout"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/trace/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/trace/stdout"
)

var (
	name    = "otel/example"
	service = resource.NewSchemaless(
		semconv.ServiceName(name),
		semconv.ServiceVersion("λ"),
		semconv.ServiceInstanceID("example"),
		semconv.ServiceNamespace("webitel"),
	)
)

func main() {

	ctx := context.Background()
	shutdown, err := otelsdk.Setup(
		ctx,
		otelsdk.WithResource(service),
		otelsdk.WithLogLevel(log.SeverityDebug),
	)
	defer shutdown(ctx)

	stdlog := otelslog.NewLogger(name)
	if err != nil {
		stdlog.ErrorContext(
			ctx, "OTel setup",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	stdlog.InfoContext(
		ctx, "OTel setup",
		slog.Bool("success", true),
	)

	// press any key to continue ...
	fmt.Scanln()

	stdlog.InfoContext(
		ctx, "Shutting down ...",
	)
}

```