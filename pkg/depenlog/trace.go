package depenlog

import (
	"context"
	"log/slog"

	"github.com/webitel/webitel-go-kit/pkg/semconv"
	"go.opentelemetry.io/otel/trace"
)

// traceHandler decorates a slog.Handler, copying the active span's trace_id and
// span_id from the context into every record. This is what makes log lines
// correlatable with traces — and across services — without callers passing the
// IDs by hand, provided they log through the *Context methods (so a real
// context, not context.Background(), reaches Handle).
type traceHandler struct {
	slog.Handler
}

func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String(semconv.TraceIDKey, sc.TraceID().String()),
			slog.String(semconv.SpanIDKey, sc.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h traceHandler) WithGroup(name string) slog.Handler {
	return traceHandler{Handler: h.Handler.WithGroup(name)}
}
