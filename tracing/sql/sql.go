package sql

import (
	"context"
	"database/sql/driver"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/webitel/webitel-go-kit/semconv"
)

const scopeName = "github.com/webitel/webitel-go-kit/tracing/sql"

type Tracer struct {
	tracerProvider trace.TracerProvider
	tracer         trace.Tracer

	formatSpanName spanNameFormatter
	errorToStatus  errorToSpanStatus
	queryTracer    queryTracer

	attributes []attribute.KeyValue
}

func NewTracer(opts ...Option) *Tracer {
	t := &Tracer{
		tracerProvider: otel.GetTracerProvider(),
		formatSpanName: formatSpanName,
		errorToStatus:  spanStatusFromError,
		queryTracer:    traceNoQuery,
		attributes: []attribute.KeyValue{
			semconv.DBSystemPostgreSQL,
		},
	}

	for _, o := range opts {
		o.apply(t)
	}

	t.tracer = t.tracerProvider.Tracer(scopeName)

	return t
}

func (t *Tracer) ShouldTrace(ctx context.Context) bool {
	hasSpan := trace.SpanContextFromContext(ctx).IsValid()

	return hasSpan
}

func (t *Tracer) StartTrace(ctx context.Context, method string, query string, args []driver.NamedValue) (context.Context, func(err error)) {
	hasParentSpan := t.ShouldTrace(ctx)
	newCtx, span := t.tracer.Start(ctx, t.formatSpanName(ctx, method, query), trace.WithSpanKind(trace.SpanKindClient))
	if !hasParentSpan {
		ctx = newCtx
	}

	attrs := make([]attribute.KeyValue, 0)
	attrs = append(attrs, t.attributes...)
	attrs = append(attrs, t.queryTracer(ctx, query, args)...)

	return ctx, func(err error) {
		code, desc := t.errorToStatus(err)
		span.SetAttributes(attrs...)
		span.SetStatus(code, desc)
		if code == codes.Error {
			span.RecordError(err)
		}

		span.End()
	}
}
