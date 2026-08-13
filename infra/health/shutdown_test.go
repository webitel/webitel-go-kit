package health

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type recordingStopper struct {
	name string
	err  error
	log  *[]string
	mu   *sync.Mutex
}

func (s recordingStopper) Stop(context.Context) error {
	s.mu.Lock()
	*s.log = append(*s.log, s.name)
	s.mu.Unlock()

	return s.err
}

// The registry must be drained before any transport is stopped, and stopped
// last. Both mistakes fail silently in production.
func TestShutdownOrder(t *testing.T) {
	var (
		mu  sync.Mutex
		log []string
	)

	reg := New(testConfig(), nil)
	reg.Critical("c", func(context.Context) error { return nil })

	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	a := recordingStopper{name: "a", log: &log, mu: &mu}
	b := recordingStopper{name: "b", log: &log, mu: &mu}

	if err := Shutdown(context.Background(), reg, a, b); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	mu.Lock()
	got := append([]string(nil), log...)
	mu.Unlock()

	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("transports stopped in %v, want [a b]", got)
	}

	if s := reg.Snapshot().State; s != StateStopping {
		t.Errorf("state = %v, want stopping", s)
	}
}

func TestShutdownStopsEveryTransportDespiteErrors(t *testing.T) {
	var (
		mu  sync.Mutex
		log []string
	)

	reg := New(testConfig(), nil)

	boom := errors.New("boom")
	a := recordingStopper{name: "a", err: boom, log: &log, mu: &mu}
	b := recordingStopper{name: "b", log: &log, mu: &mu}

	err := Shutdown(context.Background(), reg, a, b)
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want it to wrap boom", err)
	}

	mu.Lock()
	n := len(log)
	mu.Unlock()

	if n != 2 {
		t.Errorf("stopped %d transports, want 2 — a failure must not skip the rest", n)
	}
}

// sdnotify.New returns a nil *Notifier when NOTIFY_SOCKET is unset, and callers
// pass it straight through.
func TestShutdownSkipsNilTransports(t *testing.T) {
	reg := New(testConfig(), nil)

	if err := Shutdown(context.Background(), reg, nil); err != nil {
		t.Errorf("nil transport should be skipped, got %v", err)
	}
}

// Drain returns at once; the hold is absorbed by Stop, so the caller's context
// has to cover it.
func TestShutdownAbsorbsTheDrainHold(t *testing.T) {
	cfg := testConfig()
	cfg.DrainHold = 120 * time.Millisecond

	reg := New(cfg, nil)
	reg.Critical("c", func(context.Context) error { return nil })

	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	start := time.Now()
	if err := Shutdown(context.Background(), reg); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	if elapsed := time.Since(start); elapsed < cfg.DrainHold {
		t.Errorf("returned after %s, want at least the %s hold", elapsed, cfg.DrainHold)
	}
}

func TestShutdownOnNilRegistryIsSafe(t *testing.T) {
	if err := Shutdown(context.Background(), nil); err != nil {
		t.Errorf("nil registry: %v", err)
	}
}

type nilableStopper struct{}

func (n *nilableStopper) Stop(context.Context) error {
	// A Stopper from outside this package need not tolerate a nil receiver.
	panic("Stop called on a nil *nilableStopper")
}

// sdnotify.New returns a typed nil *Notifier, and callers pass it straight
// through. A typed nil in an interface is not == nil.
func TestShutdownSkipsTypedNilTransports(t *testing.T) {
	reg := New(testConfig(), nil)

	var typedNil *nilableStopper

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Shutdown called Stop on a typed nil: %v", r)
		}
	}()

	if err := Shutdown(context.Background(), reg, typedNil); err != nil {
		t.Errorf("typed nil transport should be skipped, got %v", err)
	}
}
