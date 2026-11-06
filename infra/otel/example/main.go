package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	otelsdk "github.com/webitel/webitel-go-kit/infra/otel/sdk"
	"github.com/webitel/webitel-go-kit/pkg/logging"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"

	// -------------------- plugin(s) -------------------- //
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/log/otlp"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/log/stdout"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/otlp"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/prometheus"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/stdout"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/trace/otlp"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/trace/stdout"
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
		logging.Bootstrap(logging.WithLevel(&verbose)).Slog(),
	)
}

func main() {

	ctx := context.Background()
	shutdown, err := otelsdk.Configure(ctx,
		otelsdk.WithResource(service),
		otelsdk.WithLogBridge(func() {
			slog.SetDefault(
				logging.New(
					logging.WithComponent(logging.Component(name)),
					logging.WithLevel(&verbose),
				).Slog(),
			)
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
