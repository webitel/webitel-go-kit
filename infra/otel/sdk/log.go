package otelsdk

import (
	// "github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	// "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log"
	// "go.opentelemetry.io/otel/log/global"
)

// SetLogLevel sets the logger's (used internally to opentelemetry) severity level.
//
// https://github.com/open-telemetry/opentelemetry-go/blob/v1.29.0/internal/global/internal_logging.go#L33
//
// To see Warn messages use a logger with `l.V(1).Enabled() == true`
// To see Info messages use a logger with `l.V(4).Enabled() == true`
// To see Debug messages use a logger with `l.V(8).Enabled() == true`
//
// https://github.com/open-telemetry/opentelemetry-go/blob/v1.29.0/internal/global/internal_logging.go#L20
func SetLogLevel(level log.Severity) {

	// According to https://github.com/open-telemetry/opentelemetry-go/blob/v1.29.0/internal/global/internal_logging.go#L20
	//
	// globalLogger = github.com/go-logr/stdr.New(log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile))

	var v int // verbosity ; "error" = 0
	// if level <= log.SeverityUndefined || log.SeverityError <= level {
	// 	// default: "error"
	// } else if log.SeverityWarn <= level {
	// 	v = 1
	// } else if log.SeverityInfo <= level {
	// 	v = 4
	// } else if log.SeverityTrace <= level {
	// 	v = 8
	// }

	switch {
	// -0 ; "error" +
	case level <= log.SeverityUndefined || log.SeverityError <= level:
		v = 0 // "error" ; default
	// 13+
	case log.SeverityWarn <= level:
		v = 1 // "warn"
	// 9+
	case log.SeverityInfo <= level:
		v = 4 // "info"
	// 1+ ..
	case log.SeverityTrace <= level:
		v = 8 // "debug"
	}

	// switch level {
	// case log.SeverityTrace:
	// 	fallthrough
	// case log.SeverityDebug:
	// 	v = 8
	// case log.SeverityInfo:
	// 	v = 4
	// case log.SeverityWarn:
	// 	v = 1
	// case log.SeverityError:
	// 	fallthrough
	// case log.SeverityFatal:
	// 	v = 0
	// }
	stdr.SetVerbosity(v)

	// var logger logr.Logger
	// if global.GetLoggerProvider() != nil {
	// 	// TODO: bridge to OTEL_LOGS_EXPORTER=
	// }
	// otel.SetLogger(logger)
}
