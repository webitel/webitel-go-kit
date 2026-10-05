package depenlog

import (
	"context"
	"log/slog"
	"slices"

	"github.com/webitel/webitel-go-kit/pkg/semconv"
	"go.opentelemetry.io/otel/trace"
)

// traceHandler decorates a slog.Handler, copying the active span's trace_id and
// span_id from the context into every record. This is what makes log lines
// correlatable with traces — and across services — without callers passing the
// IDs by hand, provided they log through the *Context methods (so a real
// context, not context.Background(), reaches Handle).
//
// The trace IDs are always emitted at the top level, even when the caller has
// opened groups via WithGroup: a single Loki/ELK query on trace_id keeps working
// regardless of grouping. To do that without paying for any reconstruction on
// the common path, pre-group WithAttrs are folded into base (preformatted once),
// and only the operations after the first WithGroup are replayed per record.
type traceHandler struct {
	base slog.Handler // root formatter; carries pre-group WithAttrs, never our groups
	mods []mod        // WithGroup/WithAttrs recorded after the first group, in order
}

// mod is one deferred WithGroup or WithAttrs call. Exactly one field is set.
type mod struct {
	group string      // non-empty => WithGroup(group)
	attrs []slog.Attr // non-nil => WithAttrs(attrs)
}

func (h traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	sc := trace.SpanContextFromContext(ctx)
	hasSpan := sc.IsValid()

	// Fast path: no groups opened, so the record's attrs are already at the top
	// level — append the trace IDs alongside them, same cost as a plain handler.
	if len(h.mods) == 0 {
		if hasSpan {
			r.AddAttrs(traceAttrs(sc)...)
		}
		return h.base.Handle(ctx, r)
	}

	// Group path: nest the record's own attrs (and any post-group WithAttrs)
	// under the opened groups, then emit through the ungrouped base so the trace
	// IDs land at the top level rather than inside the group.
	var own []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		own = append(own, a)
		return true
	})

	nr := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	if hasSpan {
		nr.AddAttrs(traceAttrs(sc)...)
	}
	nr.AddAttrs(scopeAttrs(h.mods, own)...)
	return h.base.Handle(ctx, nr)
}

// traceAttrs returns the top-level correlation attributes for sc.
func traceAttrs(sc trace.SpanContext) []slog.Attr {
	return []slog.Attr{
		slog.String(semconv.TraceIDKey, sc.TraceID().String()),
		slog.String(semconv.SpanIDKey, sc.SpanID().String()),
	}
}

func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	// No group yet: fold into the base handler so it is preformatted once.
	if len(h.mods) == 0 {
		return traceHandler{base: h.base.WithAttrs(attrs)}
	}
	return traceHandler{base: h.base, mods: appendMod(h.mods, mod{attrs: attrs})}
}

func (h traceHandler) WithGroup(name string) slog.Handler {
	if name == "" { // slog spec: an empty group name is a no-op.
		return h
	}
	return traceHandler{base: h.base, mods: appendMod(h.mods, mod{group: name})}
}

// scopeAttrs replays the WithGroup/WithAttrs mods onto own (the record's own
// attributes), returning the resulting top-level attrs. It walks mods in
// reverse: a WithAttrs prepends at the current scope; a WithGroup wraps the
// accumulator into a single group attribute.
func scopeAttrs(mods []mod, own []slog.Attr) []slog.Attr {
	acc := own
	for i := len(mods) - 1; i >= 0; i-- {
		if g := mods[i].group; g != "" {
			acc = []slog.Attr{{Key: g, Value: slog.GroupValue(acc...)}}
			continue
		}
		acc = slices.Concat(mods[i].attrs, acc)
	}
	return acc
}

// appendMod returns a fresh slice so sibling handlers never alias each other's
// mods backing array.
func appendMod(mods []mod, m mod) []mod {
	out := make([]mod, len(mods)+1)
	copy(out, mods)
	out[len(mods)] = m
	return out
}
