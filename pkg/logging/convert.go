package logging

import (
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

const (
	maxDepth         = 16
	maxDepthExceeded = "<max-depth-exceeded>"
)

// AttrToKeyValue converts an [slog.Attr] to an [attribute.KeyValue].
// Groups become maps and [slog.LogValuer] values are resolved at any nesting depth.
func AttrToKeyValue(a slog.Attr) attribute.KeyValue {
	return attribute.KeyValue{Key: attribute.Key(a.Key), Value: fromSlog(a.Value, 0)}
}

// KeyValueToAttr converts an [attribute.KeyValue] to an [slog.Attr]. Maps
// become groups and typed slices become slices of the same Go type.
// A round trip through [AttrToKeyValue] is exact for scalars, maps and typed
// slices; a heterogeneous slice keeps its elements but becomes []any.
func KeyValueToAttr(kv attribute.KeyValue) slog.Attr {
	return slog.Attr{Key: string(kv.Key), Value: slogValue(kv.Value)}
}

func slogValue(v attribute.Value) slog.Value {
	switch v.Type() {
	case attribute.BOOL:
		return slog.BoolValue(v.AsBool())
	case attribute.INT64:
		return slog.Int64Value(v.AsInt64())
	case attribute.FLOAT64:
		return slog.Float64Value(v.AsFloat64())
	case attribute.STRING:
		return slog.StringValue(v.AsString())
	case attribute.BOOLSLICE:
		return slog.AnyValue(v.AsBoolSlice())
	case attribute.INT64SLICE:
		return slog.AnyValue(v.AsInt64Slice())
	case attribute.FLOAT64SLICE:
		return slog.AnyValue(v.AsFloat64Slice())
	case attribute.STRINGSLICE:
		return slog.AnyValue(v.AsStringSlice())
	case attribute.BYTESLICE:
		return slog.AnyValue(v.AsByteSlice())
	case attribute.SLICE:
		items := v.AsSlice()
		out := make([]any, len(items))
		for i, item := range items {
			out[i] = slogValue(item).Any()
		}
		return slog.AnyValue(out)
	case attribute.MAP:
		kvs := v.AsMap()
		attrs := make([]slog.Attr, len(kvs))
		for i, kv := range kvs {
			attrs[i] = KeyValueToAttr(kv)
		}
		return slog.GroupValue(attrs...)
	default:
		return slog.Value{}
	}
}

func otelAttr(kv attribute.KeyValue) slog.Attr {
	switch kv.Value.Type() {
	case attribute.BOOL, attribute.INT64, attribute.FLOAT64, attribute.STRING:
		return KeyValueToAttr(kv)
	default:
		return slog.Attr{Key: string(kv.Key), Value: slog.AnyValue(kv.Value)}
	}
}

func fromSlog(sv slog.Value, depth int) attribute.Value {
	if depth > maxDepth {
		return attribute.StringValue(maxDepthExceeded)
	}

	sv = sv.Resolve()
	switch sv.Kind() {
	case slog.KindString:
		return attribute.StringValue(sv.String())
	case slog.KindInt64:
		return attribute.Int64Value(sv.Int64())
	case slog.KindUint64:
		return uintValue(sv.Uint64())
	case slog.KindFloat64:
		return attribute.Float64Value(sv.Float64())
	case slog.KindBool:
		return attribute.BoolValue(sv.Bool())
	case slog.KindDuration:
		return attribute.Int64Value(sv.Duration().Nanoseconds())
	case slog.KindTime:
		return attribute.Int64Value(sv.Time().UnixNano())
	case slog.KindGroup:
		return attribute.MapValue(groupAttrs(sv.Group(), depth+1)...)
	case slog.KindAny:
		return valueOf(sv.Any(), depth)
	default:
		return attribute.StringValue(sv.String())
	}
}

func groupAttrs(attrs []slog.Attr, depth int) []attribute.KeyValue {
	out := make([]attribute.KeyValue, 0, len(attrs))
	for _, a := range attrs {
		out = appendSlogAttr(out, a, depth)
	}
	return out
}

func appendSlogAttr(dst []attribute.KeyValue, a slog.Attr, depth int) []attribute.KeyValue {
	v := a.Value.Resolve()
	if a.Key == "" {
		if v.Kind() == slog.KindGroup {
			for _, inner := range v.Group() {
				dst = appendSlogAttr(dst, inner, depth)
			}
			return dst
		}
		if v.Kind() == slog.KindAny && v.Any() == nil {
			return dst
		}
	}
	if v.Kind() == slog.KindGroup && len(v.Group()) == 0 {
		return dst
	}

	attr := attribute.KeyValue{
		Key:   attribute.Key(a.Key),
		Value: fromSlog(v, depth),
	}
	return append(dst, attr)
}

func uintValue(u uint64) attribute.Value {
	if u > math.MaxInt64 {
		return attribute.StringValue(strconv.FormatUint(u, 10))
	}
	return attribute.Int64Value(int64(u))
}

func valueOf(v any, depth int) attribute.Value {
	if depth > maxDepth {
		return attribute.StringValue(maxDepthExceeded)
	}

	switch x := v.(type) {
	case nil:
		return attribute.Value{}
	case attribute.Value:
		return x
	case slog.Value:
		return fromSlog(x, depth)
	case slog.LogValuer:
		return fromSlog(slog.AnyValue(x), depth)
	case string:
		return attribute.StringValue(x)
	case bool:
		return attribute.BoolValue(x)
	case int:
		return attribute.Int64Value(int64(x))
	case int8:
		return attribute.Int64Value(int64(x))
	case int16:
		return attribute.Int64Value(int64(x))
	case int32:
		return attribute.Int64Value(int64(x))
	case int64:
		return attribute.Int64Value(x)
	case uint:
		return uintValue(uint64(x))
	case uint8:
		return attribute.Int64Value(int64(x))
	case uint16:
		return attribute.Int64Value(int64(x))
	case uint32:
		return attribute.Int64Value(int64(x))
	case uint64:
		return uintValue(x)
	case float32:
		return attribute.Float64Value(float64(x))
	case float64:
		return attribute.Float64Value(x)
	case time.Duration:
		return attribute.Int64Value(x.Nanoseconds())
	case time.Time:
		return attribute.Int64Value(x.UnixNano())
	case []byte:
		return attribute.ByteSliceValue(x)
	case []string:
		return attribute.StringSliceValue(x)
	case []bool:
		return attribute.BoolSliceValue(x)
	case []int:
		return attribute.IntSliceValue(x)
	case []int64:
		return attribute.Int64SliceValue(x)
	case []float64:
		return attribute.Float64SliceValue(x)
	case error:
		if isNilPtr(x) {
			return attribute.Value{}
		}
		return attribute.StringValue(safeString(x.Error))
	case fmt.Stringer:
		if isNilPtr(x) {
			return attribute.Value{}
		}
		return attribute.StringValue(safeString(x.String))
	}
	return reflectValue(reflect.ValueOf(v), depth)
}

func isNilPtr(v any) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

func safeString(f func() string) (s string) {
	defer func() {
		if r := recover(); r != nil {
			s = fmt.Sprintf("!PANIC: %v", r)
		}
	}()
	return f()
}

func reflectValue(rv reflect.Value, depth int) attribute.Value {
	switch rv.Kind() {
	case reflect.String:
		return attribute.StringValue(rv.String())
	case reflect.Bool:
		return attribute.BoolValue(rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return attribute.Int64Value(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintValue(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return attribute.Float64Value(rv.Float())
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return attribute.Value{}
		}
		return valueOf(rv.Elem().Interface(), depth+1)
	case reflect.Map:
		kvs := make([]attribute.KeyValue, 0, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			k := iter.Key()
			key := fmt.Sprint(k.Interface())
			if k.Kind() == reflect.String {
				key = k.String()
			}
			kvs = append(kvs, attribute.KeyValue{Key: attribute.Key(key), Value: valueOf(iter.Value().Interface(), depth+1)})
		}
		return attribute.MapValue(kvs...)
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return attribute.Value{}
		}
		items := make([]attribute.Value, rv.Len())
		for i := range items {
			items[i] = valueOf(rv.Index(i).Interface(), depth+1)
		}
		return attribute.SliceValue(items...)
	default:
		return attribute.StringValue(fmt.Sprintf("%+v", rv.Interface()))
	}
}
