package logging

import (
	"context"
	"log/slog"
)

// LevelHandler is a [slog.Handler] that drops records below Level before
// asking the wrapped Handler. A nil Level means [LevelInfo].
type LevelHandler struct {
	Level slog.Leveler
	slog.Handler
}

var _ slog.Handler = LevelHandler{}

func NewLevelHandler(level slog.Leveler, h slog.Handler) LevelHandler {
	if lh, ok := h.(LevelHandler); ok {
		h = lh.Handler
	}
	return LevelHandler{Level: level, Handler: h}
}

func (h LevelHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	if lvl < h.minLevel() {
		return false
	}
	return h.Handler.Enabled(ctx, lvl)
}

func (h LevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return LevelHandler{Level: h.Level, Handler: h.Handler.WithAttrs(attrs)}
}

func (h LevelHandler) WithGroup(name string) slog.Handler {
	return LevelHandler{Level: h.Level, Handler: h.Handler.WithGroup(name)}
}

func (h LevelHandler) minLevel() slog.Level {
	if h.Level == nil {
		return slog.LevelInfo
	}
	return h.Level.Level()
}
