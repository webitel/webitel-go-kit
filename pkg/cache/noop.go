package cache

import (
	"context"
	"sync"
	"time"
)

// Noop returns a Cache backed by a plain in-memory map with no TTL and no eviction.
// Intended for unit tests — eliminates Ristretto and Redis dependencies.
func Noop[K comparable, V any]() Cache[K, V] {
	return &noopCache[K, V]{m: make(map[K]V)}
}

type noopCache[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

func (c *noopCache[K, V]) Get(_ context.Context, key K) (V, bool, error) {
	c.mu.RLock()
	v, ok := c.m[key]
	c.mu.RUnlock()
	return v, ok, nil
}

func (c *noopCache[K, V]) Set(_ context.Context, key K, value V) error {
	c.mu.Lock()
	c.m[key] = value
	c.mu.Unlock()
	return nil
}

func (c *noopCache[K, V]) SetTTL(ctx context.Context, key K, value V, _ time.Duration) error {
	return c.Set(ctx, key, value)
}

func (c *noopCache[K, V]) Delete(_ context.Context, key K) error {
	c.mu.Lock()
	delete(c.m, key)
	c.mu.Unlock()
	return nil
}

func (c *noopCache[K, V]) Close() error { return nil }
