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
