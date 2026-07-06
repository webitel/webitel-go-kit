package cache

import (
	"encoding/json"

	"google.golang.org/protobuf/proto"
)

// Codec handles serialization for L2 (Redis) storage.
type Codec[V any] interface {
	Marshal(v V) ([]byte, error)
	Unmarshal(data []byte) (V, error)
}

// JSON returns a codec that uses encoding/json.
// Works for any JSON-serializable type.
func JSON[V any]() Codec[V] { return jsonCodec[V]{} }

type jsonCodec[V any] struct{}

func (jsonCodec[V]) Marshal(v V) ([]byte, error) { return json.Marshal(v) }

func (jsonCodec[V]) Unmarshal(data []byte) (V, error) {
	var v V
	return v, json.Unmarshal(data, &v)
}

// Proto returns a codec that uses protobuf encoding.
// factory must return a new zero-value message of type V.
//
// Example:
//
//	cache.Proto(func() *pb.Contact { return &pb.Contact{} })
func Proto[V proto.Message](factory func() V) Codec[V] {
	return protoCodec[V]{newFn: factory}
}

type protoCodec[V proto.Message] struct {
	newFn func() V
}

func (c protoCodec[V]) Marshal(v V) ([]byte, error) { return proto.Marshal(v) }

func (c protoCodec[V]) Unmarshal(data []byte) (V, error) {
	v := c.newFn()
	return v, proto.Unmarshal(data, v)
}

// RawString returns a Codec[string] that stores strings as raw bytes in Redis
// without any encoding wrapper. Preserves backward compatibility with keys
// previously written as plain strings (e.g. "1", locale codes).
func RawString() Codec[string] { return rawStringCodec{} }

type rawStringCodec struct{}

func (rawStringCodec) Marshal(v string) ([]byte, error)   { return []byte(v), nil }
func (rawStringCodec) Unmarshal(data []byte) (string, error) { return string(data), nil }
