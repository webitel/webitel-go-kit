package tracing_test

import (
	"context"

	"github.com/webitel/wlog"
	"go.opentelemetry.io/otel/codes"

	"github.com/webitel/webitel-go-kit/tracing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func ExampleTracer() {
	ctx := context.Background()
	log := wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: true})
	tp, err := tracing.New(log,
		tracing.WithServiceName("my-service"),
		tracing.WithServiceVersion("1.2.3"),
		tracing.WithAttributes(attribute.String("key1", "value1")),
		tracing.WithExporter("stdout"),
		// tracing.WithExporter("otel"),
		// tracing.WithAddress("127.0.0.1:4317"),
	)
	if err != nil {
		log.Error("create new tracer", wlog.Err(err))

		return
	}

	defer tp.Shutdown(ctx)

	_, span := tp.Start(ctx, "my-span", trace.WithAttributes(attribute.String("key2", "value2")))
	span.SetStatus(codes.Ok, "foo/bar/desc")
	defer span.End()
}
