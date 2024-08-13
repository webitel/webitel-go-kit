package logr

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

func NewLogger(name string, opts ...Option) logr.Logger {
	conf := newOptions(opts...)
	return logr.New(&Handler{
		opts:   conf,
		logger: conf.logger(name),
	})
}

type Options struct {
	Provider  log.LoggerProvider
	Version   string
	SchemaURL string

	SeverityError int
}

type Option func(c *Options)

func newOptions(opts ...Option) (conf Options) {
	conf = Options{
		Provider:      global.GetLoggerProvider(),
		SeverityError: int(log.SeverityError),
	}
	for _, opt := range opts {
		opt(&conf)
	}
	return
}

func (c Options) logger(name string) log.Logger {
	var opts []log.LoggerOption
	if c.Version != "" {
		opts = append(opts, log.WithInstrumentationVersion(c.Version))
	}
	if c.SchemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.SchemaURL))
	}
	return c.Provider.Logger(name, opts...)
}

type Handler struct {
	opts Options

	attrs  *kvBuffer
	logger log.Logger
}

var _ logr.LogSink = (*Handler)(nil)

// Init receives optional information about the logr library for LogSink
// implementations that need it.
func (h *Handler) Init(info logr.RuntimeInfo) {}

// Enabled tests whether this LogSink is enabled at the specified V-level.
// For example, commandline flags might be used to set the logging
// verbosity and disable some info logs.
func (h *Handler) Enabled(level int) bool {
	var emit log.Record
	emit.SetSeverity(log.Severity(level))
	return h.logger.Enabled(context.Background(), emit)
}

func (h *Handler) convertRecord(level int, msg string, keysAndValues []any) log.Record {
	var (
		record    log.Record
		timestamp = time.Now()
	)

	record.SetTimestamp(timestamp)
	record.SetBody(log.StringValue(msg))

	// const sevOffset = slog.Level(log.SeverityDebug) - slog.LevelDebug
	// record.SetSeverity(log.Severity(r.Level + sevOffset))
	record.SetSeverity(log.Severity(level))

	if h.attrs.Len() > 0 {
		record.AddAttributes(h.attrs.KeyValues()...)
	}

	n := len(keysAndValues)
	if n%2 == 1 {
		// keysAndValues = append(keysAndValues, nil)
		// n++
		keysAndValues = keysAndValues[0 : n-1]
		n--
	}
	n = n / 2
	if n > 0 {
		buf, free := getKVBuffer()
		defer free()
		// r.Attrs(buf.AddAttr)
		buf.AddKeysAndValues(keysAndValues...)
		record.AddAttributes(buf.KeyValues()...)
	}

	// attrs := keysAndValues(keysAndValues)
	// n := h.attrs.Len() // r.NumAttrs()
	// if h.group != nil {
	// 	if n > 0 {
	// 		buf, free := getKVBuffer()
	// 		defer free()
	// 		r.Attrs(buf.AddAttr)
	// 		record.AddAttributes(h.group.KeyValue(buf.KeyValues()...))
	// 	} else {
	// 		// A Handler should not output groups if there are no attributes.
	// 		g := h.group.NextNonEmpty()
	// 		if g != nil {
	// 			record.AddAttributes(g.KeyValue())
	// 		}
	// 	}
	// } else if n > 0 {
	// 	buf, free := getKVBuffer()
	// 	defer free()
	// 	r.Attrs(buf.AddAttr)
	// 	record.AddAttributes(buf.KeyValues()...)
	// }

	return record
}

// Info logs a non-error message with the given key/value pairs as context.
// The level argument is provided for optional logging.
// This method will only be called when Enabled(level) is true.
// See [logr.Logger.Info] for more details.
func (h *Handler) Info(level int, msg string, keysAndValues ...any) {
	h.logger.Emit(context.Background(), h.convertRecord(
		level, msg, keysAndValues,
	))
}

// Error logs an error, with the given message
// and key/value pairs as context.
// See [logr.Logger.Error] for more details.
func (h *Handler) Error(err error, msg string, keysAndValues ...any) {
	h.logger.Emit(context.Background(), h.convertRecord(
		h.opts.SeverityError, msg, append(keysAndValues, "error", err),
	))
}

// WithValues returns a new LogSink with additional key/value pairs.
// See [logr.Logger.WithValues] for more details.
func (h *Handler) WithValues(keysAndValues ...any) logr.LogSink {
	h2 := *h
	if h2.attrs != nil {
		h2.attrs = h2.attrs.Clone()
		h2.attrs.AddKeysAndValues(keysAndValues...)
	} else {
		if h2.attrs == nil {
			n := len(keysAndValues)
			n = (n / 2) + (n % 2)
			h2.attrs = newKVBuffer(n)
		} else {
			h2.attrs = h2.attrs.Clone()
		}
		h2.attrs.AddKeysAndValues(keysAndValues...)
	}
	return &h2
}

// WithName returns a new LogSink with the specified name appended.
// See [logr.Logger.WithName] for more details.
func (h *Handler) WithName(name string) logr.LogSink {
	h2 := *h
	h2.logger = h.opts.logger(name)
	return &h2
}
