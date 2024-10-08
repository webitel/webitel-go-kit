package tracing

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

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
	"go.opentelemetry.io/otel/trace/embedded"

	"github.com/webitel/webitel-go-kit/logging/wlog"
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
	serviceName    string
	serviceVersion string

	enabled       string
	address       string
	propagation   string
	customAttribs []attribute.KeyValue
	writer        io.Writer

	sampler          string
	samplerParam     float64
	samplerRemoteURL string

	batcherMaxQueueSize       int
	batcherWithBlocking       bool
	batcherBatchTimeout       time.Duration
	batcherExportTimeout      time.Duration
	batcherMaxExportBatchSize int

	profilingIntegration bool
}

type Tracing struct {
	embedded.TracerProvider

	tracerProvider TracerProvider

	log *wlog.Logger
	cfg *config
}

type TracerProvider interface {
	embedded.TracerProvider

	Tracer(name string, options ...trace.TracerOption) trace.Tracer
	Shutdown(ctx context.Context) error
}

func New(log *wlog.Logger, service string, opts ...Option) (*Tracing, error) {
	cfg := config{
		serviceName:               service,
		serviceVersion:            "0.0.0",
		writer:                    os.Stdout,
		batcherBatchTimeout:       5 * time.Second,
		batcherMaxQueueSize:       2048,
		batcherExportTimeout:      30 * time.Second,
		batcherMaxExportBatchSize: 512,
	}

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
		if exp, err = stdout.New(stdout.WithWriter(t.cfg.writer)); err != nil {
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
		otel.SetTracerProvider(t)
	}

	t.initPropagators()

	return t, nil
}

func (t *Tracing) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	return t.tracerProvider.Tracer(name, options...)
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
	case "":
		t.log.Warn("default sampler samples every trace: a new trace will be started and exported for every request")

		return tracesdk.AlwaysSample(), nil
	case constSampler:
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
		resource.WithProcessRuntimeDescription(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, err
	}

	var bo []tracesdk.BatchSpanProcessorOption
	bo = append(bo, tracesdk.WithMaxQueueSize(t.cfg.batcherMaxQueueSize),
		tracesdk.WithBatchTimeout(t.cfg.batcherBatchTimeout),
		tracesdk.WithExportTimeout(t.cfg.batcherExportTimeout),
		tracesdk.WithMaxExportBatchSize(t.cfg.batcherMaxExportBatchSize),
	)

	if t.cfg.batcherWithBlocking {
		bo = append(bo, tracesdk.WithBlocking())
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp, bo...),
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

var _ trace.TracerProvider = &Tracing{}
