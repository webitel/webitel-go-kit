package fxadapter

import (
	"context"
	"strings"

	"go.uber.org/fx/fxevent"

	"github.com/webitel/webitel-go-kit/pkg/logging"
)

const Component logging.Component = "fx"

const (
	keyCallee      = "fx.callee"
	keyCaller      = "fx.caller"
	keyRuntime     = "fx.runtime"
	keyType        = "fx.type"
	keyModule      = "fx.module"
	keyConstructor = "fx.constructor"
	keyDecorator   = "fx.decorator"
	keyFunction    = "fx.function"
	keyName        = "fx.name"
	keyKind        = "fx.kind"
	keyPrivate     = "fx.private"
	keySignal      = "fx.signal"
	keyStackTrace  = "fx.stacktrace"
	keyModuleTrace = "fx.moduletrace"
)

// New returns new [fxevent.Logger] with "fx" component.
// Routine lifecycle events are DEBUG, start and stop signals
// are INFO, failures are ERROR.
func New(l *logging.Logger) fxevent.Logger {
	return &logger{log: l.WithComponent(Component)}
}

type logger struct {
	log *logging.Logger
}

var _ fxevent.Logger = (*logger)(nil)

func (l *logger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		l.debug("OnStart hook executing",
			logging.Str(keyCallee, e.FunctionName),
			logging.Str(keyCaller, e.CallerName),
		)
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.error("OnStart hook failed", e.Err,
				logging.Str(keyCallee, e.FunctionName),
				logging.Str(keyCaller, e.CallerName),
			)
		} else {
			l.debug("OnStart hook executed",
				logging.Str(keyCallee, e.FunctionName),
				logging.Str(keyCaller, e.CallerName),
				logging.Dur(keyRuntime, e.Runtime),
			)
		}
	case *fxevent.OnStopExecuting:
		l.debug("OnStop hook executing",
			logging.Str(keyCallee, e.FunctionName),
			logging.Str(keyCaller, e.CallerName),
		)
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.error("OnStop hook failed", e.Err,
				logging.Str(keyCallee, e.FunctionName),
				logging.Str(keyCaller, e.CallerName),
			)
		} else {
			l.debug("OnStop hook executed",
				logging.Str(keyCallee, e.FunctionName),
				logging.Str(keyCaller, e.CallerName),
				logging.Dur(keyRuntime, e.Runtime),
			)
		}
	case *fxevent.Supplied:
		if e.Err != nil {
			l.error("error encountered while applying options", e.Err,
				logging.Str(keyType, e.TypeName),
				module(e.ModuleName),
				traces(e.StackTrace, e.ModuleTrace),
			)
		} else {
			l.debug("supplied",
				logging.Str(keyType, e.TypeName),
				module(e.ModuleName),
			)
		}
	case *fxevent.Provided:
		for _, rtype := range e.OutputTypeNames {
			l.debug("provided",
				logging.Str(keyConstructor, e.ConstructorName),
				logging.Str(keyType, rtype),
				module(e.ModuleName),
				private(e.Private),
			)
		}
		if e.Err != nil {
			l.error("error encountered while applying options", e.Err,
				module(e.ModuleName),
				traces(e.StackTrace, e.ModuleTrace),
			)
		}
	case *fxevent.Replaced:
		for _, rtype := range e.OutputTypeNames {
			l.debug("replaced",
				logging.Str(keyType, rtype),
				module(e.ModuleName),
			)
		}
		if e.Err != nil {
			l.error("error encountered while replacing", e.Err,
				module(e.ModuleName),
				traces(e.StackTrace, e.ModuleTrace),
			)
		}
	case *fxevent.Decorated:
		for _, rtype := range e.OutputTypeNames {
			l.debug("decorated",
				logging.Str(keyDecorator, e.DecoratorName),
				logging.Str(keyType, rtype),
				module(e.ModuleName),
			)
		}
		if e.Err != nil {
			l.error("error encountered while applying options", e.Err,
				module(e.ModuleName),
				traces(e.StackTrace, e.ModuleTrace),
			)
		}
	case *fxevent.BeforeRun:
		l.debug("before run",
			logging.Str(keyName, e.Name),
			logging.Str(keyKind, e.Kind),
			module(e.ModuleName),
		)
	case *fxevent.Run:
		if e.Err != nil {
			l.error("error returned", e.Err,
				logging.Str(keyName, e.Name),
				logging.Str(keyKind, e.Kind),
				module(e.ModuleName),
			)
		} else {
			l.debug("run",
				logging.Str(keyName, e.Name),
				logging.Str(keyKind, e.Kind),
				logging.Dur(keyRuntime, e.Runtime),
				module(e.ModuleName),
			)
		}
	case *fxevent.Invoking:
		l.debug("invoking",
			logging.Str(keyFunction, e.FunctionName),
			module(e.ModuleName),
		)
	case *fxevent.Invoked:
		if e.Err != nil {
			l.error("invoke failed", e.Err,
				logging.Str(keyFunction, e.FunctionName),
				module(e.ModuleName),
				stack(e.Trace),
			)
		}
	case *fxevent.Stopping:
		l.info("received signal",
			logging.Str(keySignal, strings.ToUpper(e.Signal.String())),
		)
	case *fxevent.Stopped:
		if e.Err != nil {
			l.error("stop failed", e.Err)
		}
	case *fxevent.RollingBack:
		l.error("start failed, rolling back", e.StartErr)
	case *fxevent.RolledBack:
		if e.Err != nil {
			l.error("rollback failed", e.Err)
		}
	case *fxevent.Started:
		if e.Err != nil {
			l.error("start failed", e.Err)
		} else {
			l.info("app started")
		}
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			l.error("custom logger initialization failed", e.Err)
		} else {
			l.debug("initialized custom fxevent.Logger",
				logging.Str(keyFunction, e.ConstructorName),
			)
		}
	}
}

func (l *logger) debug(msg string, fields ...logging.Field) {
	l.log.DebugCtx(context.Background(), msg, fields...)
}

func (l *logger) info(msg string, fields ...logging.Field) {
	l.log.InfoCtx(context.Background(), msg, fields...)
}

func (l *logger) error(msg string, err error, fields ...logging.Field) {
	l.log.ErrorCtx(context.Background(), msg, append(fields, logging.Err(err))...)
}

func module(name string) logging.Field {
	if name == "" {
		return logging.Field{}
	}
	return logging.Str(keyModule, name)
}

func private(on bool) logging.Field {
	if !on {
		return logging.Field{}
	}
	return logging.Bool(keyPrivate, true)
}

func traces(stack, modules []string) logging.Field {
	var fields []logging.Field
	if len(stack) > 0 {
		fields = append(fields, logging.Any(keyStackTrace, stack))
	}
	if len(modules) > 0 {
		fields = append(fields, logging.Any(keyModuleTrace, modules))
	}
	return logging.Group("", fields...)
}

func stack(trace string) logging.Field {
	if trace == "" {
		return logging.Field{}
	}
	return logging.Str(keyStackTrace, trace)
}
