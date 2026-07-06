package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig configures the L2 distributed Redis cache.
type RedisConfig[V any] struct {
	// Client is the go-redis client. Required.
	Client *redis.Client
	// Prefix is prepended to all keys: "{prefix}:{key}".
	// Use a unique prefix per cache to avoid key collisions.
	Prefix string
	// TTL is the default time-to-live for stored entries. Required.
	TTL time.Duration
	// Codec handles serialization to/from []byte. Required.
	Codec Codec[V]
}

type redisCache[K comparable, V any] struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
	codec  Codec[V]
	keyFn  KeyFunc[K]
}

func newRedisCache[K comparable, V any](cfg RedisConfig[V], keyFn KeyFunc[K]) *redisCache[K, V] {
	return &redisCache[K, V]{
		client: cfg.Client,
		prefix: cfg.Prefix,
		ttl:    cfg.TTL,
		codec:  cfg.Codec,
		keyFn:  keyFn,
	}
}

func (c *redisCache[K, V]) key(k K) string {
	return fmt.Sprintf("%s:%s", c.prefix, c.keyFn(k))
}

func (c *redisCache[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	data, err := c.client.Get(ctx, c.key(key)).Bytes()
	if err == redis.Nil {
		var zero V
		return zero, false, nil
	}
	if err != nil {
		var zero V
		return zero, false, err
	}
	v, err := c.codec.Unmarshal(data)
	if err != nil {
		var zero V
		return zero, false, err
	}
	return v, true, nil
}

func (c *redisCache[K, V]) Set(ctx context.Context, key K, value V) error {
	return c.SetTTL(ctx, key, value, c.ttl)
}

func (c *redisCache[K, V]) SetTTL(ctx context.Context, key K, value V, ttl time.Duration) error {
	data, err := c.codec.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(key), data, ttl).Err()
}

func (c *redisCache[K, V]) Delete(ctx context.Context, key K) error {
	return c.client.Del(ctx, c.key(key)).Err()
}

func (c *redisCache[K, V]) Close() error { return nil }
