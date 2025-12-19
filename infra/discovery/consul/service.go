package consul

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/webitel/webitel-go-kit/infra/discovery"
)

type serviceSet struct {
	registry    *Registry
	serviceName string
	watcher     map[*watcher]struct{}
	ref         atomic.Int32
	services    *atomic.Value
	lock        sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

// broadcast sends a signal to all watchers that are registered with the service set.
// The signal is sent by sending an empty struct{}{} on the watcher's event channel.
// If the event channel is already closed, the broadcast call does not block.
// The broadcast call is thread-safe and can be called concurrently from multiple goroutines.
// The broadcast call is protected by a read-write lock, which ensures that the call is atomic.
// The broadcast call also stores the given service instances in the service set's cache.
// The cache is updated atomically, which ensures that the call is thread-safe.
func (ss *serviceSet) broadcast(si []*discovery.ServiceInstance) {
	ss.services.Store(si)
	ss.lock.RLock()         //+[R] lock
	defer ss.lock.RUnlock() //-[R] lock

	for k := range ss.watcher {
		select {
		case k.event <- struct{}{}:
		default:
		}
	}
}

// delete removes the given watcher from the service set and cancels the context of the service set.
// If the reference count of the service set is zero after removing the watcher, it will try to delete the service set from the registry.
// The delete call is thread-safe and protected by a read-write lock, which ensures that the call is atomic.
// It does not return any error.
func (ss *serviceSet) delete(w *watcher) {
	ss.lock.Lock() //+[RW] lock
	delete(ss.watcher, w)
	ss.lock.Unlock() //-[RW] lock

	ss.registry.tryDelete(ss)
}
