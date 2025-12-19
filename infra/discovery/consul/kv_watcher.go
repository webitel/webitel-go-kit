package consul

import (
	"context"
	"time"

	"github.com/hashicorp/consul/api"
)

type consulKVWatcher struct {
	ctx       context.Context
	cancel    context.CancelFunc
	key       string
	client    *api.Client
	lastIndex uint64
}

// Next blocks until the context is canceled or a signal.
// If the context is canceled, it will return an error.
// If a signal is sent, it will return the current value of the key-value pair.
func (w *consulKVWatcher) Next() ([]byte, error) {
	for {
		select {
		case <-w.ctx.Done():
			return nil, w.ctx.Err()
		default:
			pair, meta, err := w.client.KV().Get(w.key,
				(&api.QueryOptions{WaitIndex: w.lastIndex, WaitTime: 30 * time.Second}).WithContext(w.ctx),
			)

			if err != nil {
				return nil, err
			}

			if meta.LastIndex > w.lastIndex {
				w.lastIndex = meta.LastIndex
				if pair == nil {
					return nil, nil
				}
				return pair.Value, nil
			}
		}
	}
}

// Stop cancels the context of the watcher and stops watching the key.
// After calling Stop, the Next method will not block anymore and will return an error.
// It does not return any error.
func (w *consulKVWatcher) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}

	return nil
}
