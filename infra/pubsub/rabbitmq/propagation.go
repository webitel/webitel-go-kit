package rabbitmq

import (
	"context"
	"maps"
	"slices"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// HeaderCarrier adapts amqp.Table to propagation.TextMapCarrier.
type HeaderCarrier amqp.Table

var _ propagation.TextMapCarrier = HeaderCarrier(nil)

func (c HeaderCarrier) Get(key string) string {
	switch v := c[key].(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}

func (c HeaderCarrier) Set(key, value string) {
	c[key] = value
}

func (c HeaderCarrier) Keys() []string {
	return slices.Collect(maps.Keys(c))
}

// publishParent returns parent context for publish span.
// When headers carry any propagation field, they fully replace trace and baggage of ctx,
// so captured context (e.g. outbox) never mixes with current one.
func publishParent(ctx context.Context, headers amqp.Table) context.Context {
	prop := otel.GetTextMapPropagator()

	for _, field := range prop.Fields() {
		if _, ok := headers[field]; ok {
			ctx = trace.ContextWithSpanContext(ctx, trace.SpanContext{})
			ctx = baggage.ContextWithoutBaggage(ctx)

			return prop.Extract(ctx, HeaderCarrier(headers))
		}
	}

	return ctx
}

// injectHeaders returns header map copy where propagation fields are replaced by ctx ones.
// Copy is important to avoid modifying caller's table, which may be shared.
// Such changes will cause map data races.
func injectHeaders(ctx context.Context, headers amqp.Table) amqp.Table {
	prop := otel.GetTextMapPropagator()

	fields := prop.Fields()
	if len(fields) == 0 {
		return headers
	}

	out := make(amqp.Table, len(headers)+len(fields))
	maps.Copy(out, headers)
	// clear any propagation fields from header
	for _, field := range fields {
		delete(out, field)
	}
	// inject current ctx propagation fields
	prop.Inject(ctx, HeaderCarrier(out))

	return out
}

func extractHeaders(ctx context.Context, headers amqp.Table) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, HeaderCarrier(headers))
}
