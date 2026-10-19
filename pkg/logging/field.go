package logging

import (
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/webitel/webitel-go-kit/pkg/logging/redact"
)

type fieldKind uint8

const (
	kindValue fieldKind = iota
	kindGroup
	kindAny
	kindAttr
	kindKV
	kindEvent
	kindErr
	kindMap
)

const (
	errorKey    = "error"
	maxErrNodes = 32
)

type Field struct {
	key   string
	val   slog.Value
	kind  fieldKind
	class redact.Class
}

type Class = redact.Class

func Str(key, v string) Field {
	return Field{
		key: key,
		val: slog.StringValue(v),
	}
}

func Int(key string, v int) Field {
	return Field{
		key: key,
		val: slog.Int64Value(int64(v)),
	}
}

func Int64(key string, v int64) Field {
	return Field{
		key: key,
		val: slog.Int64Value(v),
	}
}

func Uint64(key string, v uint64) Field {
	return Field{
		key: key,
		val: slog.Uint64Value(v),
	}
}

func Float64(key string, v float64) Field {
	return Field{key: key, val: slog.Float64Value(v)}
}

func Bool(key string, v bool) Field {
	return Field{
		key: key,
		val: slog.BoolValue(v),
	}
}

func Time(key string, v time.Time) Field {
	return Field{
		key: key,
		val: slog.TimeValue(v),
	}
}

func Dur(key string, v time.Duration) Field {
	return Field{
		key: key,
		val: slog.DurationValue(v),
	}
}

func Err(err error) Field {
	if err == nil {
		return Field{}
	}

	return Field{
		key:  errorKey,
		val:  slog.AnyValue(err),
		kind: kindErr,
	}
}

func Group(key string, fields ...Field) Field {
	return Field{
		key:  key,
		val:  slog.AnyValue(fields),
		kind: kindGroup,
	}
}

func Any(key string, v any) Field {
	return Field{
		key:  key,
		val:  slog.AnyValue(v),
		kind: kindAny,
	}
}

func Attr(a slog.Attr) Field {
	return Field{
		key:  a.Key,
		val:  a.Value,
		kind: kindAttr,
	}
}

func KV(kv ...attribute.KeyValue) Field {
	return Field{
		val:  slog.AnyValue(kv),
		kind: kindKV,
	}
}

func Ev(e Event) Field {
	if e == nil {
		return Field{}
	}

	return Field{
		val:  slog.AnyValue(e),
		kind: kindEvent,
	}
}

func Map[V any](key string, m map[string]V, classOf func(string) Class) Field {
	return Field{
		key: key,
		val: slog.AnyValue(mapOf[V]{
			m:       m,
			classOf: classOf,
		}),
		kind: kindMap,
	}
}

func (f Field) PII() Field {
	f.class = redact.ClassPII
	return f
}

func (f Field) Secret() Field {
	f.class = redact.ClassSecret
	return f
}

type mapper interface {
	attributes(depth int) []attribute.KeyValue
}

type mapOf[V any] struct {
	m       map[string]V
	classOf func(string) Class
}

func (m mapOf[V]) attributes(depth int) []attribute.KeyValue {
	out := make([]attribute.KeyValue, 0, len(m.m))
	for k, v := range m.m {
		val := valueOf(v, depth+1)
		if m.classOf != nil {
			var keep bool
			if val, keep = RedactValue(m.classOf(k), val); !keep {
				continue
			}
		}

		kv := attribute.KeyValue{
			Key:   attribute.Key(k),
			Value: val,
		}
		out = append(out, kv)
	}
	return out
}
