package tracing

import (
	"context"

	otelpyroscope "github.com/grafana/otel-profiling-go"
	"go.opentelemetry.io/otel/trace"
)

type profilingTracerProvider struct {
	trace.TracerProvider

	wrappedTp TracerProvider
}

// NewProfilingTracerProvider creates a new tracer provider that annotates pprof
// samples with span_id label. This allows to establish a relationship
// between pprof profiles and reported tracing spans.
func NewProfilingTracerProvider(tp TracerProvider) TracerProvider {
	return &profilingTracerProvider{
		TracerProvider: otelpyroscope.NewTracerProvider(tp),
		wrappedTp:      tp,
	}
}

func (tp *profilingTracerProvider) Shutdown(ctx context.Context) error {
	return tp.wrappedTp.Shutdown(ctx)
}
