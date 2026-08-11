package health

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// cr builds a CheckResult for driving the pure derivation helpers.
func cr(name string, group Group, status Status) CheckResult {
	return CheckResult{Name: name, Group: group, Status: status}
}

func TestZeroSnapshotReadsUnknown(t *testing.T) {
	// StateUnknown must be the iota-0 zero value: a zero Snapshot is never ready.
	var s Snapshot

	if s.State != StateUnknown {
		t.Fatalf("zero Snapshot state is %s, want %s", s.State, NameUnknown)
	}
	if s.State.Ready() {
		t.Fatal("zero Snapshot reports ready")
	}
}

func TestStateNames(t *testing.T) {
	for state, want := range map[State]string{
		StateUnknown:  NameUnknown,
		StateNotReady: NameNotReady,
		StateDegraded: NameDegraded,
		StateReady:    NameReady,
		StateStopping: NameStopping,
		State(200):    NameUnknown,
	} {
		if got := state.String(); got != want {
			t.Errorf("State(%d) = %q, want %q", state, got, want)
		}
	}
}

func TestVerdictTable(t *testing.T) {
	allOK := []CheckResult{
		cr("db", GroupCritical, StatusOK),
		cr("live", GroupLiveness, StatusOK),
		cr("s3", GroupInformational, StatusOK),
	}

	for _, tt := range []struct {
		name     string
		checks   []CheckResult
		draining bool
		stopped  bool
		want     State
		ready    bool
	}{
		{"draining", allOK, true, false, StateStopping, false},
		{"stopped without drain", allOK, false, true, StateStopping, false}, // E4
		{"zero checks", nil, false, false, StateUnknown, false},
		{"counting fail", []CheckResult{cr("db", GroupCritical, StatusFail)}, false, false, StateNotReady, false},
		{"counting unknown", []CheckResult{cr("db", GroupCritical, StatusUnknown)}, false, false, StateUnknown, false},
		{"informational fail only degrades", []CheckResult{cr("db", GroupCritical, StatusOK), cr("s3", GroupInformational, StatusFail)}, false, false, StateDegraded, true},
		{"informational unknown only degrades", []CheckResult{cr("db", GroupCritical, StatusOK), cr("s3", GroupInformational, StatusUnknown)}, false, false, StateDegraded, true}, // E2/E3
		{"all ok", allOK, false, false, StateReady, true},
		{"fail outranks unknown", []CheckResult{cr("a", GroupCritical, StatusUnknown), cr("b", GroupLiveness, StatusFail)}, false, false, StateNotReady, false}, // E1
		// Nothing registered answers "can this node take traffic?", so no verdict
		// is earned — the same rule as an empty registry. These two rows used to
		// assert ready/degraded, which was the defect: a registry holding only
		// informational checks reported Ready before any of them had run.
		{"only informational, all ok", []CheckResult{cr("s3", GroupInformational, StatusOK)}, false, false, StateUnknown, false},
		{"only informational, one failing", []CheckResult{cr("s3", GroupInformational, StatusFail)}, false, false, StateUnknown, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveState(tt.checks, tt.draining, tt.stopped)
			if got != tt.want {
				t.Fatalf("state = %s, want %s", got, tt.want)
			}
			if got.Ready() != tt.ready {
				t.Fatalf("%s.Ready() = %v, want %v", got, got.Ready(), tt.ready)
			}
		})
	}
}

func TestSnapshotChecksAreSortedAndTimeIsStamped(t *testing.T) {
	nop := func(context.Context) error { return nil }

	reg := New(testConfig(), nil)
	reg.Critical("b", nop)
	reg.Liveness("a", nop)
	reg.Informational("c", nop)

	s := reg.Snapshot()
	if s.Time.IsZero() {
		t.Fatal("Snapshot.Time is zero")
	}
	for i, want := range []string{"a", "b", "c"} {
		if s.Checks[i].Name != want {
			t.Fatalf("Checks[%d] = %q, want %q", i, s.Checks[i].Name, want)
		}
	}

	// results() is re-pointed at Snapshot().Checks; the two must agree.
	res := reg.results()
	if len(res) != len(s.Checks) {
		t.Fatalf("results() has %d checks, Snapshot().Checks has %d", len(res), len(s.Checks))
	}
	for i := range res {
		if res[i].Name != s.Checks[i].Name || res[i].Group != s.Checks[i].Group || res[i].Status != s.Checks[i].Status {
			t.Fatalf("results()[%d] = %+v, want %+v", i, res[i], s.Checks[i])
		}
	}
}

func TestColdStartIsUnknownNotReady(t *testing.T) {
	// E7: engine calls ReadyFunc synchronously at registration, before any
	// check has completed a run — this is its very first production call.
	release := make(chan struct{})
	defer close(release)

	reg := start(t, testConfig())
	reg.Critical("db", func(context.Context) error { <-release; return nil })

	if got := reg.Snapshot().State; got != StateUnknown {
		t.Fatalf("cold-start state = %s, want %s", got, NameUnknown)
	}

	ok, err := reg.ReadyFunc()()
	if ok {
		t.Fatal("cold-start verdict is true, want false")
	}
	if err == nil {
		t.Fatal("cold-start verdict carries a nil error")
	}
}

func TestDrainFlipsTheSnapshotToStopping(t *testing.T) {
	reg := New(testConfig(), nil)
	reg.Drain()

	s := reg.Snapshot()
	if s.State != StateStopping {
		t.Fatalf("state after Drain = %s, want %s", s.State, NameStopping)
	}
	if !s.Draining {
		t.Fatal("Snapshot.Draining is false after Drain")
	}
}

func TestStopWithoutDrainIsStopping(t *testing.T) {
	// E4: Stop never sets draining, but a stopped registry must not report
	// Ready for up to StaleAfter — stopped alone forces Stopping.
	reg := New(testConfig(), nil)
	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := reg.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	s := reg.Snapshot()
	if s.State != StateStopping {
		t.Fatalf("state after Stop = %s, want %s", s.State, NameStopping)
	}
	if s.Draining {
		t.Fatal("Snapshot.Draining is true without a Drain")
	}
}

// exact asserts the verdict error's whole text — the fixed strings engine
// shows in the Consul UI.
func exact(want string) func(*testing.T, error) {
	return func(t *testing.T, err error) {
		t.Helper()

		if got := err.Error(); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}

func TestFalseVerdictAlwaysCarriesAnError(t *testing.T) {
	// The reason this phase exists: engine/pkg/discovery/consul.go:129 calls
	// err.Error() unconditionally on the not-ok branch, so (false, nil) panics
	// the pilot consumer. Every false-producing configuration must pair the
	// false with a non-nil error.
	for _, tt := range []struct {
		name  string
		setup func(t *testing.T) *Registry
		check func(t *testing.T, err error)
	}{
		{
			name:  "nil registry",
			setup: func(t *testing.T) *Registry { return nil },
			check: exact("health: unknown: no checks registered"),
		},
		{
			name:  "never started, zero checks",
			setup: func(t *testing.T) *Registry { return New(testConfig(), nil) },
			check: exact("health: unknown: no checks registered"),
		},
		{
			name:  "started, zero checks",
			setup: func(t *testing.T) *Registry { return start(t, testConfig()) },
			check: exact("health: unknown: no checks registered"),
		},
		{
			name: "cold start: check registered, first run not recorded",
			setup: func(t *testing.T) *Registry {
				release := make(chan struct{})

				reg := start(t, testConfig())
				// Registered after start, so LIFO cleanup releases the check
				// before Stop tries to join it.
				t.Cleanup(func() { close(release) })
				reg.Critical("db", func(context.Context) error { <-release; return nil })

				return reg
			},
			check: exact("health: unknown: unknown [db]"),
		},
		{
			name: "critical failing",
			setup: func(t *testing.T) *Registry {
				reg := start(t, testConfig())
				reg.Critical("db", func(context.Context) error { return errors.New("sentinel-dsn-secret") })
				waitFor(t, "the check to fail", func() bool { return find(reg, "db").Status == StatusFail })

				return reg
			},
			check: func(t *testing.T, err error) {
				t.Helper()

				// The synthesis rule, asserted: state token and check name in,
				// the check's own error text never.
				msg := err.Error()
				if strings.Contains(msg, "sentinel-dsn-secret") {
					t.Fatalf("verdict error leaks the check's error text: %q", msg)
				}
				if !strings.Contains(msg, NameNotReady) || !strings.Contains(msg, "db") {
					t.Fatalf("got %q, want the %s token and the check name", msg, NameNotReady)
				}
			},
		},
		{
			name: "critical unknown via staleness",
			setup: func(t *testing.T) *Registry {
				cfg := testConfig()
				cfg.FailThreshold = 3
				cfg.StaleAfter = 100 * time.Millisecond

				var runs atomic.Int32
				release := make(chan struct{})

				reg := start(t, cfg)
				t.Cleanup(func() { close(release) })
				reg.Critical("db", func(context.Context) error {
					if runs.Add(1) > 1 {
						<-release // healthy once, then hangs forever
					}

					return nil
				})

				// Wait for OK first: a never-ran check is also unknown, and
				// with the same message, so skipping this makes the subtest
				// pass without the staleness transition ever happening.
				waitFor(t, "the healthy first run", func() bool { return find(reg, "db").Status == StatusOK })
				waitFor(t, "the result to go stale", func() bool { return find(reg, "db").Status == StatusUnknown })

				return reg
			},
			check: exact("health: unknown: unknown [db]"),
		},
		{
			name: "draining",
			setup: func(t *testing.T) *Registry {
				reg := New(testConfig(), nil)
				reg.Drain()

				return reg
			},
			check: exact("health: stopping: shutting down"),
		},
		{
			name: "stopped",
			setup: func(t *testing.T) *Registry {
				reg := New(testConfig(), nil)
				if err := reg.Stop(context.Background()); err != nil {
					t.Fatalf("Stop: %v", err)
				}

				return reg
			},
			check: exact("health: stopping: shutting down"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reg := tt.setup(t)

			fn := reg.ReadyFunc()
			if fn == nil {
				t.Fatal("ReadyFunc returned a nil closure")
			}

			ok, err := fn()
			if ok {
				t.Fatal("verdict is true, want false")
			}
			if err == nil {
				t.Fatal("false verdict with a nil error — engine would panic on err.Error()")
			}
			tt.check(t, err)
		})
	}
}

func TestNilRegistrySnapshotIsSafe(t *testing.T) {
	var reg *Registry

	s := reg.Snapshot()
	if s.State != StateUnknown {
		t.Fatalf("nil-registry state = %s, want %s", s.State, NameUnknown)
	}
	if s.Time.IsZero() {
		t.Fatal("nil-registry Snapshot.Time is zero")
	}
}

func TestSchedulerAlive(t *testing.T) {
	// D3, at both grace boundaries, driven with an explicit now.
	now := time.Now()
	stale := DefaultConfig().StaleAfter

	neverRan := CheckResult{Name: "never"}
	ranAt := func(at time.Time) CheckResult { return CheckResult{Name: "db", LastRun: at} }

	for _, tt := range []struct {
		name      string
		checks    []CheckResult
		started   bool
		startedAt time.Time
		want      bool
	}{
		{"zero checks, never started", nil, false, time.Time{}, true},
		{"zero checks, started long ago", nil, true, now.Add(-time.Hour), true},
		{"grace: started exactly StaleAfter ago, nothing ran", []CheckResult{neverRan}, true, now.Add(-stale), true},
		{"grace over: started just past StaleAfter, nothing ran", []CheckResult{neverRan}, true, now.Add(-stale - time.Millisecond), false},
		{"one run exactly StaleAfter ago", []CheckResult{neverRan, ranAt(now.Add(-stale))}, true, now.Add(-time.Hour), true},
		{"all runs older than StaleAfter", []CheckResult{ranAt(now.Add(-stale - time.Millisecond))}, true, now.Add(-time.Hour), false},
		{"never started, with checks", []CheckResult{neverRan}, false, time.Time{}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := schedulerAlive(now, tt.checks, tt.started, tt.startedAt, stale); got != tt.want {
				t.Fatalf("schedulerAlive = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSchedulerAliveLifecycle(t *testing.T) {
	reg := New(testConfig(), nil)
	reg.Critical("db", func(context.Context) error { return nil })

	// Never started with checks registered: nothing proves the scheduler turns.
	if reg.Snapshot().SchedulerAlive {
		t.Fatal("SchedulerAlive on a never-started registry with checks")
	}

	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		reg.Stop(ctx)
	})

	if !reg.Snapshot().SchedulerAlive {
		t.Fatal("SchedulerAlive false inside the cold-start grace window")
	}

	waitFor(t, "a completed run", func() bool { return !find(reg, "db").LastRun.IsZero() })

	if !reg.Snapshot().SchedulerAlive {
		t.Fatal("SchedulerAlive false with a run completed within StaleAfter")
	}
}
