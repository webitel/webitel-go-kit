package consul

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/webitel/webitel-go-kit/infra/discovery"
)

func TestServiceSetBroadcastSuccess(t *testing.T) {
	ss := &serviceSet{
		watcher:  make(map[*watcher]struct{}),
		services: &atomic.Value{},
	}

	w1 := &watcher{event: make(chan struct{}, 1)}
	w2 := &watcher{event: make(chan struct{}, 1)}
	ss.watcher[w1] = struct{}{}
	ss.watcher[w2] = struct{}{}

	instances := []*discovery.ServiceInstance{
		{Id: "node-1", Endpoints: []string{"10.0.0.1"}},
		{Id: "node-2", Endpoints: []string{"10.0.0.2"}},
	}

	ss.broadcast(instances)

	stored := ss.services.Load().([]*discovery.ServiceInstance)
	assert.Len(t, stored, 2)
	assert.Equal(t, "node-1", stored[0].Id)

	select {
	case <-w1.event:
	default:
		t.Error("Watcher 1 did not receive the broadcast signal")
	}

	select {
	case <-w2.event:
	default:
		t.Error("Watcher 2 did not receive the broadcast signal")
	}
}

func TestServiceSetBroadcastNonBlocking(t *testing.T) {
	ss := &serviceSet{
		watcher:  make(map[*watcher]struct{}),
		services: &atomic.Value{},
	}

	w := &watcher{event: make(chan struct{})}
	ss.watcher[w] = struct{}{}

	instances := []*discovery.ServiceInstance{{Id: "node-1"}}

	done := make(chan bool)
	go func() {
		ss.broadcast(instances)
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("broadcast blocked on a full channel, but it should be non-blocking")
	}
}

func TestServiceSetBroadcastEmptyWatchers(t *testing.T) {
	ss := &serviceSet{
		watcher:  make(map[*watcher]struct{}),
		services: &atomic.Value{},
	}

	instances := []*discovery.ServiceInstance{{Id: "node-1"}}

	assert.NotPanics(t, func() {
		ss.broadcast(instances)
	})

	stored := ss.services.Load().([]*discovery.ServiceInstance)
	assert.Equal(t, "node-1", stored[0].Id)
}

func TestServiceSetBroadcastConcurrency(t *testing.T) {
	ss := &serviceSet{
		watcher:  make(map[*watcher]struct{}),
		services: &atomic.Value{},
	}

	w := &watcher{event: make(chan struct{}, 10)}
	ss.watcher[w] = struct{}{}
	go func() {
		for range w.event {
		}
	}()

	instances := []*discovery.ServiceInstance{{Id: "race-node"}}

	const iterations = 1000
	for range iterations {
		go ss.broadcast(instances)
	}
}
