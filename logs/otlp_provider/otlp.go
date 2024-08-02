package otlp_provider

import (
	"context"
	"github.com/webitel/webitel-go-kit/logs/model"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"time"
)

type OtlpProvider struct {
	config   *config
	provider log.LoggerProvider
	logger   log.Logger
}

func (o *OtlpProvider) Info(ctx context.Context, s *model.Record) error {
	b, err := s.Jsonify()
	if err != nil {
		return nil
	}
	record := log.Record{}
	record.SetSeverity(log.SeverityInfo)
	record.SetObservedTimestamp(time.Now())
	record.SetTimestamp(time.Now())
	record.SetBody(log.BytesValue(b))
	o.logger.Emit(ctx, record)
	return nil
}

func (o *OtlpProvider) Debug(ctx context.Context, s *model.Record) error {
	b, err := s.Jsonify()
	if err != nil {
		return nil
	}
	record := log.Record{}
	record.SetSeverity(log.SeverityDebug)
	record.SetObservedTimestamp(time.Now())
	record.SetTimestamp(time.Now())
	record.SetBody(log.BytesValue(b))
	o.logger.Emit(ctx, record)
	return nil
}

func (o *OtlpProvider) Warn(ctx context.Context, s *model.Record) error {
	b, err := s.Jsonify()
	if err != nil {
		return nil
	}
	record := log.Record{}
	record.SetSeverity(log.SeverityWarn)
	record.SetObservedTimestamp(time.Now())
	record.SetTimestamp(time.Now())
	record.SetBody(log.BytesValue(b))
	o.logger.Emit(ctx, record)
	return nil
}

func (o *OtlpProvider) Critical(ctx context.Context, s *model.Record) error {
	b, err := s.Jsonify()
	if err != nil {
		return nil
	}
	record := log.Record{}
	record.SetSeverity(log.SeverityFatal)
	record.SetObservedTimestamp(time.Now())
	record.SetTimestamp(time.Now())
	record.SetBody(log.BytesValue(b))
	o.logger.Emit(ctx, record)
	return nil
}

func (o *OtlpProvider) Error(ctx context.Context, s *model.Record) error {
	b, err := s.Jsonify()
	if err != nil {
		return nil
	}
	record := log.Record{}
	record.SetSeverity(log.SeverityError)
	record.SetObservedTimestamp(time.Now())
	record.SetTimestamp(time.Now())
	record.SetBody(log.BytesValue(b))
	o.logger.Emit(ctx, record)
	return nil
}

func (o *OtlpProvider) SetAsGlobal() {
	global.SetLoggerProvider(o.provider)
}

func New(settings ...Option) (*OtlpProvider, error) {
	var (
		c    config
		root OtlpProvider
	)
	for _, setting := range settings {
		setting.apply(&c)
	}
	exp, err := otlploggrpc.New(
		context.Background(),
		otlploggrpc.WithInsecure(),
		otlploggrpc.WithEndpoint(c.address),
		otlploggrpc.WithTimeout(c.requestTimeoutAfter),
	)
	if err != nil {
		return nil, err
	}
	root.provider = sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)))
	global.SetLoggerProvider(root.provider)
	root.logger = root.provider.Logger("main-component")
	root.config = &c
	return &root, nil
}
