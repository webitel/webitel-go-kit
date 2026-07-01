package cache

import (
	"context"
	"math/rand/v2"
	"sync"
	"time"
)

// GetMany fetches multiple keys concurrently.
// Returns a map of found entries and a slice of keys that were not in cache.
// All gets run in parallel; the first error aborts remaining work and is returned.
func GetMany[K comparable, V any](ctx context.Context, c Cache[K, V], keys []K) (found map[K]V, missed []K, err error) {
	found = make(map[K]V, len(keys))
	missed = make([]K, 0)

	type result struct {
		key K
		val V
		ok  bool
		err error
	}

	results := make(chan result, len(keys))
	for _, key := range keys {
		go func() {
			v, ok, e := c.Get(ctx, key)
			results <- result{key: key, val: v, ok: ok, err: e}
		}()
	}

	var mu sync.Mutex
	for range keys {
		r := <-results
		if r.err != nil {
			err = r.err
			continue
		}
		mu.Lock()
		if r.ok {
			found[r.key] = r.val
		} else {
			missed = append(missed, r.key)
		}
		mu.Unlock()
	}
	return found, missed, err
}

// SetMany stores multiple key-value pairs concurrently.
// Returns the first error encountered; other writes are not aborted.
func SetMany[K comparable, V any](ctx context.Context, c Cache[K, V], entries map[K]V) error {
	errc := make(chan error, len(entries))
	for k, v := range entries {
		go func() { errc <- c.Set(ctx, k, v) }()
	}
	var first error
	for range entries {
		if e := <-errc; e != nil && first == nil {
			first = e
		}
	}
	return first
}

// DeleteMany removes multiple keys concurrently.
// Returns the first error encountered; other deletes are not aborted.
func DeleteMany[K comparable, V any](ctx context.Context, c Cache[K, V], keys []K) error {
	errc := make(chan error, len(keys))
	for _, k := range keys {
		go func() { errc <- c.Delete(ctx, k) }()
	}
	var first error
	for range keys {
		if e := <-errc; e != nil && first == nil {
			first = e
		}
	}
	return first
}

// Jitter randomizes a TTL duration by ±fraction to prevent cache avalanche.
// When many keys are set with the same TTL they would all expire simultaneously,
// causing a thundering herd to the backing store. Jitter spreads expirations.
//
// Example: Jitter(24*time.Hour, 0.1) returns a duration between 21.6h and 26.4h.
func Jitter(ttl time.Duration, fraction float64) time.Duration {
	if fraction <= 0 {
		return ttl
	}
	// Random offset in [-fraction, +fraction] of ttl.
	delta := float64(ttl) * fraction * (2*rand.Float64() - 1)
	return ttl + time.Duration(delta)
}
