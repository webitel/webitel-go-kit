package cache

import (
	"bytes"
	"compress/gzip"
	"io"
)

// Compressed wraps a Codec with gzip compression (BestSpeed level).
// Useful for large proto/JSON payloads stored in Redis to reduce memory
// and network I/O. Adds CPU overhead — benchmark before applying to
// values under ~1 KB where compression gains are negligible.
//
// Example:
//
//	cache.RedisConfig[*pb.Contact]{
//	    Codec: cache.Compressed(cache.Proto(func() *pb.Contact { return &pb.Contact{} })),
//	}
func Compressed[V any](inner Codec[V]) Codec[V] {
	return compressedCodec[V]{inner: inner}
}

type compressedCodec[V any] struct {
	inner Codec[V]
}

func (c compressedCodec[V]) Marshal(v V) ([]byte, error) {
	raw, err := c.inner.Marshal(v)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return nil, err
	}
	if _, err = w.Write(raw); err != nil {
		_ = w.Close()
		return nil, err
	}
	if err = w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c compressedCodec[V]) Unmarshal(data []byte) (V, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		var zero V
		return zero, err
	}
	defer r.Close()

	raw, err := io.ReadAll(r)
	if err != nil {
		var zero V
		return zero, err
	}
	return c.inner.Unmarshal(raw)
}
