package otelsdk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/webitel/webitel-go-kit/infra/otel/internal"
	logv "github.com/webitel/webitel-go-kit/infra/otel/log"
	"github.com/webitel/webitel-go-kit/infra/otel/sdk/log"
	"github.com/webitel/webitel-go-kit/infra/otel/sdk/metric"
	"github.com/webitel/webitel-go-kit/infra/otel/sdk/trace"

	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	// "go.opentelemetry.io/contrib/processors/minsev"
	"go.opentelemetry.io/otel"
	otelog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
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
	// RuntimeMetrics enables the standard Go runtime metric set.
	RuntimeMetrics bool
	// option used to set[build] level[logger] for SDK internals
	// https://pkg.go.dev/go.opentelemetry.io/otel#SetLogger
	// https://github.com/open-telemetry/opentelemetry-go/blob/v1.29.0/internal/global/internal_logging.go#L20
	setLevel func(level otelog.Severity) // , bridge bool)
	// option used to bridge third-party logger(s)
	setBridge     []func()
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

// WithSetLevel option allows you [re]set/bridge
// go.opentelemetry.io/otel/log/global.SetLogger()
// the logger used internally to opentelemetry
// with OTEL_LOG_LEVEL= severity specified.
//
// Always called except if OTEL_SDK_DISABLED='true'.
// By default log.SeverityError level is set.
// Default [SetLogLevel] method is used.
func WithSetLevel(set func(otelog.Severity)) Option {
	return option(func(conf *options) {
		conf.setLevel = set
	})
}

// WithLogBridge option allows you to register hook(s)
// that will be triggered for you to redirect (bridge) logger(s)
// which you project is internally using.
//
// Runs when OTEL_LOGS_EXPORTER=? is specified
// and otel.GetLoggerProvider() has processor(s).
func WithLogBridge(do ...func()) Option {
	return option(func(conf *options) {
		conf.setBridge = append(
			conf.setBridge, do...,
		)
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

// WithRuntimeMetrics option enables the standard Go runtime metric set:
// heap use, allocations, the GC goal, goroutine count, GOMAXPROCS and GOGC.
// Default is off. Note there is no CPU-utilisation metric in this set --
// go.processor.limit reports GOMAXPROCS, not load.
//
// Has no effect unless a metrics exporter is also configured: the instruments
// are registered on the MeterProvider that OTEL_METRICS_EXPORTER builds, so
// with no exporter there is nothing to register them on. Likewise suppressed
// entirely by OTEL_SDK_DISABLED=true.
//
// Because newOptions applies environment configuration first and explicit
// Options last, WithRuntimeMetrics(false) overrides OTEL_METRICS_RUNTIME=true.
// Prefer passing an operator-controlled value over a hardcoded false: a
// constant false permanently defeats OTEL_METRICS_RUNTIME for that service,
// and that variable is the only lever a fleet-wide rollout has.
func WithRuntimeMetrics(enable bool) Option {
	return option(func(conf *options) {
		conf.RuntimeMetrics = enable
	})
}

func newOptions(ctx context.Context, opts ...Option) (conf options) {
	// var level slog.LevelVar
	// level.Set(slog.LevelError + 4) // [MIN]: ERROR4, e.g. FATAL
	conf = options{
		Lvl:      otelog.SeverityError, // default: "error"
		Logs:     nil,                  // noop
		Traces:   nil,                  // noop
		Metrics:  nil,                  // noop
		Resource: nil,                  // resource.Environment(),

		setLevel: SetLogLevel,
	}
	// Environment, as next defaults layer ...
	internal.Environment.Apply(
		internal.EnvString("LOG_LEVEL", func(input string) {
			var level logv.Severity
			err := level.UnmarshalText([]byte(input))
			if err != nil {
				err = fmt.Errorf("invalid %s value %s: %w", "OTEL_LOG_LEVEL", input, err)
				otel.Handle(err)
				return // err
			}
			WithLogLevel(otelog.Severity(level)).apply(&conf)
		}),
		internal.EnvString("LOGS_EXPORTER", func(input string) {
			opts, err := log.NewOptions(ctx, input)
			if err != nil {
				err = fmt.Errorf("invalid %s value %s: %w", "OTEL_LOGS_EXPORTER", input, err)
				otel.Handle(err)
				return // err
			}
			WithLogOptions(opts...).apply(&conf)
			// exporter, err := log.NewExporter(ctx, input)
			// if err != nil {
			// 	err = fmt.Errorf("invalid %s value %s: %w", "OTEL_LOGS_EXPORTER", input, err)
			// 	otel.Handle(err)
			// 	return // err
			// }
			// var processor sdklog.Processor
			// processor = sdklog.NewBatchProcessor(
			// 	exporter, // options ...
			// 	// sdklog.WithMaxQueueSize(2048),
			// 	// sdklog.WithExportMaxBatchSize(512),
			// 	// sdklog.WithExportInterval(time.Second),
			// 	// sdklog.WithExportTimeout(time.Second*30),
			// )
			// // Wrap log.Processor with log.Severity filter !
			// // // TODO: default conf.Lvl !!!
			// // processor = minsev.NewLogProcessor(
			// // 	processor, conf.Lvl, // otelog.SeverityInfo, // [ INFO+, WARN+, ERROR+, FATAL+ ]
			// // )
			// WithLogOptions(
			// 	sdklog.WithProcessor(processor),
			// ).apply(&conf)
		}),
		internal.EnvString("TRACES_EXPORTER", func(input string) {
			opts, err := trace.NewOptions(ctx, input)
			if err != nil {
				err = fmt.Errorf("invalid %s value %s: %w", "OTEL_TRACES_EXPORTER", input, err)
				otel.Handle(err)
				return // err
			}
			WithTraceOptions(opts...).apply(&conf)
		}),
		internal.EnvString("METRICS_EXPORTER", func(input string) {
			opts, err := metric.NewOptions(ctx, input)
			if err != nil {
				err = fmt.Errorf("invalid %s value %s: %w", "OTEL_METRICS_EXPORTER", input, err)
				otel.Handle(err)
				return // err
			}
			WithMetricOptions(opts...).apply(&conf)
		}),
		internal.EnvString("METRICS_RUNTIME", func(input string) {
			// Never route this parse error through otel.Handle. Sibling
			// callbacks do, but their handler joins into Configure's local
			// errs, which is gated BEFORE resource, logs, traces and metrics
			// are built -- so one mistyped value here would fail Configure and
			// blind the whole process. Fall back to "off", as SDK_DISABLED
			// does below.
			//
			// slog is safe where otel.Handle is not: it never touches errs.
			// Warn rather than fail, because ParseBool rejects values an
			// operator may try ("yes", "on", "TrUe") and this variable is
			// Webitel-local, so silence leaves them nothing to search for.
			enable, err := strconv.ParseBool(input)
			if err != nil {
				slog.Warn(
					"otel: invalid OTEL_METRICS_RUNTIME value, runtime metrics stay disabled",
					"value", input,
				)
			}
			WithRuntimeMetrics(enable).apply(&conf)
		}),
	)
	// Apply custom after, to be able to override defaults
	for _, opt := range opts {
		opt.apply(&conf)
	}
	return // conf
}

// // Deprecated. Use [Configure] instead.
// func Setup(ctx context.Context, opts ...Option) (ShutdownFunc, error) {
// 	return Configure(ctx, opts...)
// }

// Configure [O]pen[Tel]emetry SDK environment.
func Configure(ctx context.Context, opts ...Option) (ShutdownFunc, error) {

	disabled := false
	internal.Environment.Apply(
		internal.EnvString("SDK_DISABLED", func(s string) {
			disabled, _ = strconv.ParseBool(s)
		}),
	)

	if disabled {
		return func(context.Context) error { return nil }, nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	// Handles any error deemed irremediable by an OpenTelemetry component.
	// errorHandler := otel.GetErrorHandler()
	var (
		errs         error
		errorHandler = otel.ErrorHandlerFunc(func(err error) {

			if err == nil {
				return // none
			}

			errs = errors.Join(errs, err)

			// slog.Error(err.Error())
			slog.Error(
				"[O]pen[Tel]emetry configuration ;",
				"error", err,
			)

			// // Fatal
			// os.Exit(1)

		})
	)
	// Log&exit: errors while initialization ...
	otel.SetErrorHandler(errorHandler)

	// Read ENV configuration ...
	setup := newOptions(ctx, opts...)
	if errs != nil {
		// An exporter's option constructor may already have acquired an OS
		// resource: the prometheus metrics exporter opens its own listener
		// while building options, because that is the only place a bind error
		// can still be reported. On this path the MeterProvider below is never
		// constructed, so nothing owns that reader and nothing ever closes it
		// -- and because ANY signal's bad exporter lands here, a misconfigured
		// OTEL_LOGS_EXPORTER would strand the metrics port for the life of the
		// process. Give the readers an owner so the returned ShutdownFunc can
		// release them.
		if len(setup.Metrics) > 0 {
			setup.OnShutdown(sdkmetric.NewMeterProvider(setup.Metrics...).Shutdown)
		}
		// USE: otel.Handle(err)
		return setup.Shutdown, errs
	}

	// --------------------------------------- //
	//                resource                 //
	// --------------------------------------- //

	src, _ := resource.Merge(
		resource.Default(), setup.Resource,
	)

	// --------------------------------------- //
	//              propagation                //
	// --------------------------------------- //

	// TODO: make as default with Option(-al) changes
	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)

	// --------------------------------------- //
	//                  logs                   //
	// --------------------------------------- //

	if len(setup.Logs) > 0 {
		provider := sdklog.NewLoggerProvider(
			append(setup.Logs, sdklog.WithResource(src))...,
		)
		setup.OnShutdown(provider.Shutdown)
		global.SetLoggerProvider(provider)

		// otel/sdk/log.Logger("otel").Error(err)
		errorHandler = func(err error) {

			if err == nil {
				return // none
			}

			var (
				event otelog.Record
				level = otelog.SeverityError
			)

			event.SetTimestamp(time.Now())
			event.SetSeverityText("ERROR")
			event.SetSeverity(level)

			event.SetBody(otelog.StringValue(err.Error()))

			global.GetLoggerProvider().Logger("otel").
				Emit(context.Background(), event)

			// // Fatal
			// os.Exit(1)

		}
		// Just log any ERROR deemed irremediable by an OpenTelemetry component
		otel.SetErrorHandler(errorHandler)

		// TODO: redirect (bridge) internal logger(s)
		for _, doBridge := range setup.setBridge {
			if doBridge != nil {
				doBridge()
			}
		}

	}
	// OTEL_LOG_LEVEL = ? ; default: "error"
	if setup.Lvl > otelog.SeverityUndefined {
		// [re]set https://pkg.go.dev/go.opentelemetry.io/otel#SetLogger(!)
		// severity/verbosity OTEL_LOG_LEVEL=
		if setup.setLevel != nil {
			setup.setLevel(setup.Lvl)
		}
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

		if setup.RuntimeMetrics {
			// The interval overrides an upstream default of 15s that caches
			// samples between collections -- a fossil of the deprecated
			// implementation's runtime.ReadMemStats, where the current path
			// calls runtime/metrics.Read for well under a microsecond. Left at
			// 15s, the pull-based Prometheus exporter (where the scrape IS the
			// collection) serves identical stale values for up to 15s. Must be
			// a small POSITIVE duration: newConfig maps <= 0 back to 15s.
			//
			// errs is already gated above and never read again, so otel.Handle
			// here only logs -- deliberately: failing to register runtime
			// metrics must not tear down logs and traces.
			if err := otelruntime.Start(
				otelruntime.WithMeterProvider(provider),
				otelruntime.WithMinimumReadMemStatsInterval(time.Second),
			); err != nil {
				otel.Handle(fmt.Errorf("otel/sdk: runtime metrics: %w", err))
			}
		}
	}

	return setup.Shutdown, nil
}
