package rabbitmq

import (
	"log/slog"
)

// SlogLogger is an adapter for slog.Logger to implement the rabbitmq.Logger interface.
type SlogLogger struct {
	log *slog.Logger
}

// NewSlogLogger creates a new SlogLogger.
func NewSlogLogger(logger *slog.Logger) *SlogLogger {
	return &SlogLogger{log: logger}
}

// Info logs informational messages with structured args.
func (l *SlogLogger) Info(msg string, args ...any) {
	l.log.LogAttrs(nil, slog.LevelInfo, msg, toSlogAttrs(args)...)
}

// Warn logs warning messages with structured args.
func (l *SlogLogger) Warn(msg string, args ...any) {
	l.log.LogAttrs(nil, slog.LevelWarn, msg, toSlogAttrs(args)...)
}

// Error logs error messages with the error as an attribute.
func (l *SlogLogger) Error(msg string, err error, args ...any) {
	attrs := append([]any{"error", err}, args...)
	l.log.LogAttrs(nil, slog.LevelError, msg, toSlogAttrs(attrs)...)
}

// toSlogAttrs converts key-value pairs to slog.Attr.
func toSlogAttrs(args []any) []slog.Attr {
	var attrs []slog.Attr
	for i := 0; i < len(args)-1; i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		attrs = append(attrs, slog.Any(key, args[i+1]))
	}
	return attrs
}
