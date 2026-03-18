package slogadapter

import "log/slog"

type Adapter struct {
	log *slog.Logger
}

func New(log *slog.Logger) *Adapter {
	return &Adapter{log: log}
}

// Info logs informational messages with structured args.
func (a *Adapter) Info(msg string, args ...any) {
	a.log.LogAttrs(nil, slog.LevelInfo, msg, toSlogAttrs(args)...)
}

// Error logs error messages with the error as an attribute.
func (a *Adapter) Error(msg string, args ...any) {
	a.log.LogAttrs(nil, slog.LevelError, msg, toSlogAttrs(args)...)
}

// Debug logs debug messages with the structured args.
func (a *Adapter) Debug(msg string, args ...any) {
	a.log.LogAttrs(nil, slog.LevelDebug, msg, toSlogAttrs(args)...)
}

// Warn logs warning messages with structured args.
func (a *Adapter) Warn(msg string, args ...any) {
	a.log.LogAttrs(nil, slog.LevelWarn, msg, toSlogAttrs(args)...)
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
