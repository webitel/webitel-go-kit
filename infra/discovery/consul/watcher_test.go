package consul

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/webitel/webitel-go-kit/infra/discovery"
)

type mockWatcher struct {
	event chan struct{}
}

func TestServiceSet_Broadcast(t *testing.T) {
	ss := &serviceSet{
		watcher:  make(map[*watcher]struct{}),
		services: &atomic.Value{},
	}

	w1 := &watcher{event: make(chan struct{}, 1)}
	w2 := &watcher{event: make(chan struct{}, 1)}

	ss.watcher[w1] = struct{}{}
	ss.watcher[w2] = struct{}{}

	svcInstances := []*discovery.ServiceInstance{
		{Id: "1", Name: "svc1"},
		{Id: "2", Name: "svc2"},
	}

	ss.broadcast(svcInstances)

	got := ss.services.Load().([]*discovery.ServiceInstance)
	assert.Equal(t, svcInstances, got, "services cache should be updated")

	select {
	case <-w1.event:
	default:
		t.Error("w1 should receive signal")
	}

	select {
	case <-w2.event:
	default:
		t.Error("w2 should receive signal")
	}
}

func TestServiceSet_Broadcast_ChannelNonBlocking(t *testing.T) {
	ss := &serviceSet{
		watcher:  make(map[*watcher]struct{}),
		services: &atomic.Value{},
	}

	w := &watcher{event: make(chan struct{}, 1)}
	ss.watcher[w] = struct{}{}

	w.event <- struct{}{}

	done := make(chan struct{})
	go func() {
		ss.broadcast(nil)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("broadcast blocked on full channel")
	}
}
