package cache

import (
	"context"
	"time"
)

// multiLevel composes L1 and L2 into a single Cache.
//
// Read strategy: L1 hit → return immediately.
// L1 miss → try L2 → on hit, populate L1 (best-effort) and return.
//
// Write strategy: write-through — Set/Delete propagates to both layers.
// L1 write failure is returned immediately without touching L2.
type multiLevel[K comparable, V any] struct {
	l1 Cache[K, V]
	l2 Cache[K, V]
}

func (c *multiLevel[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	v, ok, err := c.l1.Get(ctx, key)
	if err == nil && ok {
		return v, true, nil
	}

	v, ok, err = c.l2.Get(ctx, key)
	if err != nil || !ok {
		return v, ok, err
	}

	// Populate L1 on L2 hit; ignore error — L1 is best-effort.
	_ = c.l1.Set(ctx, key, v)
	return v, true, nil
}

func (c *multiLevel[K, V]) Set(ctx context.Context, key K, value V) error {
	if err := c.l1.Set(ctx, key, value); err != nil {
		return err
	}
	return c.l2.Set(ctx, key, value)
}

func (c *multiLevel[K, V]) SetTTL(ctx context.Context, key K, value V, ttl time.Duration) error {
	if err := c.l1.SetTTL(ctx, key, value, ttl); err != nil {
		return err
	}
	return c.l2.SetTTL(ctx, key, value, ttl)
}

func (c *multiLevel[K, V]) Delete(ctx context.Context, key K) error {
	if err := c.l1.Delete(ctx, key); err != nil {
		return err
	}
	return c.l2.Delete(ctx, key)
}

func (c *multiLevel[K, V]) Close() error {
	_ = c.l1.Close()
	return c.l2.Close()
}
