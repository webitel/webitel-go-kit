package logger

import "context"

// Logger is the kit's structured logging surface; implementations adapt a
// concrete backend (slog, wlog) behind it.
//
// The *Context variants carry a context.Context so context-aware backends can
// correlate records with the active span's trace_id/span_id. With returns a
// child logger that tags every subsequent record with the given key/value args.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)

	InfoContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
	DebugContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)

	With(args ...any) Logger
}
