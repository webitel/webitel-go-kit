package tracing

import (
	"context"
	"fmt"
	"strings"

	"github.com/webitel/wlog"
	jaegerpropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/contrib/samplers/jaegerremote"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	jaegerExporter string = "jaeger"
	otlpExporter   string = "otlp"
	stdoutExporter string = "stdout"

	jaegerPropagator string = "jaeger"
	w3cPropagator    string = "w3c"

	constSampler         string = "const"
	probabilisticSampler string = "probabilistic"
	rateLimitingSampler  string = "rateLimiting"
	remoteSampler        string = "remote"
)

type config struct {
	enabled       string
	address       string
	propagation   string
	customAttribs []attribute.KeyValue

	sampler          string
	samplerParam     float64
	samplerRemoteURL string

	serviceName    string
	serviceVersion string

	profilingIntegration bool
}

type Tracing struct {
	trace.Tracer

	log *wlog.Logger
	cfg *config

	tracerProvider tracerProvider
}

type tracerProvider interface {
	trace.TracerProvider

	Shutdown(ctx context.Context) error
}

// Tracer defines the service used to create new spans.
type Tracer interface {
	trace.Tracer

	// Inject adds identifying information for the span to the
	// metadata defined in map.
	//
	// Implementation quirk: Where OpenTelemetry is used, the [Span] is
	// picked up from [context.Context] and for OpenTracing the
	// information passed as [Span] is preferred.
	// Both the context and span must be derived from the same call to
	// [Tracer.Start].
	Inject(context.Context, trace.Span)
}

func New(log *wlog.Logger, opts ...Option) (*Tracing, error) {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	t := &Tracing{
		log: log,
		cfg: &cfg,
	}

	var exp tracesdk.SpanExporter
	var err error
	switch t.cfg.enabled {
	case stdoutExporter:
		if exp, err = stdout.New(); err != nil {
			return nil, err
		}
	case otlpExporter, jaegerExporter:
		client := otlptracegrpc.NewClient(otlptracegrpc.WithEndpoint(t.cfg.address), otlptracegrpc.WithInsecure())
		if exp, err = otlptrace.New(context.Background(), client); err != nil {
			return nil, err
		}
	default:
		log.Warn("unsupported opentelemetry exporter, using noop provider")
	}

	if exp != nil {
		if t.tracerProvider, err = t.initTracerProvider(exp); err != nil {
			return nil, err
		}
	}

	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it
	// only if tracing is enabled
	if t.cfg.enabled != "" {
		otel.SetTracerProvider(t.tracerProvider)
	}

	t.initPropagators()
	t.Tracer = otel.GetTracerProvider().Tracer("component-main")

	return t, nil
}

func (t *Tracing) Inject(ctx context.Context, span trace.Span) {
	// TODO implement me
	panic("implement me")
}

func (t *Tracing) Shutdown(ctx context.Context) error {
	return t.tracerProvider.Shutdown(ctx)
}

func (t *Tracing) initPropagators() {
	var propagators []propagation.TextMapPropagator
	for _, p := range strings.Split(t.cfg.propagation, ",") {
		switch p {
		case w3cPropagator:
			propagators = append(propagators, propagation.TraceContext{}, propagation.Baggage{})
		case jaegerPropagator:
			propagators = append(propagators, jaegerpropagator.Jaeger{})
		case "":
		default:
			propagators = append(propagators, propagation.TraceContext{}, propagation.Baggage{})
			t.log.Warn("unsupported opentelemetry propagator, using default", wlog.Any("propagator", p))
		}
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagators...))
}

func (t *Tracing) initSampler() (tracesdk.Sampler, error) {
	switch t.cfg.sampler {
	case constSampler, "":
		if t.cfg.samplerParam >= 1 {
			return tracesdk.AlwaysSample(), nil
		} else if t.cfg.samplerParam <= 0 {
			return tracesdk.NeverSample(), nil
		}

		return nil, fmt.Errorf("invalid param for const sampler - must be 0 or 1: %f", t.cfg.samplerParam)
	case probabilisticSampler:
		return tracesdk.TraceIDRatioBased(t.cfg.samplerParam), nil
	case rateLimitingSampler:
		return newRateLimiter(t.cfg.samplerParam), nil
	case remoteSampler:
		return jaegerremote.New(t.cfg.serviceName,
			jaegerremote.WithSamplingServerURL(t.cfg.samplerRemoteURL),
			jaegerremote.WithInitialSampler(tracesdk.TraceIDRatioBased(t.cfg.samplerParam)),
		), nil
	default:
		return nil, fmt.Errorf("invalid sampler type: %s", t.cfg.sampler)
	}
}

func (t *Tracing) initTracerProvider(exp tracesdk.SpanExporter) (*tracesdk.TracerProvider, error) {
	sampler, err := t.initSampler()
	if err != nil {
		return nil, err
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(t.cfg.serviceName),
			semconv.ServiceVersionKey.String(t.cfg.serviceVersion),
		),
		resource.WithAttributes(t.cfg.customAttribs...),
		resource.WithFromEnv(),
		resource.WithProcessRuntimeDescription(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, err
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithSampler(tracesdk.ParentBased(sampler)),
		tracesdk.WithResource(res),
	)

	return tp, nil
}

func TraceIDFromContext(ctx context.Context, requireSampled bool) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.HasTraceID() || !spanCtx.IsValid() || (requireSampled && !spanCtx.IsSampled()) {
		return ""
	}

	return spanCtx.TraceID().String()
}

func splitCustomAttribs(s string) ([]attribute.KeyValue, error) {
	var res []attribute.KeyValue

	attribs := strings.Split(s, ",")
	for _, v := range attribs {
		parts := strings.SplitN(v, ":", 2)
		if len(parts) > 1 {
			res = append(res, attribute.String(parts[0], parts[1]))
		} else if v != "" {
			return nil, fmt.Errorf("custom attribute malformed - must be in 'key:value' form: %q", v)
		}
	}

	return res, nil
}

var _ Tracer = &Tracing{}
