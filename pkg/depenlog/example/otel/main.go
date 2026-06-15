// Command otel shows the OTel-pipeline mode: instead of writing JSON to stdout
// itself, pkg/depenlog hands its slog handler over to the OpenTelemetry bridge, so
// every record (including grpc-go/fx/HTTP adapter logs) flows through the OTel
// LoggerProvider and its exporter. The exporter owns the output schema — the
// kit's stdout codec already emits the canonical date/level/message/trace_id
// shape — and trace_id/span_id are attached automatically from the active span.
//
//	go run .
package main

import (
	"context"
	"os"
	"time"

	bridgeslog "github.com/webitel/webitel-go-kit/infra/otel/log/bridge/slog"
	otelsdk "github.com/webitel/webitel-go-kit/infra/otel/sdk"
	gokitlog "github.com/webitel/webitel-go-kit/pkg/depenlog"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"

	// Exporters are plugins: blank-import the ones referenced by the
	// OTEL_*_EXPORTER env vars so their init() registers them.
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/log/stdout"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/trace/stdout"
)

func main() {
	ctx := context.Background()

	// Send OTel logs and traces to stdout so the example is self-contained.
	// In production these point at an OTLP collector instead.
	_ = os.Setenv("OTEL_LOGS_EXPORTER", "stdout")
	_ = os.Setenv("OTEL_TRACES_EXPORTER", "stdout")
	_ = os.Setenv("OTEL_LOGRECORD_CODEC", "json") // canonical JSON schema

	// Describe this service. Reusing pkg/semconv keys keeps resource attributes
	// named the same way as log fields.
	res := resource.NewSchemaless(
		attribute.String(semconv.ServiceNameKey, "log-otel-example"),
		attribute.String(semconv.ServiceVersionKey, "0.0.1"),
	)

	shutdown, err := otelsdk.Configure(ctx, otelsdk.WithResource(res))
	if err != nil {
		panic(err)
	}
	defer func() { _ = shutdown(ctx) }()

	// Bridge slog -> the global OTel LoggerProvider, then hand that handler to
	// pkg/depenlog. New still installs slog default + grpc-go, so all of those now
	// flow into the OTel pipeline too.
	otelHandler := bridgeslog.New("log-otel-example").Handler()
	l, err := gokitlog.New(gokitlog.Config{}, gokitlog.WithHandler(otelHandler))
	if err != nil {
		panic(err)
	}

	// A real span: its trace_id/span_id are emitted by the exporter with no
	// manual plumbing, because we log via the *Context method.
	tr := otel.Tracer("log-otel-example")
	ctx, span := tr.Start(ctx, "handle-request")
	l.InfoContext(ctx, "handling request", semconv.UserIDKey, 42)
	span.End()

	// Give the batch processors a moment before shutdown flushes them.
	time.Sleep(200 * time.Millisecond)
}
