/*
Copyright Agoda Services Co.,Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logr

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
)

func otelAttribute(attr slog.Attr) []attribute.KeyValue {
	switch attr.Value.Kind() {
	case slog.KindBool:
		return []attribute.KeyValue{attribute.Bool(attr.Key, attr.Value.Bool())}
	//case slog.KindDuration: ???
	case slog.KindFloat64:
		return []attribute.KeyValue{attribute.Float64(attr.Key, attr.Value.Float64())}
	case slog.KindInt64:
		return []attribute.KeyValue{attribute.Int64(attr.Key, attr.Value.Int64())}
	case slog.KindString:
		return []attribute.KeyValue{attribute.String(attr.Key, attr.Value.String())}
	//case slog.KindTime: ???
	case slog.KindUint64:
		return []attribute.KeyValue{attribute.Int64(attr.Key, int64(attr.Value.Uint64()))}
	case slog.KindGroup:
		group := attr.Value.Group()
		var result []attribute.KeyValue
		for _, v := range group {
			v.Key = attr.Key + "." + v.Key
			result = append(result, otelAttribute(v)...)
		}
		return result
	}
	return []attribute.KeyValue{attribute.String(attr.Key, attr.Value.String())}
}

var kvBufferPool = sync.Pool{
	New: func() any {
		// Based on slog research (https://go.dev/blog/slog#performance), 95%
		// of use-cases will use 5 or less attributes.
		return newKVBuffer(5)
	},
}

func getKVBuffer() (buf *kvBuffer, free func()) {
	buf = kvBufferPool.Get().(*kvBuffer)
	return buf, func() {
		// TODO: limit returned size so the pool doesn't hold on to very large
		// buffers. Idea is based on
		// https://cs.opensource.google/go/x/exp/+/814bf88c:slog/internal/buffer/buffer.go;l=27-34

		// Do not modify any previously held data.
		buf.data = buf.data[:0:0]
		kvBufferPool.Put(buf)
	}
}

type kvBuffer struct {
	data []log.KeyValue
}

func newKVBuffer(n int) *kvBuffer {
	return &kvBuffer{data: make([]log.KeyValue, 0, n)}
}

// Len returns the number of [log.KeyValue] held by b.
func (b *kvBuffer) Len() int {
	if b == nil {
		return 0
	}
	return len(b.data)
}

// Clone returns a copy of b.
func (b *kvBuffer) Clone() *kvBuffer {
	if b == nil {
		return nil
	}
	return &kvBuffer{data: slices.Clone(b.data)}
}

// KeyValues returns kvs appended to the [log.KeyValue] held by b.
func (b *kvBuffer) KeyValues(kvs ...log.KeyValue) []log.KeyValue {
	if b == nil {
		return kvs
	}
	return append(b.data, kvs...)
}

func keysAndValues(kvs []any) []log.KeyValue {
	n := len(kvs)
	if n%2 == 1 {
		kvs = append(kvs, nil)
		n++
	}
	n = n / 2
	attrs := make([]log.KeyValue, 0, n)
	for i := 0; i < n; i += 2 {
		attrs = append(attrs, log.KeyValue{
			Key:   fmt.Sprint(kvs[i]),
			Value: convertValue(kvs[i+1]),
		})
	}
	return attrs
}

func (b *kvBuffer) AddKeysAndValues(kvs ...any) {
	n := len(kvs)
	if n%2 == 1 {
		// kvs = append(kvs, nil)
		// n++
		// A Handler should ignore an empty Attr.
		kvs = kvs[:n-1]
		n--
	}
	n = n / 2
	b.data = slices.Grow(b.data, n)
	for i := 0; i < n; i += 2 {
		b.data = append(b.data, log.KeyValue{
			Key:   fmt.Sprint(kvs[i*2]),
			Value: convertValue(kvs[(i*2)+1]),
		})
	}
}

// AddAttrs adds attrs to b.
func (b *kvBuffer) AddAttrs(attrs []slog.Attr) {
	b.data = slices.Grow(b.data, len(attrs))
	for _, a := range attrs {
		_ = b.AddAttr(a)
	}
}

// AddAttr adds attr to b and returns true.
//
// This is designed to be passed to the AddAttributes method of an
// [slog.Record].
//
// If attr is a group with an empty key, its values will be flattened.
//
// If attr is empty, it will be dropped.
func (b *kvBuffer) AddAttr(attr slog.Attr) bool {
	if attr.Key == "" {
		if attr.Value.Kind() == slog.KindGroup {
			// A Handler should inline the Attrs of a group with an empty key.
			for _, a := range attr.Value.Group() {
				b.data = append(b.data, log.KeyValue{
					Key:   a.Key,
					Value: convertValue(a.Value),
				})
			}
			return true
		}

		if attr.Value.Any() == nil {
			// A Handler should ignore an empty Attr.
			return true
		}
	}
	b.data = append(b.data, log.KeyValue{
		Key:   attr.Key,
		Value: convertValue(attr.Value),
	})
	return true
}

// TODO: !!!!!!!
func convertValue(v any) log.Value {
	// func BoolValue(v bool) Value
	// func BytesValue(v []byte) Value
	// func Float64Value(v float64) Value
	// func Int64Value(v int64) Value
	// func IntValue(v int) Value
	// func MapValue(kvs ...KeyValue) Value
	// func SliceValue(vs ...Value) Value
	// func StringValue(v string) Value
	switch v := v.(type) {
	case bool:
		return log.BoolValue(v)
	case []byte:
		return log.BytesValue(v)
	case float64:
		return log.Float64Value(v)
	case int64:
		return log.Int64Value(v)
	case int:
		return log.IntValue(v)
	case string:
		return log.StringValue(v)
	case map[string]any:
		group := make([]log.KeyValue, len(v))
		for k, v := range v {
			group = append(group, log.KeyValue{
				Key:   k,
				Value: convertValue(v),
			})
		}
		return log.MapValue(group...)
	}
	return log.StringValue(fmt.Sprintf("%+v", v))
}

// func convertValue(v slog.Value) log.Value {
// 	switch v.Kind() {
// 	case slog.KindAny:
// 		return log.StringValue(fmt.Sprintf("%+v", v.Any()))
// 	case slog.KindBool:
// 		return log.BoolValue(v.Bool())
// 	case slog.KindDuration:
// 		return log.Int64Value(v.Duration().Nanoseconds())
// 	case slog.KindFloat64:
// 		return log.Float64Value(v.Float64())
// 	case slog.KindInt64:
// 		return log.Int64Value(v.Int64())
// 	case slog.KindString:
// 		return log.StringValue(v.String())
// 	case slog.KindTime:
// 		return log.Int64Value(v.Time().UnixNano())
// 	case slog.KindUint64:
// 		return log.Int64Value(int64(v.Uint64()))
// 	case slog.KindGroup:
// 		buf, free := getKVBuffer()
// 		defer free()
// 		buf.AddAttrs(v.Group())
// 		return log.MapValue(buf.data...)
// 	case slog.KindLogValuer:
// 		return convertValue(v.Resolve())
// 	default:
// 		// Try to handle this as gracefully as possible.
// 		//
// 		// Don't panic here. The goal here is to have developers find this
// 		// first if a new slog.Kind is added. A test on the new kind will find
// 		// this malformed attribute as well as a panic. However, it is
// 		// preferable to have user's open issue asking why their attributes
// 		// have a "unhandled: " prefix than say that their code is panicking.
// 		return log.StringValue(fmt.Sprintf("unhandled: (%s) %+v", v.Kind(), v.Any()))
// 	}
// }
