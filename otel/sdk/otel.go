package otelsdk

import (
	"context"
	"errors"
	stdlog "log"
	"os"
	"sync"

	"github.com/go-logr/stdr"
	"github.com/webitel/webitel-go-kit/otel/internal"
	"github.com/webitel/webitel-go-kit/otel/log/bridge/logr"
	"github.com/webitel/webitel-go-kit/otel/sdk/log"
	"github.com/webitel/webitel-go-kit/otel/sdk/metric"
	"github.com/webitel/webitel-go-kit/otel/sdk/trace"
	"go.opentelemetry.io/otel"
	otelog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// // go.opentelemetry.io/otel/sdk/** Provider(s) implemented
// type Shutable interface {
// 	// Shutdown shuts down the provider and releasing any held resources.
// 	Shutdown(context.Context) error
// }

type ShutdownFunc func(context.Context) error

// var _ Shutable = ShutdownFunc(nil)

// func (hook ShutdownFunc) Shutdown(ctx context.Context) error {
// 	if hook != nil {
// 		return hook.Shutdown(ctx)
// 	}
// 	// ignore
// 	return nil
// }

// configuration
type options struct {
	// Logs output severity level
	Lvl      otelog.Severity
	Logs     []log.Option
	Traces   []trace.Option
	Metrics  []metric.Option
	Resource *resource.Resource

	shutdownTx    sync.Mutex
	shutdownHooks []ShutdownFunc
}

func (c *options) OnShutdown(hook ShutdownFunc) {
	if hook == nil {
		return
	}
	c.shutdownTx.Lock()
	defer c.shutdownTx.Unlock()
	c.shutdownHooks = append(c.shutdownHooks, hook)
}

func (c *options) Shutdown(ctx context.Context) error {
	if c == nil {
		return nil
	}

	var err error
	c.shutdownTx.Lock()
	defer c.shutdownTx.Unlock()

	for _, do := range c.shutdownHooks {
		err = errors.Join(err, do(ctx))
	}

	c.shutdownHooks = nil
	return err
}

type Option interface {
	apply(*options)
}

type option func(c *options)

var _ Option = option(nil)

func (fn option) apply(c *options) {
	if fn != nil {
		fn(c)
	}
}

func WithResource(src *resource.Resource) Option {
	return option(func(conf *options) {
		conf.Resource = src
	})
}

func WithLogLevel(max otelog.Severity) Option {
	return option(func(conf *options) {
		conf.Lvl = max
	})
}

func WithLogOptions(opts ...log.Option) Option {
	return option(func(conf *options) {
		if len(opts) > 0 {
			conf.Logs = append(conf.Logs, opts...)
		}
	})
}

func WithTraceOptions(opts ...trace.Option) Option {
	return option(func(conf *options) {
		if len(opts) > 0 {
			conf.Traces = append(conf.Traces, opts...)
		}
	})
}

func WithMetricOptions(opts ...metric.Option) Option {
	return option(func(conf *options) {
		if len(opts) > 0 {
			conf.Metrics = append(conf.Metrics, opts...)
		}
	})
}

func newOptions(ctx context.Context, opts ...Option) (conf options) {
	// var level slog.LevelVar
	// level.Set(slog.LevelError + 4) // [MIN]: ERROR4, e.g. FATAL
	conf = options{
		Lvl:      0,   // otelog.SeverityUndefined, // disabled
		Logs:     nil, // noop
		Traces:   nil, // noop
		Metrics:  nil, // noop
		Resource: nil, // resource.Environment(),
	}
	// Environment, as next defaults layer ...
	internal.Environment.Apply(
		internal.EnvString("LOG_LEVEL", func(input string) {
			WithLogLevel(otelog.SeverityTrace).apply(&conf)
		}),
		internal.EnvString("LOG_EXPORT", func(input string) {
			exports, err := log.NewOptions(ctx, input)
			if err != nil {
				panic(err)
			}
			WithLogOptions(exports...).apply(&conf)
		}),
		internal.EnvString("TRACE_EXPORT", func(input string) {
			exports, err := trace.NewOptions(ctx, input)
			if err != nil {
				panic(err)
			}
			WithTraceOptions(exports...).apply(&conf)
		}),
		internal.EnvString("METRIC_EXPORT", func(input string) {
			exports, err := metric.NewOptions(ctx, input)
			if err != nil {
				panic(err)
			}
			WithMetricOptions(exports...).apply(&conf)
		}),
	)
	// Apply custom after, to be able to override defaults
	for _, opt := range opts {
		opt.apply(&conf)
	}
	return // conf
}

func Setup(ctx context.Context, opts ...Option) (ShutdownFunc, error) {

	if ctx == nil {
		ctx = context.Background()
	}
	setup := newOptions(ctx, opts...)

	// --------------------------------------- //
	//                resource                 //
	// --------------------------------------- //

	src := setup.Resource
	if src == nil {
		src, _ = resource.Merge(
			resource.Default(),
			resource.Environment(),
		)
	}

	// --------------------------------------- //
	//                  logs                   //
	// --------------------------------------- //

	// STDOUT while OTEL log.Provider initialization ...
	if setup.Lvl > otelog.SeverityUndefined {
		logger := stdr.NewWithOptions(stdlog.New(
			os.Stdout, "", stdlog.LstdFlags|stdlog.Lshortfile,
		),
			stdr.Options{
				Depth:     2,
				LogCaller: stdr.All,
			},
		)
		stdr.SetVerbosity(99) // int(setup.Lvl))
		// NOTE: logger used internally to opentelemetry.
		otel.SetLogger(logger)
	}

	if len(setup.Logs) > 0 {
		provider := sdklog.NewLoggerProvider(
			append(setup.Logs, sdklog.WithResource(src))...,
		)
		setup.OnShutdown(provider.Shutdown)
		global.SetLoggerProvider(provider)
		// emitter := provider.Logger(
		// 	"service",
		// 	// otelog.WithInstrumentationAttributes(attr ...attribute.KeyValue),
		// 	// otelog.WithInstrumentationVersion(version string),
		// 	// otelog.WithSchemaURL(schemaURL string),
		// )
	}

	// OTEL log.Provider bridge !
	if setup.Lvl > otelog.SeverityUndefined {
		logger := logr.NewLogger("otel")
		// NOTE: logger used internally to opentelemetry.
		otel.SetLogger(logger)
	}

	// --------------------------------------- //
	//                traces                   //
	// --------------------------------------- //

	if len(setup.Traces) > 0 {
		provider := sdktrace.NewTracerProvider(
			append(setup.Traces, sdktrace.WithResource(src))...,
		)
		setup.OnShutdown(provider.Shutdown)
		otel.SetTracerProvider(provider)
	}

	// --------------------------------------- //
	//               metrics                   //
	// --------------------------------------- //

	if len(setup.Metrics) > 0 {
		provider := sdkmetric.NewMeterProvider(
			append(setup.Metrics, sdkmetric.WithResource(src))...,
		)
		setup.OnShutdown(provider.Shutdown)
		otel.SetMeterProvider(provider)
	}

	return setup.Shutdown, nil
}
