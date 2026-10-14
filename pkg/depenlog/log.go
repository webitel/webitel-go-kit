// Package depenlog builds the kit's unified logger: a single slog-based handler that
// emits one record schema (field names from pkg/semconv), auto-attaches the
// active span's trace_id/span_id, and funnels third-party logs (grpc-go, fx,
// HTTP, the standard library log package) through that same handler so a single
// Loki/ELK query works across every service.
package depenlog

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/webitel/webitel-go-kit/pkg/logger"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config controls how the unified logger is built. It mirrors the fields of
// appconfig.Log without importing it, keeping pkg/depenlog free of the (heavy)
// configuration dependency; callers map their appconfig.Log onto it.
type Config struct {
	Level   string // debug|info|warn|error (default: info)
	JSON    bool   // emit JSON; otherwise human-readable text
	File    string // optional file path to also write to (rotated, see below)
	Console bool   // write to stdout

	// File rotation, applied when File != ""; zero values use lumberjack
	// defaults (≈100 MB per file, keep all backups, no compression).
	MaxSizeMB  int  // rotate after this many megabytes
	MaxBackups int  // max rotated files to retain (0 = keep all)
	MaxAgeDays int  // max days to retain rotated files (0 = keep forever)
	Compress   bool // gzip rotated files
}

type options struct {
	handler slog.Handler
}

// Option customizes New.
type Option func(*options)

// WithHandler replaces the base slog handler entirely. Use it to route logs
// through an OpenTelemetry bridge (so the OTel LoggerProvider/exporter owns the
// schema and trace correlation) or any other sink. When set, Config's
// JSON/File/Console fields and the built-in trace/semconv decorators are
// bypassed — the provided handler is wired in as-is, and slog default + grpc-go
// (and any FxLogger/Middleware you attach) still flow through it.
func WithHandler(h slog.Handler) Option {
	return func(o *options) { o.handler = h }
}

// New builds the unified logger from cfg and installs it process-wide:
//   - slog.SetDefault, so slog.* and the standard library log package share it;
//   - grpc-go's global logger, via UseGRPC.
//
// The returned logger.Logger is the handle services inject. Per-app wiring for
// fx and HTTP is explicit, through FxLogger / ErrorLog / Middleware.
func New(cfg Config, opts ...Option) logger.Logger {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	h := o.handler
	if h == nil {
		h = buildPlainHandler(writer(cfg), cfg)
	}

	sl := slog.New(h)
	slog.SetDefault(sl)

	l := logger.NewSlog(sl)
	UseGRPC(l)
	return l
}

// buildPlainHandler assembles the default handler chain: a JSON or text base
// handler that renames fields to semconv keys, wrapped by traceHandler so
// trace_id/span_id are attached from context. Kept separate from New so tests
// can target a buffer without touching process-wide state.
func buildPlainHandler(w io.Writer, cfg Config) slog.Handler {
	opts := &slog.HandlerOptions{
		Level:       parseLevel(cfg.Level),
		ReplaceAttr: replaceAttr,
	}
	var base slog.Handler
	if cfg.JSON {
		base = slog.NewJSONHandler(w, opts)
	} else {
		base = slog.NewTextHandler(w, opts)
	}
	return traceHandler{base: base}
}

// writer resolves the output sink from cfg: stdout and/or a rotated file. It
// never returns nil — if neither is configured it falls back to stdout so logs
// are never silently dropped. File output is rotated via lumberjack.
func writer(cfg Config) io.Writer {
	var ws []io.Writer
	if cfg.Console {
		ws = append(ws, os.Stdout)
	}
	if cfg.File != "" {
		ws = append(ws, &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
		})
	}
	switch len(ws) {
	case 0:
		return os.Stdout
	case 1:
		return ws[0]
	default:
		return io.MultiWriter(ws...)
	}
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// replaceAttr renames slog's built-in keys to the kit's semantic-convention
// keys and normalizes the common "err" misspelling to the canonical error key,
// so every service emits the same field names.
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	// Normalize error values to their message string. slog's JSON handler would
	// otherwise marshal most errors to "{}" (no exported fields), losing the
	// message — so every service logs errors the same, readable way.
	if err, ok := a.Value.Any().(error); ok {
		a.Value = slog.StringValue(err.Error())
	}
	if len(groups) != 0 {
		return a
	}
	switch a.Key {
	case slog.TimeKey:
		a.Key = semconv.TimestampKey
	case slog.MessageKey:
		a.Key = semconv.MessageKey
	case slog.LevelKey:
		a.Key = semconv.LevelKey
	case "err":
		a.Key = semconv.ErrorKey
	}
	return a
}
