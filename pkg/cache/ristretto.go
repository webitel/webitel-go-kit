package cache

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

// RistrettoConfig configures the L1 in-memory Ristretto cache.
type RistrettoConfig struct {
	// MaxCost is the maximum total cost the cache may hold.
	// With DefaultCost=1 this equals the maximum number of entries.
	MaxCost int64
	// NumCounters is the number of keys tracked for admission frequency.
	// Recommended: 10× the expected number of unique keys.
	// If zero, defaults to MaxCost*10.
	NumCounters int64
	// BufferItems is the size of the Get result buffer per shard. Default: 64.
	BufferItems int64
	// TTL is the per-entry expiration duration.
	// When used with L2, keep L1 TTL shorter than L2 TTL to bound staleness
	// without requiring pub/sub invalidation across nodes.
	TTL time.Duration
	// DefaultCost is the cost charged per entry when SetTTL is called.
	// Defaults to 1 if zero.
	DefaultCost int64
}

// ristrettoCache is the L1 implementation backed by Ristretto.
// Keys are converted to string via keyFn so the inner cache always uses
// ristretto.Cache[string, V], sidestepping ristretto's Key type constraints.
type ristrettoCache[K comparable, V any] struct {
	inner       *ristretto.Cache[string, V]
	keyFn       KeyFunc[K]
	defaultCost int64
	ttl         time.Duration
}

func newRistretto[K comparable, V any](cfg RistrettoConfig, keyFn KeyFunc[K]) (*ristrettoCache[K, V], error) {
	if cfg.NumCounters == 0 {
		cfg.NumCounters = cfg.MaxCost * 10
	}
	if cfg.BufferItems == 0 {
		cfg.BufferItems = 64
	}
	if cfg.DefaultCost == 0 {
		cfg.DefaultCost = 1
	}

	inner, err := ristretto.NewCache(&ristretto.Config[string, V]{
		NumCounters: cfg.NumCounters,
		MaxCost:     cfg.MaxCost,
		BufferItems: cfg.BufferItems,
	})
	if err != nil {
		return nil, err
	}

	return &ristrettoCache[K, V]{
		inner:       inner,
		keyFn:       keyFn,
		defaultCost: cfg.DefaultCost,
		ttl:         cfg.TTL,
	}, nil
}

func (c *ristrettoCache[K, V]) Get(_ context.Context, key K) (V, bool, error) {
	v, ok := c.inner.Get(c.keyFn(key))
	return v, ok, nil
}

func (c *ristrettoCache[K, V]) Set(_ context.Context, key K, value V) error {
	if c.ttl > 0 {
		c.inner.SetWithTTL(c.keyFn(key), value, c.defaultCost, c.ttl)
	} else {
		c.inner.Set(c.keyFn(key), value, c.defaultCost)
	}
	return nil
}

func (c *ristrettoCache[K, V]) SetTTL(_ context.Context, key K, value V, ttl time.Duration) error {
	c.inner.SetWithTTL(c.keyFn(key), value, c.defaultCost, ttl)
	return nil
}

func (c *ristrettoCache[K, V]) Delete(_ context.Context, key K) error {
	c.inner.Del(c.keyFn(key))
	return nil
}

func (c *ristrettoCache[K, V]) Close() error {
	c.inner.Close()
	return nil
}
