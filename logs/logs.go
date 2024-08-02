package logs

import (
	"context"
	"errors"
	"github.com/webitel/webitel-go-kit/logs/jsonl_provider"
	"github.com/webitel/webitel-go-kit/logs/model"
	"github.com/webitel/webitel-go-kit/logs/otlp_provider"
	"github.com/webitel/webitel-go-kit/logs/stdout_provider"
	"github.com/webitel/webitel-go-kit/logs/wlog_provider"
	"github.com/webitel/wlog"
	"go.opentelemetry.io/otel/trace"
	"time"
)

const (
	Otlp   = "otlp"
	Stdout = "stdout"
	Wlog   = "wlog"
	Jsonl  = "jsonl"
)

var (
	globalLogger *DynamicLogger
)

type DynamicLogger struct {
	logger       model.LogProvider
	nativeLogger *wlog.Logger
	config       *config
}

// New creates new logger
func New(nativeLogger *wlog.Logger, opts ...Option) (*DynamicLogger, error) {
	var (
		conf      config
		err       error
		dynLogger DynamicLogger
	)
	if nativeLogger == nil {
		return nil, errors.New("default logger is nil")
	}
	for _, opt := range opts {
		opt.apply(&conf)
	}
	dynLogger.config = &conf
	switch conf.exporter {
	case Otlp:
		dynLogger.logger, err = otlp_provider.New(otlp_provider.WithAddress(conf.address), otlp_provider.WithTimeout(time.Second*5))
	case Stdout:
		dynLogger.logger, err = stdout_provider.New()
	case Jsonl:
		dynLogger.logger, err = jsonl_provider.New(jsonl_provider.WithServiceName(conf.serviceName))
	default:
		dynLogger.logger, err = wlog_provider.New(conf.logger, wlog_provider.WithLogType(conf.fileFormat))
	}
	if err != nil {
		return nil, err
	}
	dynLogger.logger.SetAsGlobal()
	globalLogger = &dynLogger
	dynLogger.nativeLogger = nativeLogger
	return &dynLogger, nil
}

func (d *DynamicLogger) Info(ctx context.Context, message string, requestCtx map[string]string) {
	lvl := model.InfoLevel
	if includeLog(lvl, d.config.logLevel) || message == "" {
		return
	}
	log := d.GetDefaultLog(ctx, lvl, requestCtx)
	log.Message = message
	err := d.logger.Info(ctx, log)
	if err != nil {
		d.nativeLogger.Error(err.Error())
	}
}

func (d *DynamicLogger) Debug(ctx context.Context, message string, requestCtx map[string]string) {
	lvl := model.DebugLevel
	if !includeLog(lvl, d.config.logLevel) {
		return
	}
	log := d.GetDefaultLog(ctx, lvl, requestCtx)
	log.Message = message
	err := d.logger.Debug(ctx, log)
	if err != nil {
		d.nativeLogger.Error(err.Error())
	}
}

func (d *DynamicLogger) Warn(ctx context.Context, message string, requestCtx map[string]string) {
	lvl := model.WarnLevel
	if !includeLog(lvl, d.config.logLevel) {
		return
	}
	log := d.GetDefaultLog(ctx, lvl, requestCtx)
	log.Message = message
	err := d.logger.Warn(ctx, log)
	if err != nil {
		d.nativeLogger.Error(err.Error())
	}
}

func (d *DynamicLogger) Error(ctx context.Context, err model.AppError, requestCtx map[string]string) {
	lvl := model.ErrorLevel
	if !includeLog(lvl, globalLogger.config.logLevel) {
		return
	}
	log := d.GetDefaultLog(ctx, lvl, requestCtx)
	log.Error = convertDefaultErrorOutput(err)
	logErr := d.logger.Error(ctx, log)
	if logErr != nil {
		d.nativeLogger.Error(err.Error())
	}
}

func (d *DynamicLogger) Critical(ctx context.Context, err model.AppError, requestCtx map[string]string) {
	lvl := model.CriticalLevel
	if !includeLog(lvl, globalLogger.config.logLevel) {
		return
	}
	log := d.GetDefaultLog(ctx, lvl, requestCtx)
	log.Error = convertDefaultErrorOutput(err)
	logErr := d.logger.Critical(ctx, log)
	if logErr != nil {
		d.nativeLogger.Error(err.Error())
	}
}

type logOpts struct {
	ctx        context.Context
	requestCtx map[string]string
}

// Option specifies instrumentation configuration options.
type LogOption interface {
	apply(*logOpts)
}

type logOptFunc func(*logOpts)

func (o logOptFunc) apply(c *logOpts) {
	o(c)
}

func LogWithContext(ctx context.Context) LogOption {
	return logOptFunc(func(c *logOpts) {
		c.ctx = ctx
	})
}

func LogWithRequestContext(ctx map[string]string) LogOption {
	return logOptFunc(func(c *logOpts) {
		c.requestCtx = ctx
	})
}

func Info(message string, opts ...LogOption) {
	if globalLogger == nil {
		return
	}
	conf := getLogOptsConfig(opts...)
	globalLogger.Info(conf.ctx, message, conf.requestCtx)
}

func Debug(message string, opts ...LogOption) {
	if globalLogger == nil {
		return
	}
	conf := getLogOptsConfig(opts...)
	globalLogger.Debug(conf.ctx, message, conf.requestCtx)
}

func Warn(message string, opts ...LogOption) {
	if globalLogger == nil {
		return
	}
	conf := getLogOptsConfig(opts...)
	globalLogger.Warn(conf.ctx, message, conf.requestCtx)
}

func Error(e model.AppError, opts ...LogOption) {
	if globalLogger == nil {
		return
	}
	conf := getLogOptsConfig(opts...)
	globalLogger.Error(conf.ctx, e, conf.requestCtx)
}

func Critical(e model.AppError, opts ...LogOption) {
	if globalLogger == nil {
		return
	}
	conf := getLogOptsConfig(opts...)
	globalLogger.Error(conf.ctx, e, conf.requestCtx)
}

func includeLog(inputLevel model.LogLevel, defaultLevel model.LogLevel) bool {
	return inputLevel.Int() <= defaultLevel.Int()
}

func (d *DynamicLogger) GetDefaultLog(ctx context.Context, logLevel model.LogLevel, requestCtx map[string]string) *model.Record {
	var (
		log = model.Record{
			Service: model.Service{
				Id:      d.config.consulId,
				Name:    d.config.serviceName,
				Version: d.config.serviceVersion,
				Build:   d.config.build,
			},
			Context:   requestCtx,
			Timestamp: time.Now(),
			Level:     logLevel.String(),
		}
	)
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		log.TraceId = spanCtx.TraceID().String()
	}
	if spanCtx.HasSpanID() {
		log.SpanId = spanCtx.SpanID().String()
	}
	return &log
}

func convertDefaultErrorOutput(err model.AppError) *model.ErrorType {
	return &model.ErrorType{
		Id:      err.GetId(),
		Code:    err.GetStatusCode(),
		Message: err.GetDetailedError(),
	}
}

func getLogOptsConfig(opts ...LogOption) *logOpts {
	var (
		conf logOpts
	)
	for _, opt := range opts {
		opt.apply(&conf)
	}
	return &conf
}
