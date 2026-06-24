package logger

import (
	"context"
	"log/slog"
)

type SlogAdapter struct {
	log *slog.Logger
}

func NewSlog(log *slog.Logger) Logger {
	return &SlogAdapter{log: log}
}

// Info logs informational messages with structured args.
func (a *SlogAdapter) Info(msg string, args ...any) {
	a.InfoContext(context.Background(), msg, args...)
}

// Error logs error messages with the error as an attribute.
func (a *SlogAdapter) Error(msg string, args ...any) {
	a.ErrorContext(context.Background(), msg, args...)
}

// Debug logs debug messages with the structured args.
func (a *SlogAdapter) Debug(msg string, args ...any) {
	a.DebugContext(context.Background(), msg, args...)
}

// Warn logs warning messages with structured args.
func (a *SlogAdapter) Warn(msg string, args ...any) {
	a.WarnContext(context.Background(), msg, args...)
}

// InfoContext logs at info level, passing ctx through so a context-aware
// handler can attach the active span's trace_id/span_id. The sibling *Context
// methods behave the same at their respective levels.
func (a *SlogAdapter) InfoContext(ctx context.Context, msg string, args ...any) {
	a.log.LogAttrs(ctx, slog.LevelInfo, msg, toSlogAttrs(args)...)
}

func (a *SlogAdapter) ErrorContext(ctx context.Context, msg string, args ...any) {
	a.log.LogAttrs(ctx, slog.LevelError, msg, toSlogAttrs(args)...)
}

func (a *SlogAdapter) DebugContext(ctx context.Context, msg string, args ...any) {
	a.log.LogAttrs(ctx, slog.LevelDebug, msg, toSlogAttrs(args)...)
}

func (a *SlogAdapter) WarnContext(ctx context.Context, msg string, args ...any) {
	a.log.LogAttrs(ctx, slog.LevelWarn, msg, toSlogAttrs(args)...)
}

// With returns a child logger that tags every record with args, given as
// key/value pairs.
func (a *SlogAdapter) With(args ...any) Logger {
	return &SlogAdapter{log: a.log.With(args...)}
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
