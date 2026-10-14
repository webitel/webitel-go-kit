package depenlog

import (
	"github.com/webitel/webitel-go-kit/pkg/logger"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
	"go.uber.org/fx/fxevent"
)

// FxLogger adapts the kit logger to fx's event logger, tagged component=fx.
// Wire it into an fx app with:
//
//	fx.WithLogger(func() fxevent.Logger { return log.FxLogger(l) })
//
// Routine lifecycle events are logged at debug to keep startup quiet; signals
// and start/stop transitions at info; failures at error (semconv.ErrorKey).
func FxLogger(l logger.Logger) fxevent.Logger {
	return &fxLogger{log: WithComponent(l, "fx")}
}

type fxLogger struct {
	log logger.Logger
}

var _ fxevent.Logger = (*fxLogger)(nil)

func (l *fxLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		l.log.Debug("OnStart hook executing", "callee", e.FunctionName, "caller", e.CallerName)
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.log.Error("OnStart hook failed", "callee", e.FunctionName, "caller", e.CallerName, semconv.ErrorKey, e.Err)
		} else {
			l.log.Debug("OnStart hook executed", "callee", e.FunctionName, "caller", e.CallerName, "runtime", e.Runtime.String())
		}
	case *fxevent.OnStopExecuting:
		l.log.Debug("OnStop hook executing", "callee", e.FunctionName, "caller", e.CallerName)
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.log.Error("OnStop hook failed", "callee", e.FunctionName, "caller", e.CallerName, semconv.ErrorKey, e.Err)
		} else {
			l.log.Debug("OnStop hook executed", "callee", e.FunctionName, "caller", e.CallerName, "runtime", e.Runtime.String())
		}
	case *fxevent.Supplied:
		if e.Err != nil {
			l.log.Error("error encountered while applying options", "type", e.TypeName, "module", e.ModuleName, semconv.ErrorKey, e.Err)
		} else {
			l.log.Debug("supplied", "type", e.TypeName, "module", e.ModuleName)
		}
	case *fxevent.Provided:
		for _, rtype := range e.OutputTypeNames {
			l.log.Debug("provided", "constructor", e.ConstructorName, "module", e.ModuleName, "type", rtype)
		}
		if e.Err != nil {
			l.log.Error("error encountered while applying options", "module", e.ModuleName, semconv.ErrorKey, e.Err)
		}
	case *fxevent.Replaced:
		for _, rtype := range e.OutputTypeNames {
			l.log.Debug("replaced", "module", e.ModuleName, "type", rtype)
		}
		if e.Err != nil {
			l.log.Error("error encountered while replacing", "module", e.ModuleName, semconv.ErrorKey, e.Err)
		}
	case *fxevent.Decorated:
		for _, rtype := range e.OutputTypeNames {
			l.log.Debug("decorated", "decorator", e.DecoratorName, "module", e.ModuleName, "type", rtype)
		}
		if e.Err != nil {
			l.log.Error("error encountered while applying options", "module", e.ModuleName, semconv.ErrorKey, e.Err)
		}
	case *fxevent.Invoking:
		l.log.Debug("invoking", "function", e.FunctionName, "module", e.ModuleName)
	case *fxevent.Invoked:
		if e.Err != nil {
			l.log.Error("invoke failed", "function", e.FunctionName, "module", e.ModuleName, semconv.ErrorKey, e.Err)
		}
	case *fxevent.Stopping:
		l.log.Info("received signal", "signal", e.Signal.String())
	case *fxevent.Stopped:
		if e.Err != nil {
			l.log.Error("stop failed", semconv.ErrorKey, e.Err)
		}
	case *fxevent.RollingBack:
		l.log.Error("start failed, rolling back", semconv.ErrorKey, e.StartErr)
	case *fxevent.RolledBack:
		if e.Err != nil {
			l.log.Error("rollback failed", semconv.ErrorKey, e.Err)
		}
	case *fxevent.Started:
		if e.Err != nil {
			l.log.Error("start failed", semconv.ErrorKey, e.Err)
		} else {
			l.log.Info("started")
		}
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			l.log.Error("custom logger initialization failed", semconv.ErrorKey, e.Err)
		} else {
			l.log.Debug("initialized custom fxevent.Logger", "constructor", e.ConstructorName)
		}
	}
}
