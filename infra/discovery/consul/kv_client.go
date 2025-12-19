package consul

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/webitel/webitel-go-kit/infra/discovery"
)

// interface guard to assure that kVClient implements [discovery.KVProvider]
var (
	_ discovery.KVProvider = (*kVClient)(nil)
)

type kVClient struct {
	client *api.Client
}

// DeleteFromKV deletes the given key-value pair from the Consul KV.
// If the context is canceled, it will return an error.
// If the Delete call failed with an error, it will return an error.
// Otherwise, it will return nil.
func (c *kVClient) DeleteFromKV(ctx context.Context, key string) error {
	if _, err := c.client.KV().Delete(key, new(api.WriteOptions).WithContext(ctx)); err != nil {
		return fmt.Errorf("consul kv delete error (key: %s): %w", key, err)
	}

	return nil
}

// GetFromKV returns the value associated with the given key from the Consul KV.
// If the context is canceled, it will return an error.
// If the key is not found in the Consul KV, it will return an error.
// If the Get call failed with an error, it will return an error.
// Otherwise, it will return the value associated with the given key.
func (c *kVClient) GetFromKV(ctx context.Context, key string) ([]byte, error) {
	pair, _, err := c.client.KV().Get(key, new(api.QueryOptions).WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("consul kv get error: %w", err)
	}

	if pair == nil {
		return nil, fmt.Errorf("key '%s' not found in consul kv", key)
	}

	return pair.Value, nil
}

// It returns a KVWatcher for the given key.
// The KVWatcher is a data structure that can be used to watch the value of the given key.
// The KVWatcher is created by calling the GetKV method of the Consul client. If the timeout field of the Registry is set,
// it will wrap the GetKV method with a context.WithTimeout call. If the value of the key is resolved successfully, it will broadcast the value to all
// registered watchers of the KVWatcher. The GetKVWatcher method is run in a separate goroutine and will continuously resolve the value of the key until the context is
// canceled.
func (c *kVClient) GetKVWatcher(ctx context.Context, key string) discovery.KVWatcher {
	var (
		cancelCtx, cancel = context.WithCancel(ctx)
		_, meta, _        = c.client.KV().Get(key, nil)
		lastIdx           = uint64(0)
	)

	if meta != nil {
		lastIdx = meta.LastIndex
	}

	watcher := new(consulKVWatcher)
	{
		watcher.cancel = cancel
		watcher.ctx = cancelCtx
		watcher.client = c.client
		watcher.key = key
		watcher.lastIndex = lastIdx
	}

	return watcher
}

// It writes the given value to the Consul KV with the given key.
// If the context is canceled, it will return an error.
// If the Put call failed with an error, it will return an error.
// Otherwise, it will return nil.
func (c *kVClient) PutToKV(ctx context.Context, key string, value []byte) error {
	p := new(api.KVPair)
	{
		p.Key = key
		p.Value = value
	}

	if _, err := c.client.KV().Put(p, new(api.WriteOptions).WithContext(ctx)); err != nil {
		return fmt.Errorf("consul kv put error: %w", err)
	}

	return nil
}
