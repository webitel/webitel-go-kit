package cache

import (
	"context"
	"sync/atomic"
	"time"
)

// CircuitBreakerConfig configures the circuit breaker applied to the L2 layer.
// When L2 errors exceed Threshold, the circuit opens and L2 operations are
// skipped silently — the cache degrades to L1-only until Redis recovers.
type CircuitBreakerConfig struct {
	// Threshold is the number of consecutive errors that open the circuit.
	Threshold int64
	// Timeout is how long the circuit stays open before allowing a probe.
	Timeout time.Duration
}

type cbState int32

const (
	cbClosed   cbState = 0
	cbOpen     cbState = 1
	cbHalfOpen cbState = 2
)

// circuitBreaker wraps a Cache (typically L2) with a state machine:
//
//	Closed  → errors ≥ Threshold  → Open
//	Open    → Timeout elapsed     → HalfOpen (probe)
//	HalfOpen → success            → Closed
//	HalfOpen → failure            → Open
//
// In Open state, Get returns (zero, false, nil) and Set/Delete are no-ops.
// This lets the multilevel cache degrade gracefully to L1 when Redis is down.
type circuitBreaker[K comparable, V any] struct {
	inner     Cache[K, V]
	threshold int64
	timeout   time.Duration

	state    atomic.Int32
	failures atomic.Int64
	openedAt atomic.Int64 // unix nanoseconds
}

func newCircuitBreaker[K comparable, V any](inner Cache[K, V], cfg CircuitBreakerConfig) Cache[K, V] {
	return &circuitBreaker[K, V]{
		inner:     inner,
		threshold: cfg.Threshold,
		timeout:   cfg.Timeout,
	}
}

func (c *circuitBreaker[K, V]) allow() bool {
	switch cbState(c.state.Load()) {
	case cbClosed:
		return true
	case cbOpen:
		elapsed := time.Since(time.Unix(0, c.openedAt.Load()))
		if elapsed >= c.timeout {
			// Transition to half-open for a single probe.
			c.state.CompareAndSwap(int32(cbOpen), int32(cbHalfOpen))
			return true
		}
		return false
	default: // cbHalfOpen
		return true
	}
}

func (c *circuitBreaker[K, V]) recordSuccess() {
	c.failures.Store(0)
	c.state.Store(int32(cbClosed))
}

func (c *circuitBreaker[K, V]) recordFailure() {
	if c.failures.Add(1) >= c.threshold {
		if c.state.CompareAndSwap(int32(cbClosed), int32(cbOpen)) ||
			c.state.CompareAndSwap(int32(cbHalfOpen), int32(cbOpen)) {
			c.openedAt.Store(time.Now().UnixNano())
		}
	}
}

func (c *circuitBreaker[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	if !c.allow() {
		var zero V
		return zero, false, nil // degrade gracefully — appears as cache miss
	}
	v, ok, err := c.inner.Get(ctx, key)
	if err != nil {
		c.recordFailure()
		var zero V
		return zero, false, err
	}
	c.recordSuccess()
	return v, ok, nil
}

func (c *circuitBreaker[K, V]) Set(ctx context.Context, key K, value V) error {
	if !c.allow() {
		return nil // silent drop — L1 was already written by multilevel
	}
	err := c.inner.Set(ctx, key, value)
	if err != nil {
		c.recordFailure()
		return err
	}
	c.recordSuccess()
	return nil
}

func (c *circuitBreaker[K, V]) SetTTL(ctx context.Context, key K, value V, ttl time.Duration) error {
	if !c.allow() {
		return nil
	}
	err := c.inner.SetTTL(ctx, key, value, ttl)
	if err != nil {
		c.recordFailure()
		return err
	}
	c.recordSuccess()
	return nil
}

func (c *circuitBreaker[K, V]) Delete(ctx context.Context, key K) error {
	if !c.allow() {
		return nil
	}
	err := c.inner.Delete(ctx, key)
	if err != nil {
		c.recordFailure()
		return err
	}
	c.recordSuccess()
	return nil
}

func (c *circuitBreaker[K, V]) Close() error { return c.inner.Close() }
