package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// ErrNotFound is returned by LoadFunc to signal a known-absent key.
// The miss is cached for LoadingConfig.NegativeTTL to prevent repeated loads.
var ErrNotFound = errors.New("cache: not found")

// LoadFunc loads a value from the authoritative source on cache miss.
// Return (zero, ErrNotFound) to enable negative caching for this key.
type LoadFunc[K comparable, V any] func(ctx context.Context, key K) (V, error)

// LoadingConfig configures the loading and refresh behaviour.
type LoadingConfig struct {
	// SoftTTL is the age after which a cached entry is considered stale.
	// On Get, stale entries are returned immediately while a background
	// refresh is triggered (stale-while-revalidate). Must be shorter than
	// the L1/L2 TTL so the hard expiry evicts entries that were never refreshed.
	// Zero disables stale-while-revalidate.
	SoftTTL time.Duration

	// NegativeTTL is how long to cache a LoadFunc ErrNotFound result.
	// Prevents thundering-herd on non-existent keys.
	// Zero disables negative caching.
	NegativeTTL time.Duration
}

// sfValue wraps the generic result for singleflight so nil-interface values
// round-trip correctly through the any-typed singleflight.Result.Val.
type sfValue[V any] struct{ v V }

// loadingCache wraps a Cache with automatic loading on miss.
// Concurrent loads for the same key are deduplicated via singleflight —
// one goroutine calls LoadFunc; all others wait and share the result.
// Individual callers can cancel their context without aborting the shared load.
type loadingCache[K comparable, V any] struct {
	cache   Cache[K, V]
	loader  LoadFunc[K, V]
	group   singleflight.Group
	keyFn   KeyFunc[K]
	softTTL time.Duration

	// softExpiry tracks per-key stale-while-revalidate deadlines.
	// Process-local; reset on restart is acceptable — one extra loader call.
	softExpiry sync.Map // string → time.Time

	// refreshing guards against duplicate background refreshes per key.
	refreshing sync.Map // string → struct{}

	// negative caches known-absent keys (ErrNotFound from LoadFunc).
	// nil when NegativeTTL == 0.
	negative Cache[K, struct{}]
}

func newLoadingCache[K comparable, V any](
	c Cache[K, V],
	loader LoadFunc[K, V],
	keyFn KeyFunc[K],
	cfg LoadingConfig,
) Cache[K, V] {
	lc := &loadingCache[K, V]{
		cache:   c,
		loader:  loader,
		keyFn:   keyFn,
		softTTL: cfg.SoftTTL,
	}
	if cfg.NegativeTTL > 0 {
		// Negative cache is always local — no need to replicate to Redis.
		neg, err := newRistretto[K, struct{}](RistrettoConfig{
			MaxCost:     10_000,
			NumCounters: 100_000,
			TTL:         cfg.NegativeTTL,
		}, keyFn)
		if err == nil {
			lc.negative = neg
		}
	}
	return lc
}

func (c *loadingCache[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	// Check negative cache (known-absent keys) before touching main cache.
	if c.negative != nil {
		if _, ok, _ := c.negative.Get(ctx, key); ok {
			var zero V
			return zero, false, ErrNotFound
		}
	}

	v, ok, err := c.cache.Get(ctx, key)
	if err == nil && ok {
		if c.softTTL > 0 {
			keyStr := c.keyFn(key)
			if exp, loaded := c.softExpiry.Load(keyStr); loaded {
				if time.Now().After(exp.(time.Time)) {
					// Stale — return immediately, refresh in background.
					c.triggerBackgroundRefresh(key, keyStr)
				}
			}
		}
		return v, true, nil
	}

	// Cache miss (or read error) — load from source, deduplicated.
	loaded, err := c.doLoad(ctx, key)
	if err != nil {
		var zero V
		return zero, false, err
	}
	return loaded, true, nil
}

func (c *loadingCache[K, V]) doLoad(ctx context.Context, key K) (V, error) {
	keyStr := c.keyFn(key)

	// DoChan lets individual callers cancel without aborting the shared load.
	ch := c.group.DoChan(keyStr, func() (any, error) {
		// Detach from caller context: one cancellation must not abort the
		// load that other waiting goroutines are relying on.
		v, err := c.loader(context.WithoutCancel(ctx), key)

		if errors.Is(err, ErrNotFound) {
			if c.negative != nil {
				_ = c.negative.Set(context.Background(), key, struct{}{})
			}
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}

		_ = c.cache.Set(context.Background(), key, v)
		if c.softTTL > 0 {
			c.softExpiry.Store(keyStr, time.Now().Add(c.softTTL))
		}
		return sfValue[V]{v: v}, nil
	})

	select {
	case res := <-ch:
		if res.Err != nil {
			var zero V
			return zero, res.Err
		}
		return res.Val.(sfValue[V]).v, nil
	case <-ctx.Done():
		var zero V
		return zero, fmt.Errorf("cache: load cancelled: %w", ctx.Err())
	}
}

func (c *loadingCache[K, V]) triggerBackgroundRefresh(key K, keyStr string) {
	if _, alreadyRunning := c.refreshing.LoadOrStore(keyStr, struct{}{}); alreadyRunning {
		return
	}
	go func() {
		defer c.refreshing.Delete(keyStr)

		v, err := c.loader(context.Background(), key)
		if err != nil {
			return
		}
		_ = c.cache.Set(context.Background(), key, v)
		c.softExpiry.Store(keyStr, time.Now().Add(c.softTTL))
	}()
}

func (c *loadingCache[K, V]) Set(ctx context.Context, key K, value V) error {
	if c.softTTL > 0 {
		c.softExpiry.Store(c.keyFn(key), time.Now().Add(c.softTTL))
	}
	if c.negative != nil {
		_ = c.negative.Delete(ctx, key)
	}
	return c.cache.Set(ctx, key, value)
}

func (c *loadingCache[K, V]) SetTTL(ctx context.Context, key K, value V, ttl time.Duration) error {
	return c.cache.SetTTL(ctx, key, value, ttl)
}

func (c *loadingCache[K, V]) Delete(ctx context.Context, key K) error {
	keyStr := c.keyFn(key)
	c.softExpiry.Delete(keyStr)
	c.refreshing.Delete(keyStr)
	if c.negative != nil {
		_ = c.negative.Delete(ctx, key)
	}
	return c.cache.Delete(ctx, key)
}

func (c *loadingCache[K, V]) Close() error {
	if c.negative != nil {
		_ = c.negative.Close()
	}
	return c.cache.Close()
}
