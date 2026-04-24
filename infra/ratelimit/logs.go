package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

// Logger type shorthand
type Logger = slog.Logger

// Log helper method
func Log(ctx context.Context, out *Logger, level slog.Level, msg string, args ...any) {
	if out == nil {
		return // not available
	}
	if !out.Enabled(ctx, level) {
		return // skip args evaluation..
	}
	// msg = "[ RATE:LIMIT ] " + msg
	out.Log(ctx, level, msg, args...)
}

// Log from Request context
func (req *Request) Log(level slog.Level, msg string, args ...any) {
	Log(req.Context, req.Logger, level, msg, args...)
}

// LogValue used to defer evaluations until they are needed,
// or to expand a single value into a sequence of components.
type LogValue func() slog.Value

var _ slog.LogValuer = LogValue(nil)

func (fn LogValue) LogValue() slog.Value {
	return fn()
}

// LogValue implements slog.Valuer interface
func (res *Status) LogValue() slog.Value {
	status := "forbidden"
	if res == nil {
		return slog.StringValue(status)
	}
	if res.Allowed > 0 {
		status = "OK" // passthrough
	}
	params := []slog.Attr{
		// slog.String("msec", res.Date.Format(":05.000")),                       // Time precision (in milliseconds) ; Date from log record
		slog.Int("allow", int(res.Allowed)),                                   // consumed token(-s) count ; cost of the request
		slog.String("remain", fmt.Sprintf("%d/%d", res.Remaining, res.Limit)), // more tokens left in bucket after been consumed
		slog.Duration("reset", res.ResetAfter.Round(time.Millisecond)),        // tokens count (according to the limit rate) will be fully refreshed in ..
	}
	if res.RetryAfter > 0 {
		status = "flood_wait_" + strconv.FormatInt(MinSeconds(res.RetryAfter), 10) // flood_wait_$(sec)
		params = append(params, slog.Duration("retry", res.RetryAfter.Round(time.Millisecond)))
	}
	params = append(params, slog.String("status", status))
	return slog.GroupValue(params...)
}

var noLogs = slog.New(slog.DiscardHandler)

func LogModule(module string, logger *slog.Logger) *slog.Logger {
	h := logger.Handler()
	h2 := LogHandler(module, h)
	if h2 == h {
		return logger
	}
	return slog.New(h2)
}

func LogHandler(module string, handler slog.Handler) slog.Handler {
	module = strings.TrimSpace(module)
	if module == "" {
		return handler
	}
	module += " " // indent: default
	// module = "[ " + module + " ] "
	// is already desired handler ?
	wrap, _ := handler.(prefixLogHandler)
	if wrap.prefix == module {
		// duplicate: return original
		return handler
	}
	if wrap.Handler != nil {
		// IS handler.(prefixLogHandler) !
		// Extract source [slog.Handler] to override [module] prefix ..
		handler = wrap.Handler
	}
	return prefixLogHandler{prefix: module, Handler: handler}
}

// Used to add extra prefix for each Log record message
type prefixLogHandler struct {
	prefix string
	slog.Handler
}

var _ slog.Handler = prefixLogHandler{}

// Enabled reports whether the handler handles records at the given level.
// The handler ignores records whose level is lower.
// It is called early, before any arguments are processed,
// to save effort if the log event should be discarded.
// If called from a Logger method, the first argument is the context
// passed to that method, or context.Background() if nil was passed
// or the method does not take a context.
// The context is passed so Enabled can use its values
// to make a decision.
func (h prefixLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Handler.Enabled(ctx, level)
}

// Handle handles the Record.
// It will only be called when Enabled returns true.
// The Context argument is as for Enabled.
// It is present solely to provide Handlers access to the context's values.
// Canceling the context should not affect record processing.
// (Among other things, log messages may be necessary to debug a
// cancellation-related problem.)
//
// Handle methods that produce output should observe the following rules:
//   - If r.Time is the zero time, ignore the time.
//   - If r.PC is zero, ignore it.
//   - Attr's values should be resolved.
//   - If an Attr's key and value are both the zero value, ignore the Attr.
//     This can be tested with attr.Equal(Attr{}).
//   - If a group's key is empty, inline the group's Attrs.
//   - If a group has no Attrs (even if it has a non-empty key),
//     ignore it.
//
// [Logger] discards any errors from Handle. Wrap the Handle method to
// process any errors from Handlers.
func (h prefixLogHandler) Handle(ctx context.Context, event slog.Record) error {
	if !strings.HasPrefix(event.Message, h.prefix) {
		event.Message = h.prefix + event.Message
	}
	return h.Handler.Handle(ctx, event)
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
// The Handler owns the slice: it may retain, modify or discard it.
func (h prefixLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return prefixLogHandler{prefix: h.prefix, Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
// The keys of all subsequent attributes, whether added by With or in a
// Record, should be qualified by the sequence of group names.
//
// How this qualification happens is up to the Handler, so long as
// this Handler's attribute keys differ from those of another Handler
// with a different sequence of group names.
//
// A Handler should treat WithGroup as starting a Group of Attrs that ends
// at the end of the log event. That is,
//
//	logger.WithGroup("s").LogAttrs(ctx, level, msg, slog.Int("a", 1), slog.Int("b", 2))
//
// should behave like
//
//	logger.LogAttrs(ctx, level, msg, slog.Group("s", slog.Int("a", 1), slog.Int("b", 2)))
//
// If the name is empty, WithGroup returns the receiver.
func (h prefixLogHandler) WithGroup(name string) slog.Handler {
	return prefixLogHandler{prefix: name, Handler: h.Handler.WithGroup(name)}
}
