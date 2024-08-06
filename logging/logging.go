package logging

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/log/global"
	logsdk "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type Exporter string

const (
	DefaultExporter Exporter = ""
	NoopExporter    Exporter = "noop"
	STDOutExporter  Exporter = "stdout"
	OTLPExporter    Exporter = "otlp"
)

// LoggerProvider have compilation failure as default behavior of implementations
type LoggerProvider interface {
	Logger(name string, options ...log.LoggerOption) log.Logger
	Shutdown(ctx context.Context) error
}

type Logging struct {
	embedded.LoggerProvider

	provider LoggerProvider

	serviceName    string
	serviceVersion string

	enabled Exporter

	// TODO: split options within exporters
	address       string
	insecure      bool
	customAttribs []attribute.KeyValue
	writer        io.Writer
	timestamps    bool

	batcherMaxExportBatchSize int
	batcherExportTimeout      time.Duration
	batcherExportInterval     time.Duration
	batcherMaxQueueSize       int
}

func New(ctx context.Context, service string, opts ...Option) (*Logging, error) {
	l := Logging{
		serviceName:               service,
		enabled:                   NoopExporter,
		batcherMaxQueueSize:       2048,
		batcherExportTimeout:      30 * time.Second,
		batcherExportInterval:     1 * time.Second,
		batcherMaxExportBatchSize: 512,
	}

	for _, opt := range opts {
		opt.apply(&l)
	}

	var exp logsdk.Exporter
	var err error
	switch l.enabled {
	case NoopExporter, DefaultExporter:
		// TODO: add noop logging provider
	case STDOutExporter:
		var o []stdoutlog.Option
		if l.writer != nil {
			o = append(o, stdoutlog.WithWriter(l.writer))
		}

		if !l.timestamps {
			o = append(o, stdoutlog.WithoutTimestamps())
		}

		exp, err = stdoutlog.New(o...)
		if err != nil {
			return nil, err
		}
	case OTLPExporter:
		u, err := url.Parse(l.address)
		if err != nil {
			return nil, fmt.Errorf("address must be specified for OTLP exporter: %w", err)
		}

		// TODO: add more otlploggrpc options

		switch u.Scheme {
		case "http", "https":
			exp, err = otlploghttp.New(ctx, otlploghttp.WithEndpoint(fmt.Sprintf("%s:%s", u.Hostname(), u.Port())), otlploghttp.WithInsecure())
			if err != nil {
				return nil, err
			}
		case "grpc":
			exp, err = otlploggrpc.New(ctx, otlploggrpc.WithEndpoint(fmt.Sprintf("%s:%s", u.Hostname(), u.Port())), otlploggrpc.WithInsecure())
			if err != nil {
				return nil, err
			}
		}

	}

	if l.provider == nil {
		res, err := resource.New(context.Background(),
			resource.WithAttributes(
				semconv.ServiceNameKey.String(l.serviceName),
				semconv.ServiceVersionKey.String(l.serviceVersion),
			),
			resource.WithAttributes(l.customAttribs...),
			resource.WithProcessRuntimeDescription(),
			resource.WithTelemetrySDK(),
			resource.WithHost(),
		)
		if err != nil {
			return nil, err
		}

		var bo []logsdk.BatchProcessorOption
		bo = append(bo, logsdk.WithMaxQueueSize(l.batcherMaxQueueSize),
			logsdk.WithExportTimeout(l.batcherExportTimeout),
			logsdk.WithExportInterval(l.batcherExportInterval),
			logsdk.WithMaxQueueSize(l.batcherMaxExportBatchSize),
		)

		processor := logsdk.NewBatchProcessor(exp, bo...)
		l.provider = logsdk.NewLoggerProvider(logsdk.WithProcessor(processor), logsdk.WithResource(res))
	}

	global.SetLoggerProvider(&l)

	return &l, nil
}

func (l *Logging) Shutdown(ctx context.Context) error {
	return l.provider.Shutdown(ctx)
}

func (l *Logging) Logger(name string, options ...log.LoggerOption) log.Logger {
	return l.provider.Logger(name, options...)
}

var _ LoggerProvider = &Logging{}
