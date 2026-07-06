package cache

import (
	"context"
	"fmt"
	"time"
)

// Cache is a generic key-value cache with optional L1 (Ristretto) and L2 (Redis) layers.
type Cache[K comparable, V any] interface {
	// Get returns the cached value. The second return is false on cache miss.
	Get(ctx context.Context, key K) (V, bool, error)
	// Set stores the value using the layer's configured TTL.
	Set(ctx context.Context, key K, value V) error
	// SetTTL stores the value with an explicit TTL, overriding the layer default.
	SetTTL(ctx context.Context, key K, value V, ttl time.Duration) error
	// Delete removes the entry from all layers.
	Delete(ctx context.Context, key K) error
	// Close releases resources held by the cache.
	Close() error
}

// KeyFunc serializes a cache key to string for internal storage.
// Used by both L1 (Ristretto string key) and L2 (Redis key prefix).
type KeyFunc[K comparable] func(K) string

// DefaultKeyFunc returns a KeyFunc that uses fmt.Sprint.
// Works correctly for string, int, int64, and other scalar types.
// For struct keys, prefer an explicit KeyFunc for deterministic output.
func DefaultKeyFunc[K comparable]() KeyFunc[K] {
	return func(k K) string { return fmt.Sprint(k) }
}
