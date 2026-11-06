package rabbitmq

import (
	"context"
	"maps"
	"slices"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// headerCarrier adapts amqp.Table to propagation.TextMapCarrier.
type headerCarrier amqp.Table

var _ propagation.TextMapCarrier = headerCarrier(nil)

func (c headerCarrier) Get(key string) string {
	switch v := c[key].(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}

func (c headerCarrier) Set(key, value string) {
	c[key] = value
}

func (c headerCarrier) Keys() []string {
	return slices.Collect(maps.Keys(c))
}

// InjectHeaders returns headers with propagation fields replaced by ones from ctx.
// Caller's table is never modified: a copy is returned when anything changes,
// otherwise headers are returned as is. Nil headers are allowed.
func InjectHeaders(ctx context.Context, headers amqp.Table) amqp.Table {
	prop := otel.GetTextMapPropagator()

	fields := prop.Fields()
	injected := make(propagation.MapCarrier, len(fields))
	prop.Inject(ctx, injected)

	if len(injected) == 0 && !hasAnyField(headers, fields) {
		return headers
	}

	// alloc one new map with enough space to hold all headers and injected fields
	// instead of populating injected with multiple small reallocations
	out := make(amqp.Table, len(headers)+len(injected))
	maps.Copy(out, headers)
	// clear any propagation fields from header
	for _, field := range fields {
		delete(out, field)
	}
	// inject current ctx propagation fields
	for key, value := range injected {
		out[key] = value
	}

	return out
}

func hasAnyField(headers amqp.Table, fields []string) bool {
	for _, field := range fields {
		if _, ok := headers[field]; ok {
			return true
		}
	}

	return false
}

func extractHeaders(ctx context.Context, headers amqp.Table) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, headerCarrier(headers))
}
