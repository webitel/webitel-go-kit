package sdnotify

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/webitel/webitel-go-kit/infra/health"
)

// Run under -race. Most of these assert only "no race, no panic, no deadlock,
// no leak" — the detector and the test timeout are the assertions.

func goroutinesSettled(t *testing.T, want int) int {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	got := runtime.NumGoroutine()
	for time.Now().Before(deadline) {
		got = runtime.NumGoroutine()
		if got <= want {
			return got
		}
		time.Sleep(2 * time.Millisecond)
	}

	return got
}

func TestRaceConcurrentStartAndStop(t *testing.T) {
	// A signal handler racing the startup path. Both sync.Once uses have to
	// order this without a mutex.
	for round := 0; round < 15; round++ {
		addr, _ := notifySocket(t)
		reg := newTestRegistry(t, fastConfig())
		reg.Critical("db", func(context.Context) error { return nil })
		n := newNotifier(t, reg, addr)

		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()

				if i%2 == 0 {
					_ = n.Start(ctx)

					return
				}
				_ = n.Stop(ctx)
			}(i)
		}
		wg.Wait()
	}
}

func TestRaceRepeatedStopReturnsTheSameVerdict(t *testing.T) {
	addr, _ := notifySocket(t)
	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", func(context.Context) error { return nil })
	n := newNotifier(t, reg, addr)

	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	var mu sync.Mutex
	seen := map[string]int{}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := n.Stop(ctx)
			key := "nil"
			if err != nil {
				key = err.Error()
			}
			mu.Lock()
			seen[key]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(seen) != 1 {
		t.Fatalf("concurrent Stops disagreed: %v", seen)
	}
}

func TestRaceNotifierCyclesDoNotLeakGoroutines(t *testing.T) {
	base := runtime.NumGoroutine()

	for i := 0; i < 20; i++ {
		addr, _ := notifySocket(t)

		// Own the registry's lifecycle here rather than via newTestRegistry:
		// its t.Cleanup fires at test end, so every iteration's checks would
		// still be running and this test would measure them as a leak.
		reg := health.New(fastConfig(), nil)
		reg.Critical("db", func(context.Context) error { return nil })
		if err := reg.Start(context.Background()); err != nil {
			t.Fatalf("cycle %d: registry Start: %v", i, err)
		}

		t.Setenv("NOTIFY_SOCKET", addr)
		n := New(reg)
		if n == nil {
			t.Fatal("New returned nil with NOTIFY_SOCKET set")
		}
		if err := n.Start(context.Background()); err != nil {
			t.Fatalf("cycle %d: Start: %v", i, err)
		}
		time.Sleep(2 * time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := n.Stop(ctx); err != nil {
			t.Fatalf("cycle %d: Stop: %v", i, err)
		}
		if err := reg.Stop(ctx); err != nil {
			t.Fatalf("cycle %d: registry Stop: %v", i, err)
		}
		cancel()
	}

	if got := goroutinesSettled(t, base+4); got > base+4 {
		t.Fatalf("goroutines %d -> %d after 20 notifier cycles", base, got)
	}
}

func TestRaceParentContextCancelRacesStop(t *testing.T) {
	base := runtime.NumGoroutine()

	for i := 0; i < 15; i++ {
		addr, _ := notifySocket(t)

		// Own the registry's lifecycle here — see the note in the cycles test.
		reg := health.New(fastConfig(), nil)
		reg.Critical("db", func(context.Context) error { return nil })
		if err := reg.Start(context.Background()); err != nil {
			t.Fatalf("registry Start: %v", err)
		}

		t.Setenv("NOTIFY_SOCKET", addr)
		n := New(reg)

		ctx, cancel := context.WithCancel(context.Background())
		if err := n.Start(ctx); err != nil {
			t.Fatalf("Start: %v", err)
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); cancel() }()
		go func() {
			defer wg.Done()
			sctx, scancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer scancel()
			n.Stop(sctx)
		}()
		wg.Wait()
		cancel()

		sctx, scancel := context.WithTimeout(context.Background(), 2*time.Second)
		reg.Stop(sctx)
		scancel()
	}

	if got := goroutinesSettled(t, base+4); got > base+4 {
		t.Fatalf("goroutines %d -> %d after 15 cancel/stop races", base, got)
	}
}

func TestRaceRegistryChurnsWhileTheNotifierRuns(t *testing.T) {
	// Registration collides with the notifier's poll loop, which snapshots on
	// every tick and sends on every state change.
	addr, ln := notifySocket(t)
	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", func(context.Context) error { return nil })

	n := newNotifier(t, reg, addr, WithPollInterval(time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			reg.Informational("late"+strconv.Itoa(i), func(context.Context) error { return nil })
			time.Sleep(time.Millisecond)
		}
	}()

	got := recvAll(t, ln, 50*time.Millisecond, 5*time.Second)
	wg.Wait()

	for _, d := range got {
		if strings.Count(d, "STATUS=") > 1 {
			t.Fatalf("a datagram carried two STATUS= lines: %q", d)
		}
		for _, line := range strings.Split(d, "\n") {
			if len(line) > 1024+len("STATUS=") {
				t.Fatalf("a line exceeded the cap: %d bytes", len(line))
			}
		}
	}
}

func TestRaceNilNotifierUnderConcurrency(t *testing.T) {
	// No NOTIFY_SOCKET: New returns nil and every method must stay a no-op.
	t.Setenv("NOTIFY_SOCKET", "")

	reg := newTestRegistry(t, fastConfig())
	n := New(reg)
	if n != nil {
		t.Fatal("New returned non-nil without NOTIFY_SOCKET")
	}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := n.Start(context.Background()); err != nil {
				panic("nil notifier Start: " + err.Error())
			}
			if err := n.Stop(context.Background()); err != nil {
				panic("nil notifier Stop: " + err.Error())
			}
		}()
	}
	wg.Wait()
}

// TestNoReadyWithOnlyInformationalChecks is the sd_notify half of the spec
// violation found in review: a registry holding only informational checks
// announced READY=1 to systemd before any check had run.
func TestNoReadyWithOnlyInformationalChecks(t *testing.T) {
	addr, ln := notifySocket(t)

	reg := health.New(fastConfig(), nil)
	reg.Informational("rabbit", func(context.Context) error { return nil })

	n := newNotifier(t, reg, addr, WithPollInterval(time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	for _, d := range recvAll(t, ln, 40*time.Millisecond, 3*time.Second) {
		if strings.Contains(d, "READY=1") {
			t.Fatalf("systemd was told READY=1 with only informational checks: %q", d)
		}
	}
}
