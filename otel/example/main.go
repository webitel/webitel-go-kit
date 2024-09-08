package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	otelsdk "github.com/webitel/webitel-go-kit/otel/sdk"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	// -------------------- bridge(s) -------------------- //

	slogutil "github.com/webitel/webitel-go-kit/otel/log/bridge/slog"
	"go.opentelemetry.io/contrib/bridges/otelslog"

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

	verbose slog.LevelVar
)

func init() {

	// default: "info"
	verbose.Set(slog.LevelInfo)

	if input := os.Getenv("LOG_LEVEL"); input != "" {
		_ = verbose.UnmarshalText([]byte(input))
	}

	slog.SetDefault(
		slog.New(slog.NewTextHandler(
			os.Stdout, &slog.HandlerOptions{
				Level: &verbose,
			},
		)),
	)
}

func main() {

	ctx := context.Background()
	shutdown, err := otelsdk.Configure(ctx,
		otelsdk.WithResource(service),
		otelsdk.WithLogBridge(func() {
			// Just for example ...
			// Redirect slog.Default().Handler() to
			// otel/log/global.LoggerProvider with slog.Level filter
			stdlog := slog.New(
				slogutil.WithLevel(
					// front: otelslog.Handler level filter
					&verbose,
					// back: otel/log/global.Logger("slog")
					otelslog.NewHandler("slog"),
				),
			)

			slog.SetDefault(stdlog)

		}),
	)

	log := slog.Default()

	if err != nil {
		log.ErrorContext(ctx,
			"OpenTelemetry configuration",
			"error", err,
		)
		os.Exit(1)
	}

	defer shutdown(ctx)

	log.InfoContext(ctx,
		"OpenTelemetry configuration",
		slog.Bool("success", true),
	)

	log.DebugContext(ctx,
		"Details of fake event",
		"event", "fake",
		"fake", true,
	)

	// press any key to continue ...
	fmt.Scanln()

	log.InfoContext(
		ctx, "Shutting down ...",
	)
}
