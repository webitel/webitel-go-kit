package consul

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/consul/api"
)

// SessionConfig describes a session for [Cluster.CreateSession].
type SessionConfig struct {
	Name string
	// TTL is how long the session lives without a renewal; Consul invalidates it at
	// up to twice that. It must be positive, since a session without checks and
	// without TTL never expires; Consul accepts 10s to 24h.
	TTL time.Duration
	// LockDelay is how long a key stays locked after its session is invalidated.
	// Zero leaves Consul's default of 15s. Consul stores a value above 60s but
	// caps it at 60s when it applies it.
	LockDelay time.Duration
	// Behavior is what happens to the keys of an invalidated session; defaults
	// to [api.SessionBehaviorRelease].
	Behavior string
}

// CreateSession creates a session that no health check is attached to, not even the
// node's serfHealth, so that only its TTL or [Cluster.DestroySession] ends it.
func (c *Cluster) CreateSession(ctx context.Context, cfg SessionConfig) (string, error) {
	if cfg.TTL <= 0 {
		return "", ErrInvalidSessionTTL
	}

	entry := &api.SessionEntry{
		Name:      cfg.Name,
		LockDelay: cfg.LockDelay,
		Behavior:  cfg.Behavior,
	}

	if entry.Behavior == "" {
		entry.Behavior = api.SessionBehaviorRelease
	}

	entry.TTL = cfg.TTL.String()

	var id string

	err := c.do(ctx, true, func(ctx context.Context, client *api.Client) error {
		sid, _, err := client.Session().CreateNoChecks(entry, new(api.WriteOptions).WithContext(ctx))
		if err != nil {
			return err
		}

		id = sid

		return nil
	})

	return id, err
}

// ErrInvalidSessionTTL is returned by [Cluster.CreateSession] for a TTL that is not positive.
var ErrInvalidSessionTTL = errors.New("consul: session ttl must be positive")

// RenewSession resets the TTL of the session, or returns [ErrSessionNotFound]
// when it is gone.
func (c *Cluster) RenewSession(ctx context.Context, id string) error {
	var found bool

	err := c.do(ctx, true, func(ctx context.Context, client *api.Client) error {
		entry, _, err := client.Session().Renew(id, new(api.WriteOptions).WithContext(ctx))
		if err != nil {
			return err
		}

		found = entry != nil

		return nil
	})
	if err != nil {
		return err
	}

	if !found {
		return ErrSessionNotFound
	}

	return nil
}

// DestroySession invalidates the session, which applies its behavior to its keys.
func (c *Cluster) DestroySession(ctx context.Context, id string) error {
	return c.do(ctx, true, func(ctx context.Context, client *api.Client) error {
		_, err := client.Session().Destroy(id, new(api.WriteOptions).WithContext(ctx))

		return err
	})
}

// Acquire locks key for session, writing value, and reports whether it got the lock;
// a session that is gone gives [ErrSessionNotFound].
func (c *Cluster) Acquire(ctx context.Context, key string, value []byte, session string) (bool, error) {
	return c.lock(ctx, true, key, value, session, (*api.KV).Acquire)
}

// Release unlocks key held by session, writing value, and reports whether it did.
func (c *Cluster) Release(ctx context.Context, key string, value []byte, session string) (bool, error) {
	return c.lock(ctx, false, key, value, session, (*api.KV).Release)
}

func (c *Cluster) lock(
	ctx context.Context,
	idempotent bool,
	key string,
	value []byte,
	session string,
	op func(*api.KV, *api.KVPair, *api.WriteOptions) (bool, *api.WriteMeta, error),
) (bool, error) {
	var ok bool

	err := c.do(ctx, idempotent, func(ctx context.Context, client *api.Client) error {
		done, _, err := op(
			client.KV(),
			&api.KVPair{Key: key, Value: value, Session: session},
			new(api.WriteOptions).WithContext(ctx),
		)
		if err != nil {
			return invalidSession(err)
		}

		ok = done

		return nil
	})

	return ok, err
}
