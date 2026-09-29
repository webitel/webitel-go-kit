package health

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type recordingTransport struct {
	name     string
	startErr error
	stopErr  error
	log      *[]string
	mu       *sync.Mutex
}

func (s recordingTransport) record(event string) {
	s.mu.Lock()
	*s.log = append(*s.log, event+":"+s.name)
	s.mu.Unlock()
}

func (s recordingTransport) Start(context.Context, *Registry) error {
	s.record("start")

	return s.startErr
}

func (s recordingTransport) Stop(context.Context) error {
	s.record("stop")

	return s.stopErr
}

type nilableTransport struct{}

func (n *nilableTransport) Start(context.Context, *Registry) error {
	panic("Start called on a nil *nilableTransport")
}

func (n *nilableTransport) Stop(context.Context) error {
	panic("Stop called on a nil *nilableTransport")
}

// testConfig scales every timing down so the scheduler's behaviour is visible
// in milliseconds. Nothing here is faked: these are the real tickers and the
// real contexts, which is the point under -race.
func testConfig() Config {
	return Config{
		Interval:      20 * time.Millisecond,
		Timeout:       10 * time.Millisecond,
		FailThreshold: 1,
		MinUnready:    time.Millisecond,
		StaleAfter:    2 * time.Second,
		DrainHold:     time.Millisecond,
	}
}

// waitFor polls until cond holds, or fails the test.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", what)
}

// start builds a started registry and stops it when the test ends.
func start(t *testing.T, cfg Config) *Registry {
	t.Helper()

	reg := New(cfg, nil)
	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		reg.Stop(ctx)
	})

	return reg
}

func find(reg *Registry, name string) CheckResult {
	for _, res := range reg.results() {
		if res.Name == name {
			return res
		}
	}

	return CheckResult{}
}

func snapshotLog(mu *sync.Mutex, log *[]string) []string {
	mu.Lock()
	defer mu.Unlock()

	return append([]string(nil), *log...)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestCheckRunsInBackground(t *testing.T) {
	reg := start(t, testConfig())
	reg.Critical("ok", func(context.Context) error { return nil })

	waitFor(t, "check to pass", func() bool { return find(reg, "ok").Status == StatusOK })
}

func TestRegistrationWorksAfterStart(t *testing.T) {
	// Engine brings its dependencies up one at a time, so it cannot hand the
	// registry a complete set at construction.
	reg := start(t, testConfig())

	waitFor(t, "an empty registry", func() bool { return len(reg.results()) == 0 })

	reg.Informational("late", func(context.Context) error { return nil })

	waitFor(t, "the late check to run", func() bool { return find(reg, "late").Status == StatusOK })
}

func TestPanicBecomesTheChecksError(t *testing.T) {
	reg := start(t, testConfig())
	reg.Critical("boom", func(context.Context) error { panic("kaboom") })

	waitFor(t, "the panic to be recorded", func() bool { return find(reg, "boom").Status == StatusFail })

	err := find(reg, "boom").Err
	if err == nil || !strings.Contains(err.Error(), "panicked") {
		t.Fatalf("got %v, want an error mentioning the panic", err)
	}
}

func TestRunsNeverOverlap(t *testing.T) {
	// The check below ignores its context, which is exactly the case the
	// "one run at a time" rule exists for: a context deadline does not kill a
	// goroutine, so if the scheduler moved on at the deadline these would pile
	// up and drain the very pool the check is probing.
	var runs atomic.Int32
	release := make(chan struct{})
	defer close(release)

	reg := start(t, testConfig())
	reg.Critical("hung", func(context.Context) error {
		runs.Add(1)
		<-release

		return nil
	})

	waitFor(t, "the check to start", func() bool { return runs.Load() == 1 })

	// Give the scheduler many intervals in which it could wrongly start more.
	time.Sleep(20 * testConfig().Interval)

	if got := runs.Load(); got != 1 {
		t.Fatalf("the check started %d times while the first was still running, want 1", got)
	}
}

func TestSlowRunsAreStillSpacedByAnInterval(t *testing.T) {
	// A Ticker would buffer a tick during a slower-than-Interval run and
	// re-run the check back to back — hammering the dependency the moment it
	// gets slow, which is the opposite of what a health check should do.
	cfg := testConfig()
	sleep := 3 * cfg.Interval

	var mu sync.Mutex
	var starts []time.Time

	reg := start(t, cfg)
	reg.Critical("slow", func(context.Context) error {
		mu.Lock()
		starts = append(starts, time.Now())
		mu.Unlock()
		time.Sleep(sleep)

		return nil
	})

	waitFor(t, "three runs", func() bool {
		mu.Lock()
		defer mu.Unlock()

		return len(starts) >= 3
	})

	mu.Lock()
	defer mu.Unlock()
	for i := 1; i < 3; i++ {
		// Every gap must be at least run duration + Interval; a buffered
		// tick would make it just the run duration.
		if want := sleep + cfg.Interval/2; starts[i].Sub(starts[i-1]) < want {
			t.Fatalf("runs %d and %d started %s apart, want at least %s",
				i-1, i, starts[i].Sub(starts[i-1]), want)
		}
	}
}

func TestSlowCheckIsRecordedAsFailed(t *testing.T) {
	// A well-behaved slow check returns the deadline error it was handed.
	reg := start(t, testConfig())
	reg.Critical("slow", func(ctx context.Context) error {
		<-ctx.Done()

		return ctx.Err()
	})

	waitFor(t, "the timeout to be recorded", func() bool { return find(reg, "slow").Status == StatusFail })
}

func TestFinishingLateIsNotPassing(t *testing.T) {
	// A check that ignores its context and "succeeds" anyway — a query that
	// came back on the timeout's fourth try, say — must not count as passing.
	cfg := testConfig()

	reg := start(t, cfg)
	reg.Critical("late", func(context.Context) error {
		time.Sleep(3 * cfg.Timeout)

		return nil
	})

	waitFor(t, "the late pass to be recorded as a failure", func() bool {
		return find(reg, "late").Status == StatusFail
	})

	err := find(reg, "late").Err
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("got %v, want an error mentioning the timeout", err)
	}
}

func TestAHangingCheckGoesStaleRatherThanFailing(t *testing.T) {
	// A check that stops returning is never recorded again, so its
	// consecutive-failure count cannot reach FailThreshold and it never
	// becomes StatusFail. Staleness carries the verdict instead — and
	// StatusUnknown is not healthy, so the node still leaves rotation.
	cfg := testConfig()
	cfg.FailThreshold = 3
	cfg.StaleAfter = 100 * time.Millisecond

	var runs atomic.Int32
	release := make(chan struct{})
	defer close(release)

	reg := start(t, cfg)
	reg.Critical("db", func(context.Context) error {
		if runs.Add(1) > 1 {
			<-release // healthy once, then hangs forever
		}

		return nil
	})

	waitFor(t, "the healthy first run", func() bool { return find(reg, "db").Status == StatusOK })
	waitFor(t, "the result to go stale", func() bool { return find(reg, "db").Status == StatusUnknown })
}

func TestStopIsNotBlockedByAHungCheck(t *testing.T) {
	var began atomic.Bool
	release := make(chan struct{})
	defer close(release)

	reg := New(testConfig(), nil)
	reg.Critical("hung", func(context.Context) error {
		began.Store(true)
		<-release

		return nil
	})
	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	waitFor(t, "the check to be running", func() bool { return began.Load() })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- reg.Stop(ctx) }()

	select {
	case err := <-done:
		// Stop must return — and must not pretend the shutdown was clean.
		if err == nil || !strings.Contains(err.Error(), "still running") {
			t.Fatalf("got %v, want an error reporting checks still running", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Stop hung on a check that ignores its context")
	}
}

func TestStopAbsorbsTheDrainHold(t *testing.T) {
	cfg := testConfig()
	cfg.DrainHold = 200 * time.Millisecond

	reg := New(cfg, nil)
	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	reg.Drain()

	began := time.Now()
	if err := reg.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if held := time.Since(began); held < cfg.DrainHold/2 {
		t.Fatalf("Stop returned after %s, want it to hold for about %s", held, cfg.DrainHold)
	}
}

func TestDrainHoldIsBoundedByTheContext(t *testing.T) {
	cfg := testConfig()
	cfg.DrainHold = 10 * time.Second

	reg := New(cfg, nil)
	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	reg.Drain()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	began := time.Now()
	reg.Stop(ctx)

	if held := time.Since(began); held > time.Second {
		t.Fatalf("Stop held for %s despite the context deadline", held)
	}
}

func TestStopWithoutDrainDoesNotHold(t *testing.T) {
	cfg := testConfig()
	cfg.DrainHold = 10 * time.Second

	reg := New(cfg, nil)
	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	began := time.Now()
	if err := reg.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if held := time.Since(began); held > time.Second {
		t.Fatalf("Stop held for %s without a preceding Drain", held)
	}
}

func TestNoRunAfterTheContextDies(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	reg := New(testConfig(), nil)
	if err := reg.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	cancel()

	var runs atomic.Int32
	reg.Critical("late", func(context.Context) error {
		runs.Add(1)

		return nil
	})

	time.Sleep(50 * time.Millisecond)

	if got := runs.Load(); got != 0 {
		t.Fatalf("check ran %d times after the context died, want 0", got)
	}
}

func TestSecondStopWaitsForTheFirst(t *testing.T) {
	cfg := testConfig()
	cfg.DrainHold = 300 * time.Millisecond

	reg := New(cfg, nil)
	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	reg.Drain()

	go reg.Stop(context.Background()) // absorbs the DrainHold

	time.Sleep(50 * time.Millisecond)

	began := time.Now()
	if err := reg.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if waited := time.Since(began); waited < cfg.DrainHold/3 {
		t.Fatalf("second Stop returned after %s, before the first finished its %s hold", waited, cfg.DrainHold)
	}
}

func TestStopOnNeverStartedRegistrySkipsTheDrainHold(t *testing.T) {
	cfg := testConfig()
	cfg.DrainHold = 10 * time.Second

	reg := New(cfg, nil)
	reg.Drain()

	began := time.Now()
	if err := reg.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if held := time.Since(began); held > time.Second {
		t.Fatalf("Stop held %s on a registry that never started", held)
	}
}

func TestDuplicateNameIsIgnoredNotReplaced(t *testing.T) {
	reg := New(testConfig(), nil)
	reg.Critical("db", func(context.Context) error { return nil })
	reg.Informational("db", func(context.Context) error { return nil })

	results := reg.results()
	if len(results) != 1 {
		t.Fatalf("got %d checks, want 1", len(results))
	}
	if results[0].Group != GroupCritical {
		t.Fatalf("the duplicate replaced the original: group is %s", results[0].Group)
	}
}

func TestUnusableRegistrationsAreIgnored(t *testing.T) {
	reg := New(testConfig(), nil)
	reg.Critical("", func(context.Context) error { return nil })
	reg.Critical("nil-fn", nil)

	if got := len(reg.results()); got != 0 {
		t.Fatalf("got %d checks, want 0", got)
	}
}

func TestRegistrationAfterStopIsIgnored(t *testing.T) {
	reg := New(testConfig(), nil)
	if err := reg.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	reg.Critical("late", func(context.Context) error { return nil })

	if got := len(reg.results()); got != 0 {
		t.Fatalf("got %d checks, want 0", got)
	}
}

func TestStartIsNotRepeatable(t *testing.T) {
	reg := start(t, testConfig())

	if err := reg.Start(context.Background()); err == nil {
		t.Fatal("a second Start succeeded, want an error")
	}
}

func TestStartAfterStopFails(t *testing.T) {
	reg := New(testConfig(), nil)
	if err := reg.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if err := reg.Start(context.Background()); err == nil {
		t.Fatal("Start after Stop succeeded, want an error")
	}
}

func TestStopIsRepeatable(t *testing.T) {
	reg := start(t, testConfig())

	for i := 0; i < 3; i++ {
		if err := reg.Stop(context.Background()); err != nil {
			t.Fatalf("Stop %d: %v", i, err)
		}
	}
}

func TestNilRegistryIsSafe(t *testing.T) {
	var reg *Registry

	reg.Critical("db", func(context.Context) error { return nil })
	reg.Liveness("live", func(context.Context) error { return nil })
	reg.Informational("info", func(context.Context) error { return nil })
	reg.Drain()

	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := reg.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestTransitionsSurviveAFlap(t *testing.T) {
	reg := start(t, testConfig())

	var down atomic.Bool
	var runs atomic.Int64
	reg.Critical("flappy", func(context.Context) error {
		runs.Add(1)
		time.Sleep(time.Millisecond)
		if down.Load() {
			return errBoom
		}

		return nil
	})

	waitFor(t, "the first pass", func() bool { return find(reg, "flappy").Status == StatusOK })
	if got := find(reg, "flappy"); got.Transitions != (Transitions{OK: 1}) || got.LastDuration < time.Millisecond {
		t.Fatalf("after the first pass got %+v, want one transition into ok and a measured duration", got)
	}

	// Transitions count status changes, not runs.
	at := runs.Load()
	waitFor(t, "three more runs", func() bool { return runs.Load() >= at+3 })
	if got := find(reg, "flappy").Transitions; got != (Transitions{OK: 1}) {
		t.Fatalf("a stable check accrued transitions: %+v", got)
	}

	down.Store(true)
	waitFor(t, "the failure", func() bool { return find(reg, "flappy").Status == StatusFail })
	if got := find(reg, "flappy").Transitions; got != (Transitions{OK: 1, Fail: 1}) {
		t.Fatalf("after failing Transitions = %+v", got)
	}

	down.Store(false)
	waitFor(t, "the recovery", func() bool { return find(reg, "flappy").Status == StatusOK })
	if got := find(reg, "flappy").Transitions; got != (Transitions{OK: 2, Fail: 1}) {
		t.Fatalf("after recovery Transitions = %+v", got)
	}
}

func TestPartialConfigIsFilledIn(t *testing.T) {
	got := Config{Interval: time.Second}.withDefaults()
	def := DefaultConfig()

	if got.Interval != time.Second {
		t.Errorf("Interval = %s, want it left alone", got.Interval)
	}
	if got.Timeout != def.Timeout || got.FailThreshold != def.FailThreshold {
		t.Errorf("got %+v, want the unset fields from DefaultConfig", got)
	}
}

func TestShutdownOrder(t *testing.T) {
	var (
		mu  sync.Mutex
		log []string
	)

	a := recordingTransport{name: "a", log: &log, mu: &mu}
	b := recordingTransport{name: "b", log: &log, mu: &mu}

	reg := New(testConfig(), nil, WithTransport(a), WithTransport(b))
	reg.Critical("c", func(context.Context) error { return nil })

	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := reg.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	want := []string{"start:a", "start:b", "stop:b", "stop:a"}
	if got := snapshotLog(&mu, &log); !equalStrings(got, want) {
		t.Errorf("transport events = %v, want %v", got, want)
	}

	if s := reg.Snapshot().State; s != StateStopping {
		t.Errorf("state = %v, want stopping", s)
	}
}

func TestStopStopsEveryTransportDespiteErrors(t *testing.T) {
	var (
		mu  sync.Mutex
		log []string
	)

	boom := errors.New("boom")
	a := recordingTransport{name: "a", stopErr: boom, log: &log, mu: &mu}
	b := recordingTransport{name: "b", log: &log, mu: &mu}

	reg := New(testConfig(), nil, WithTransport(a), WithTransport(b))

	err := reg.Shutdown(context.Background())
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want it to wrap boom", err)
	}

	if got := snapshotLog(&mu, &log); len(got) != 2 {
		t.Errorf("events = %v, want both transports stopped — a failure must not skip the rest", got)
	}
}

func TestStartFailureRollsBackAndJoinsErrors(t *testing.T) {
	var (
		mu  sync.Mutex
		log []string
	)

	startBoom := errors.New("start boom")
	stopBoom := errors.New("stop boom")
	a := recordingTransport{name: "a", stopErr: stopBoom, log: &log, mu: &mu}
	b := recordingTransport{name: "b", startErr: startBoom, log: &log, mu: &mu}
	c := recordingTransport{name: "c", log: &log, mu: &mu}

	reg := New(testConfig(), nil, WithTransport(a), WithTransport(b), WithTransport(c))
	reg.Critical("c", func(context.Context) error { return nil })

	err := reg.Start(context.Background())
	if !errors.Is(err, startBoom) || !errors.Is(err, stopBoom) {
		t.Fatalf("err = %v, want both the start and the rollback failure", err)
	}

	want := []string{"start:a", "start:b", "stop:a"}
	if got := snapshotLog(&mu, &log); !equalStrings(got, want) {
		t.Errorf("transport events = %v, want %v", got, want)
	}

	if s := reg.Snapshot().State; s != StateStopping {
		t.Errorf("state = %v, want stopping", s)
	}
	if err := reg.Start(context.Background()); err == nil {
		t.Error("Start after a failed Start succeeded, want an error")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := reg.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown after a failed Start: %v", err)
	}
}

func TestShutdownAbsorbsTheDrainHold(t *testing.T) {
	cfg := testConfig()
	cfg.DrainHold = 120 * time.Millisecond

	reg := New(cfg, nil)
	reg.Critical("c", func(context.Context) error { return nil })

	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	start := time.Now()
	if err := reg.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	if elapsed := time.Since(start); elapsed < cfg.DrainHold {
		t.Errorf("returned after %s, want at least the %s hold", elapsed, cfg.DrainHold)
	}
}

func TestShutdownOnNilRegistryIsSafe(t *testing.T) {
	var reg *Registry

	if err := reg.Shutdown(context.Background()); err != nil {
		t.Errorf("nil registry: %v", err)
	}
}

func TestWithTransportSkipsNil(t *testing.T) {
	reg := New(testConfig(), nil, WithTransport(nil))

	if err := reg.Shutdown(context.Background()); err != nil {
		t.Errorf("nil transport should be skipped, got %v", err)
	}
}

func TestWithTransportSkipsTypedNil(t *testing.T) {
	var typedNil *nilableTransport

	reg := New(testConfig(), nil, WithTransport(typedNil))

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("registry called a method on a typed nil: %v", r)
		}
	}()

	if err := reg.Start(context.Background()); err != nil {
		t.Errorf("Start: %v", err)
	}
	if err := reg.Shutdown(context.Background()); err != nil {
		t.Errorf("typed nil transport should be skipped, got %v", err)
	}
}
