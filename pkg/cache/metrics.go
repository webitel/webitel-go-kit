package cache

import (
	"context"
	"time"
)

// Stats is the observability hook for cache operations.
// Wire an implementation via Builder.WithStats().
// Use NoopStats() when metrics are not needed.
//
// To integrate with OpenTelemetry:
//
//	type otelStats struct {
//	    hits, misses, errors metric.Int64Counter
//	    loadDur              metric.Float64Histogram
//	}
//
//	func (s *otelStats) Hit(ctx context.Context, name, layer string) {
//	    s.hits.Add(ctx, 1, metric.WithAttributes(
//	        attribute.String("cache", name),
//	        attribute.String("layer", layer),
//	    ))
//	}
//	// ... implement Miss, Error, LoadDuration similarly
type Stats interface {
	// Hit is called when a key is found in the given layer ("l1" or "l2").
	Hit(ctx context.Context, name, layer string)
	// Miss is called when a key is not found in the given layer.
	// For a multilevel cache: Miss("l1") means L2 may still have the value;
	// Miss("l2") is a true cache miss across all layers.
	Miss(ctx context.Context, name, layer string)
	// Error is called when an operation fails. op is "get", "set", or "delete".
	Error(ctx context.Context, name, layer, op string)
	// LoadDuration is called after LoadFunc returns (LoadingCache only).
	// err is nil on success.
	LoadDuration(ctx context.Context, name string, d time.Duration, err error)
}

// NoopStats returns a Stats implementation that discards all events.
func NoopStats() Stats { return noopStats{} }

type noopStats struct{}

func (noopStats) Hit(context.Context, string, string)                    {}
func (noopStats) Miss(context.Context, string, string)                   {}
func (noopStats) Error(context.Context, string, string, string)          {}
func (noopStats) LoadDuration(context.Context, string, time.Duration, error) {}

// instrumentedCache wraps a Cache and records stats on every operation.
// Created internally by Builder when WithStats is set.
type instrumentedCache[K comparable, V any] struct {
	inner Cache[K, V]
	stats Stats
	name  string
	layer string // "l1" or "l2"
}

func newInstrumentedCache[K comparable, V any](
	inner Cache[K, V],
	stats Stats,
	name, layer string,
) Cache[K, V] {
	return &instrumentedCache[K, V]{
		inner: inner,
		stats: stats,
		name:  name,
		layer: layer,
	}
}

func (c *instrumentedCache[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	v, ok, err := c.inner.Get(ctx, key)
	switch {
	case err != nil:
		c.stats.Error(ctx, c.name, c.layer, "get")
	case ok:
		c.stats.Hit(ctx, c.name, c.layer)
	default:
		c.stats.Miss(ctx, c.name, c.layer)
	}
	return v, ok, err
}

func (c *instrumentedCache[K, V]) Set(ctx context.Context, key K, value V) error {
	if err := c.inner.Set(ctx, key, value); err != nil {
		c.stats.Error(ctx, c.name, c.layer, "set")
		return err
	}
	return nil
}

func (c *instrumentedCache[K, V]) SetTTL(ctx context.Context, key K, value V, ttl time.Duration) error {
	if err := c.inner.SetTTL(ctx, key, value, ttl); err != nil {
		c.stats.Error(ctx, c.name, c.layer, "set")
		return err
	}
	return nil
}

func (c *instrumentedCache[K, V]) Delete(ctx context.Context, key K) error {
	if err := c.inner.Delete(ctx, key); err != nil {
		c.stats.Error(ctx, c.name, c.layer, "delete")
		return err
	}
	return nil
}

func (c *instrumentedCache[K, V]) Close() error { return c.inner.Close() }
