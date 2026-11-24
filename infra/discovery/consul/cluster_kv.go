package consul

import (
	"context"

	"github.com/hashicorp/consul/api"
)

// KVEntry is a key of the Consul KV with the index it was last modified at.
type KVEntry struct {
	Key         string
	Value       []byte
	ModifyIndex uint64
	// Session is the session holding the key as a lock; empty when none does.
	Session string
}

// KVGet returns the key with its ModifyIndex, or [ErrKeyNotFound].
func (c *Cluster) KVGet(ctx context.Context, key string) (*KVEntry, error) {
	var pair *api.KVPair

	err := c.do(ctx, true, func(ctx context.Context, client *api.Client) error {
		p, _, err := client.KV().Get(key, new(api.QueryOptions).WithContext(ctx))
		if err != nil {
			return err
		}

		pair = p

		return nil
	})
	if err != nil {
		return nil, err
	}

	if pair == nil {
		return nil, ErrKeyNotFound
	}

	return &KVEntry{Key: pair.Key, Value: pair.Value, ModifyIndex: pair.ModifyIndex, Session: pair.Session}, nil
}

// KVList returns the keys under prefix with their ModifyIndex; none is not an error.
func (c *Cluster) KVList(ctx context.Context, prefix string) ([]KVEntry, error) {
	var pairs api.KVPairs

	err := c.do(ctx, true, func(ctx context.Context, client *api.Client) error {
		p, _, err := client.KV().List(prefix, new(api.QueryOptions).WithContext(ctx))
		if err != nil {
			return err
		}

		pairs = p

		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]KVEntry, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, KVEntry{Key: p.Key, Value: p.Value, ModifyIndex: p.ModifyIndex, Session: p.Session})
	}

	return out, nil
}

// KVCAS writes value only if the key's ModifyIndex is still modifyIndex and reports
// whether it did. Index 0 writes only when the key does not exist.
func (c *Cluster) KVCAS(ctx context.Context, key string, value []byte, modifyIndex uint64) (bool, error) {
	var ok bool

	err := c.do(ctx, false, func(ctx context.Context, client *api.Client) error {
		done, _, err := client.KV().CAS(
			&api.KVPair{Key: key, Value: value, ModifyIndex: modifyIndex},
			new(api.WriteOptions).WithContext(ctx),
		)
		if err != nil {
			return err
		}

		ok = done

		return nil
	})

	return ok, err
}

// KVDeleteCAS deletes the key only if its ModifyIndex is still modifyIndex and reports
// whether it did.
func (c *Cluster) KVDeleteCAS(ctx context.Context, key string, modifyIndex uint64) (bool, error) {
	var ok bool

	err := c.do(ctx, false, func(ctx context.Context, client *api.Client) error {
		done, _, err := client.KV().DeleteCAS(
			&api.KVPair{Key: key, ModifyIndex: modifyIndex},
			new(api.WriteOptions).WithContext(ctx),
		)
		if err != nil {
			return err
		}

		ok = done

		return nil
	})

	return ok, err
}
