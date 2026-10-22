package otelsdk

import (
	"log/slog"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/log"

	"github.com/webitel/webitel-go-kit/pkg/logging"
	logradapter "github.com/webitel/webitel-go-kit/pkg/logging/adapters/logr"
)

const internalComponent logging.Component = "go.opentelemetry.io/otel"

var internalLevel = func() *slog.LevelVar {
	var v slog.LevelVar
	v.Set(logging.LevelError.Level())
	return &v
}()

// SetLogLevel sets the minimum severity of the logger used internally by
// OpenTelemetry. Errors are always written: a severity above ERROR, or
// [log.SeverityUndefined], sets ERROR.
func SetLogLevel(level log.Severity) {
	lvl := logging.LevelError
	if level > log.SeverityUndefined {
		lvl = min(logging.LevelOf(level), logging.LevelError)
	}
	internalLevel.Set(lvl.Level())
}

func internalLogger() logr.Logger {
	return logradapter.New(
		logging.Bootstrap(logging.WithLevel(internalLevel)),
		logradapter.WithComponent(internalComponent),
		logradapter.WithVerbosity(internalVerbosity),
	)
}

func internalVerbosity(v int) logging.Level {
	switch {
	case v <= 1:
		return logging.LevelWarn
	case v <= 4:
		return logging.LevelInfo
	case v <= 8:
		return logging.LevelDebug
	default:
		return logging.LevelTrace
	}
}
