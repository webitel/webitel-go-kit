package cache

import (
	"errors"
	"fmt"
)

// Builder constructs a Cache[K, V] with optional L1 (Ristretto) and L2 (Redis)
// layers, plus cross-cutting concerns: circuit breaking, observability, and
// automatic loading with singleflight deduplication.
//
// Layer wiring order (inside → out):
//
//	L1 raw → instrument(l1) → \
//	                            multiLevel → loadingCache
//	L2 raw → circuitBreaker → instrument(l2) → /
//
// Usage — L1 + L2 with singleflight loader and OTel stats:
//
//	c, err := cache.New[string, *pb.Contact]().
//	    Name("contact").
//	    L1(cache.RistrettoConfig{MaxCost: 10_000, TTL: 30 * time.Second}).
//	    L2(cache.RedisConfig[*pb.Contact]{
//	        Client: rdb,
//	        Prefix: "contact",
//	        TTL:    24 * time.Hour,
//	        Codec:  cache.Proto(func() *pb.Contact { return &pb.Contact{} }),
//	    }).
//	    WithCircuitBreaker(cache.CircuitBreakerConfig{Threshold: 5, Timeout: 10 * time.Second}).
//	    WithStats(myOtelStats).
//	    WithLoader(func(ctx context.Context, id string) (*pb.Contact, error) {
//	        return db.GetContact(ctx, id)
//	    }, cache.LoadingConfig{
//	        SoftTTL:     20 * time.Second,
//	        NegativeTTL: 2 * time.Minute,
//	    }).
//	    Build()
type Builder[K comparable, V any] struct {
	name    string
	l1Cfg   *RistrettoConfig
	l2Cfg   *RedisConfig[V]
	keyFn   KeyFunc[K]
	cbCfg   *CircuitBreakerConfig
	stats   Stats
	loader  LoadFunc[K, V]
	loadCfg LoadingConfig
}

// New returns a Builder for Cache[K, V].
// The default KeyFunc uses fmt.Sprint; override with KeyFunc() for struct keys.
func New[K comparable, V any]() *Builder[K, V] {
	return &Builder[K, V]{
		keyFn: DefaultKeyFunc[K](),
	}
}

// Name sets a label used in Stats calls to distinguish this cache from others.
func (b *Builder[K, V]) Name(name string) *Builder[K, V] {
	b.name = name
	return b
}

// KeyFunc overrides the default fmt.Sprint key serializer.
// Provide an explicit func for struct keys or keys with non-obvious string representation.
func (b *Builder[K, V]) KeyFunc(fn KeyFunc[K]) *Builder[K, V] {
	b.keyFn = fn
	return b
}

// L1 configures the Ristretto in-memory layer.
// Keep L1 TTL shorter than L2 TTL to bound staleness without pub/sub invalidation.
func (b *Builder[K, V]) L1(cfg RistrettoConfig) *Builder[K, V] {
	b.l1Cfg = &cfg
	return b
}

// L2 configures the Redis distributed layer.
func (b *Builder[K, V]) L2(cfg RedisConfig[V]) *Builder[K, V] {
	b.l2Cfg = &cfg
	return b
}

// WithCircuitBreaker applies a circuit breaker to the L2 layer.
// When L2 errors exceed Threshold, the circuit opens and L2 is bypassed
// for Timeout duration — the cache degrades to L1-only without crashing.
// No-op when L2 is not configured.
func (b *Builder[K, V]) WithCircuitBreaker(cfg CircuitBreakerConfig) *Builder[K, V] {
	b.cbCfg = &cfg
	return b
}

// WithStats wires an observability hook. Hit/Miss/Error are called per layer.
func (b *Builder[K, V]) WithStats(s Stats) *Builder[K, V] {
	b.stats = s
	return b
}

// WithLoader attaches an automatic loading function invoked on cache miss.
// Concurrent misses for the same key are collapsed via singleflight — only
// one goroutine calls fn; all others wait and receive the same result.
// Individual callers may cancel their context without aborting the shared load.
func (b *Builder[K, V]) WithLoader(fn LoadFunc[K, V], cfg ...LoadingConfig) *Builder[K, V] {
	b.loader = fn
	if len(cfg) > 0 {
		b.loadCfg = cfg[0]
	}
	return b
}

// Build assembles and returns the configured cache.
// Returns an error when neither L1 nor L2 is configured, or L1 init fails.
func (b *Builder[K, V]) Build() (Cache[K, V], error) {
	if b.l1Cfg == nil && b.l2Cfg == nil {
		return nil, errors.New("cache: at least one layer (L1 or L2) must be configured")
	}

	name := b.name
	var (
		l1  Cache[K, V]
		l2  Cache[K, V]
		err error
	)

	if b.l1Cfg != nil {
		l1, err = newRistretto[K, V](*b.l1Cfg, b.keyFn)
		if err != nil {
			return nil, fmt.Errorf("cache: init L1: %w", err)
		}
		if b.stats != nil {
			l1 = newInstrumentedCache(l1, b.stats, name, "l1")
		}
	}

	if b.l2Cfg != nil {
		var l2chain Cache[K, V] = newRedisCache(*b.l2Cfg, b.keyFn)
		if b.cbCfg != nil {
			l2chain = newCircuitBreaker(l2chain, *b.cbCfg)
		}
		if b.stats != nil {
			l2chain = newInstrumentedCache(l2chain, b.stats, name, "l2")
		}
		l2 = l2chain
	}

	var result Cache[K, V]
	switch {
	case l1 != nil && l2 != nil:
		result = &multiLevel[K, V]{l1: l1, l2: l2}
	case l1 != nil:
		result = l1
	default:
		result = l2
	}

	if b.loader != nil {
		result = newLoadingCache(result, b.loader, b.keyFn, b.loadCfg)
	}

	return result, nil
}

// MustBuild is like Build but panics on error. Suitable for package-level init.
func (b *Builder[K, V]) MustBuild() Cache[K, V] {
	c, err := b.Build()
	if err != nil {
		panic(err)
	}
	return c
}
