package stdout_provider

import (
	"context"
	"github.com/webitel/webitel-go-kit/logs/model"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"time"
)

type StdoutProvider struct {
	provider log.LoggerProvider
	logger   log.Logger
}

func (o *StdoutProvider) Info(ctx context.Context, s *model.Record) error {
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

func (o *StdoutProvider) Debug(ctx context.Context, s *model.Record) error {
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

func (o *StdoutProvider) Warn(ctx context.Context, s *model.Record) error {
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

func (o *StdoutProvider) Critical(ctx context.Context, s *model.Record) error {
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

func (o *StdoutProvider) Error(ctx context.Context, s *model.Record) error {
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

func (o *StdoutProvider) SetAsGlobal() {
	global.SetLoggerProvider(o.provider)
}

func New() (*StdoutProvider, error) {
	var (
		root StdoutProvider
	)
	exp, err := stdoutlog.New()
	if err != nil {
		return nil, err
	}
	root.provider = sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)))
	global.SetLoggerProvider(root.provider)
	root.logger = root.provider.Logger("main-component")
	return &root, nil
}
