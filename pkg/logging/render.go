package logging

import (
	"fmt"
	"log/slog"
	"slices"

	"go.opentelemetry.io/otel/attribute"

	"github.com/webitel/webitel-go-kit/pkg/logging/redact"
)

type entry struct {
	attrs      []attribute.KeyValue
	eventAttrs []attribute.KeyValue
	eventName  string
	hasEvent   bool
	err        error
}

func (e *entry) render(out []attribute.KeyValue, fields []Field) []attribute.KeyValue {
	for _, f := range fields {
		out = e.renderField(out, f)
	}
	return out
}

func (e *entry) renderGroup(out []attribute.KeyValue, fields []Field) []attribute.KeyValue {
	for _, f := range fields {
		if f.kind == kindEvent {
			continue
		}
		out = e.renderField(out, f)
	}
	return out
}

func (e *entry) renderField(out []attribute.KeyValue, f Field) []attribute.KeyValue {
	switch f.kind {
	case kindValue:
		if f.val.Kind() == slog.KindAny && f.val.Any() == nil {
			return out
		}
		return appendValue(out, f.key, fromSlog(f.val, 0), f.class)
	case kindGroup:
		children, _ := f.val.Any().([]Field)
		if f.key == "" && f.class == redact.ClassRegular {
			return e.renderGroup(out, children)
		}
		inner := e.renderGroup(nil, children)
		if len(inner) == 0 {
			return out
		}
		return appendValue(out, f.key, attribute.MapValue(inner...), f.class)
	case kindAny:
		if f.key == "" && f.val.Any() == nil {
			return out
		}
		return appendValue(out, f.key, valueOf(f.val.Any(), 0), f.class)
	case kindAttr:
		if f.class == redact.ClassRegular {
			return appendSlogAttr(out, slog.Attr{Key: f.key, Value: f.val}, 0)
		}
		return appendValue(out, f.key, fromSlog(f.val, 0), f.class)
	case kindKV:
		kvs, _ := f.val.Any().([]attribute.KeyValue)
		for _, kv := range kvs {
			out = appendValue(out, string(kv.Key), kv.Value, f.class)
		}
		return out
	case kindEvent:
		ev := f.val.Any().(Event)
		e.hasEvent = true
		e.eventName = ev.EventName()
		e.eventAttrs = append(e.eventAttrs, ev.Attributes()...)
		return out
	case kindErr:
		err := f.val.Any().(error)
		if f.class == redact.ClassRegular {
			e.err = err
		}
		return appendValue(out, f.key, errorValue(err), f.class)
	case kindMap:
		m := f.val.Any().(mapper)
		return appendValue(out, f.key, attribute.MapValue(m.attributes(0)...), f.class)
	default:
		return out
	}
}

func appendValue(out []attribute.KeyValue, key string, v attribute.Value, c redact.Class) []attribute.KeyValue {
	v, keep := redactValue(c, v)
	if !keep {
		return out
	}
	return append(out, attribute.KeyValue{Key: attribute.Key(key), Value: v})
}

func redactValue(c redact.Class, v attribute.Value) (attribute.Value, bool) {
	if c == redact.ClassRegular {
		return v, true
	}
	s, keep := redact.Render(c, valueText(v), isEmpty(v))
	if !keep {
		return attribute.Value{}, false
	}
	return attribute.StringValue(s), true
}

func valueText(v attribute.Value) string {
	if v.Type() == attribute.BYTESLICE {
		return string(v.AsByteSlice())
	}
	return v.String()
}

func isEmpty(v attribute.Value) bool {
	switch v.Type() {
	case attribute.EMPTY:
		return true
	case attribute.INT64:
		return v.AsInt64() == 0
	case attribute.FLOAT64:
		return v.AsFloat64() == 0
	case attribute.STRING:
		return v.AsString() == ""
	case attribute.BYTESLICE:
		return len(v.AsByteSlice()) == 0
	case attribute.BOOLSLICE:
		return len(v.AsBoolSlice()) == 0
	case attribute.INT64SLICE:
		return len(v.AsInt64Slice()) == 0
	case attribute.FLOAT64SLICE:
		return len(v.AsFloat64Slice()) == 0
	case attribute.STRINGSLICE:
		return len(v.AsStringSlice()) == 0
	case attribute.SLICE:
		return len(v.AsSlice()) == 0
	case attribute.MAP:
		return len(v.AsMap()) == 0
	default:
		return false
	}
}

func errorValue(err error) attribute.Value {
	kvs := []attribute.KeyValue{
		attribute.String("type", fmt.Sprintf("%T", err)),
		attribute.String("message", safeString(err.Error)),
	}
	seen := map[attribute.Key]struct{}{
		"type":    {},
		"message": {},
		"chain":   {},
	}

	var chain []string
	traverseErr(err, func(e error) {
		msg := safeString(e.Error)
		if len(chain) == 0 || chain[len(chain)-1] != msg {
			chain = append(chain, msg)
		}

		lv, ok := e.(slog.LogValuer)
		if !ok || isNilPtr(e) {
			return
		}

		v := slog.AnyValue(lv).Resolve()
		var values []attribute.KeyValue
		if v.Kind() == slog.KindGroup {
			values = groupAttrs(v.Group(), 1)
		} else {
			values = []attribute.KeyValue{{Key: "value", Value: fromSlog(v, 1)}}
		}
		for _, kv := range values {
			if _, dup := seen[kv.Key]; dup {
				continue
			}
			seen[kv.Key] = struct{}{}
			kvs = append(kvs, kv)
		}
	})
	if len(chain) > 1 {
		kvs = append(kvs, attribute.StringSlice("chain", chain))
	}
	return attribute.MapValue(kvs...)
}

func traverseErr(err error, visit func(error)) {
	stack := []error{err}
	for n := 0; len(stack) > 0 && n < maxErrNodes; {
		e := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if e == nil {
			continue
		}

		n++
		visit(e)
		if isNilPtr(e) {
			continue
		}

		switch u := e.(type) {
		case interface{ Unwrap() []error }:
			errs := u.Unwrap()
			for _, wrappedErr := range slices.Backward(errs) {
				stack = append(stack, wrappedErr)
			}
		case interface{ Unwrap() error }:
			stack = append(stack, u.Unwrap())
		}
	}
}
