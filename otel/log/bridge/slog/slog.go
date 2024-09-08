package slog

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

type Option = otelslog.Option

func New(name string, opts ...Option) *slog.Logger {
	return otelslog.NewLogger(name, opts...)
}

type LevelHandler struct {
	// Level reports the minimum record level that will be logged.
	// The handler discards records with lower levels.
	// If Level is nil, the handler assumes LevelInfo.
	// The handler calls Level.Level for each record processed;
	// to adjust the minimum level dynamically, use a LevelVar.
	Level slog.Leveler
	// Underlying slog.Handler
	slog.Handler
}

var _ slog.Handler = LevelHandler{}

// Enabled reports whether the handler handles records at the given level.
// The handler ignores records whose level is lower.
// It is called early, before any arguments are processed,
// to save effort if the log event should be discarded.
// If called from a Logger method, the first argument is the context
// passed to that method, or context.Background() if nil was passed
// or the method does not take a context.
// The context is passed so Enabled can use its values
// to make a decision.
func (lh LevelHandler) Enabled(ctx context.Context, v slog.Level) bool {
	// front: enabled(?)
	if min := lh.Level; min != nil {
		if v < min.Level() {
			return false
		}
	}
	// back: enabled(?)
	return lh.Handler.Enabled(ctx, v)
}

func WithLevel(front slog.Leveler, back slog.Handler) slog.Handler {
	if lh, is := back.(LevelHandler); is {
		if lh.Level.Level() == front.Level() {
			return back // nothing todo
		}
		// extract
		back = lh.Handler
	}
	return LevelHandler{
		Level:   front,
		Handler: back,
	}
}
