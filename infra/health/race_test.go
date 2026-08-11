package health

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// These tests exist to be run under -race. Most of them assert nothing beyond
// "no race, no panic, no deadlock, no leak" — the detector and the test timeout
// are the assertions. CI runs `go test -race` on every module, so a regression
// here fails the build rather than surfacing in production.

// hammer runs fn in n goroutines, released together so they collide, and waits.
func hammer(n int, fn func(i int)) {
	var start sync.WaitGroup
	var done sync.WaitGroup

	start.Add(1)
	for i := 0; i < n; i++ {
		done.Add(1)
		go func(i int) {
			defer done.Done()
			start.Wait()
			fn(i)
		}(i)
	}

	start.Done()
	done.Wait()
}

// goroutinesSettled waits for the goroutine count to come back to at most want,
// so a slow-exiting goroutine is not reported as a leak.
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

func racyConfig() Config {
	return Config{
		Interval:      time.Millisecond,
		Timeout:       500 * time.Microsecond,
		FailThreshold: 1,
		MinUnready:    time.Microsecond,
		StaleAfter:    50 * time.Millisecond,
		DrainHold:     time.Millisecond,
	}
}

func TestRaceRegisterWhileRunning(t *testing.T) {
	// Engine registers dependencies one at a time as they come up, so
	// registration collides with the scheduler by design.
	reg := start(t, racyConfig())

	hammer(64, func(i int) {
		reg.Critical("c"+strconv.Itoa(i), func(context.Context) error { return nil })
	})

	waitFor(t, "every check to run", func() bool {
		for _, r := range reg.results() {
			if r.Status != StatusOK {
				return false
			}
		}

		return len(reg.results()) == 64
	})
}

func TestRaceRegisterAndReadConcurrently(t *testing.T) {
	// Registration takes the write path while Snapshot/ReadyFunc take the read
	// path, all under the one mutex.
	reg := start(t, racyConfig())
	ready := reg.ReadyFunc()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				s := reg.Snapshot()
				_ = s.State.String()
				for _, c := range s.Checks {
					_ = c.Name + c.Status.String() + c.Group.String()
				}
				ok, err := ready()
				if !ok && err == nil {
					panic("invariant broken: false verdict with a nil error")
				}
			}
		}()
	}

	hammer(48, func(i int) {
		reg.Informational("late"+strconv.Itoa(i), func(context.Context) error { return nil })
	})

	time.Sleep(20 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestRaceDuplicateRegistrationIsSerialised(t *testing.T) {
	// Every goroutine registers the same name; exactly one must win.
	reg := New(racyConfig(), nil)

	hammer(32, func(i int) {
		reg.Critical("db", func(context.Context) error { return nil })
	})

	if got := len(reg.results()); got != 1 {
		t.Fatalf("got %d checks named db, want 1", got)
	}
}

func TestRaceConcurrentDrainAndStop(t *testing.T) {
	// A signal handler and a shutdown path can both fire; neither may panic and
	// Stop must stay idempotent.
	for round := 0; round < 20; round++ {
		reg := New(racyConfig(), nil)
		reg.Critical("db", func(context.Context) error { return nil })
		if err := reg.Start(context.Background()); err != nil {
			t.Fatalf("Start: %v", err)
		}

		var errs [8]error
		hammer(8, func(i int) {
			if i%2 == 0 {
				reg.Drain()

				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			errs[i] = reg.Stop(ctx)
		})

		for i, err := range errs {
			if err != nil {
				t.Fatalf("round %d: Stop from goroutine %d: %v", round, i, err)
			}
		}
		if s := reg.Snapshot(); s.State != StateStopping {
			t.Fatalf("round %d: state after Drain+Stop = %s, want %s", round, s.State, NameStopping)
		}
	}
}

func TestRaceStartStopCyclesDoNotLeakGoroutines(t *testing.T) {
	base := runtime.NumGoroutine()

	for i := 0; i < 30; i++ {
		reg := New(racyConfig(), nil)
		for j := 0; j < 8; j++ {
			reg.Critical("c"+strconv.Itoa(j), func(context.Context) error { return nil })
		}
		if err := reg.Start(context.Background()); err != nil {
			t.Fatalf("Start: %v", err)
		}
		time.Sleep(2 * time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := reg.Stop(ctx); err != nil {
			t.Fatalf("cycle %d: Stop: %v", i, err)
		}
		cancel()
	}

	if got := goroutinesSettled(t, base+2); got > base+2 {
		t.Fatalf("goroutines %d -> %d after 30 start/stop cycles with 8 checks each", base, got)
	}
}

func TestRaceParentContextCancelRacesStop(t *testing.T) {
	// The parent ctx dying and an explicit Stop are two independent shutdown
	// triggers; together they must still terminate cleanly.
	base := runtime.NumGoroutine()

	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		reg := New(racyConfig(), nil)
		reg.Critical("db", func(c context.Context) error { <-c.Done(); return c.Err() })
		if err := reg.Start(ctx); err != nil {
			t.Fatalf("Start: %v", err)
		}

		hammer(2, func(i int) {
			if i == 0 {
				cancel()

				return
			}
			sctx, scancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer scancel()
			reg.Stop(sctx)
		})

		cancel()
	}

	if got := goroutinesSettled(t, base+2); got > base+2 {
		t.Fatalf("goroutines %d -> %d after 20 cancel/stop races", base, got)
	}
}

func TestRacePanickingCheckDoesNotCorruptTheRegistry(t *testing.T) {
	// A panic becomes that check's error. Under concurrency it must not take
	// the process down or wedge the mutex.
	reg := start(t, racyConfig())

	for i := 0; i < 16; i++ {
		name := "boom" + strconv.Itoa(i)
		reg.Critical(name, func(context.Context) error { panic("kaboom " + name) })
	}
	reg.Informational("fine", func(context.Context) error { return nil })

	waitFor(t, "the healthy check to pass despite the panics", func() bool {
		for _, r := range reg.results() {
			if r.Name == "fine" && r.Status == StatusOK {
				return true
			}
		}

		return false
	})

	ok, err := reg.ReadyFunc()()
	if ok {
		t.Fatal("verdict is true with sixteen panicking critical checks")
	}
	if err == nil {
		t.Fatal("false verdict with a nil error")
	}
}

func TestRaceReadyFuncClosureOutlivesTheRegistry(t *testing.T) {
	// Consul holds the closure for the process lifetime; it must stay safe
	// after Stop and must never return (false, nil).
	reg := New(racyConfig(), nil)
	reg.Critical("db", func(context.Context) error { return errors.New("down") })
	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	ready := reg.ReadyFunc()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := reg.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	hammer(32, func(int) {
		for i := 0; i < 50; i++ {
			ok, err := ready()
			if !ok && err == nil {
				panic("invariant broken after Stop: false verdict with a nil error")
			}
		}
	})
}

func TestRaceSnapshotIsAnIndependentCopy(t *testing.T) {
	// Transports hold a Snapshot while the scheduler keeps recording; mutating
	// or reading the returned slice must not touch registry state.
	reg := start(t, racyConfig())
	for i := 0; i < 16; i++ {
		reg.Critical("c"+strconv.Itoa(i), func(context.Context) error { return nil })
	}

	waitFor(t, "checks to appear", func() bool { return len(reg.Snapshot().Checks) == 16 })

	hammer(16, func(int) {
		for i := 0; i < 100; i++ {
			s := reg.Snapshot()
			for j := range s.Checks {
				s.Checks[j].Name = "clobbered"
				s.Checks[j].Status = StatusFail
			}
		}
	})

	for _, c := range reg.Snapshot().Checks {
		if c.Name == "clobbered" {
			t.Fatal("a caller mutating Snapshot().Checks corrupted the registry")
		}
	}
}

func TestRaceHysteresisUnderConcurrentReads(t *testing.T) {
	// A flapping check drives status transitions while readers hammer the
	// snapshot — the classic shape for a torn read.
	var flip atomic.Bool

	cfg := racyConfig()
	cfg.FailThreshold = 2
	reg := start(t, cfg)
	reg.Critical("flappy", func(context.Context) error {
		if flip.Load() {
			return errors.New("down")
		}

		return nil
	})

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				s := reg.Snapshot()
				for _, c := range s.Checks {
					// Since must never be in the future of the snapshot.
					if !c.Since.IsZero() && c.Since.After(s.Time) {
						panic(fmt.Sprintf("check %q Since %v is after Snapshot.Time %v", c.Name, c.Since, s.Time))
					}
					if c.Status != StatusOK && c.Err == nil {
						panic("check " + c.Name + " is " + c.Status.String() + " with a nil Err")
					}
				}
			}
		}()
	}

	for i := 0; i < 60; i++ {
		flip.Store(i%2 == 0)
		time.Sleep(time.Millisecond)
	}

	close(stop)
	wg.Wait()
}

func TestRaceNilRegistryUnderConcurrency(t *testing.T) {
	var reg *Registry

	hammer(32, func(i int) {
		reg.Critical("db", func(context.Context) error { return nil })
		reg.Drain()
		_ = reg.Start(context.Background())
		_ = reg.Stop(context.Background())
		s := reg.Snapshot()
		_ = s.State.String()
		ok, err := reg.ReadyFunc()()
		if !ok && err == nil {
			panic("nil registry returned a false verdict with a nil error")
		}
	})
}

func TestRaceCheckIgnoringContextDoesNotBlockShutdown(t *testing.T) {
	// The pathological consumer: a check that never returns. Stop must still
	// come back, and must say so honestly.
	release := make(chan struct{})
	defer close(release)

	reg := New(racyConfig(), nil)
	for i := 0; i < 8; i++ {
		reg.Critical("hung"+strconv.Itoa(i), func(context.Context) error { <-release; return nil })
	}
	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- reg.Stop(ctx) }()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop hung on checks that ignore their context")
	}
}
