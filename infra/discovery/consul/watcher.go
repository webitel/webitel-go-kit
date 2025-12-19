package consul

import (
	"context"

	"github.com/webitel/webitel-go-kit/infra/discovery"
)

type watcher struct {
	event chan struct{}
	set   *serviceSet

	ctx    context.Context
	cancel context.CancelFunc
}

// Next blocks until the context is canceled or a signal is sent to the watcher's event channel.
// If the context is canceled, it will return an error.
// If a signal is sent, it will return the current service instances of the service set.
// The function is thread-safe and protected by a read-write lock.
func (w *watcher) Next() ([]*discovery.ServiceInstance, error) {
	select {
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	case <-w.event:
	}

	var services []*discovery.ServiceInstance
	if ss, ok := w.set.services.Load().([]*discovery.ServiceInstance); ok {
		services = append(services, ss...)
		return services, nil
	}

	return services, nil
}

// Stop cancels the watcher and removes it from the service set.
// After calling Stop, the Next method will not block anymore and will return an error.
// The function is thread-safe and protected by a read-write lock.
// It does not return any error.
func (w *watcher) Stop() error {
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil

		w.set.delete(w)
	}
	return nil
}
