package health

import (
	"errors"
	"testing"
	"time"
)

var errBoom = errors.New("boom")

// failTo drives cs to StatusFail and returns the moment it got there.
func failTo(t *testing.T, cs *checkState, at time.Time, cfg Config) {
	t.Helper()

	for i := 0; i < cfg.FailThreshold; i++ {
		cs.record(at, errBoom, cfg)
	}
	if cs.status != StatusFail {
		t.Fatalf("check did not reach %s after %d failures", NameFail, cfg.FailThreshold)
	}
}

func TestFirstSuccessIsImmediate(t *testing.T) {
	cfg := DefaultConfig()
	cs := &checkState{}

	status, changed := cs.record(time.Now(), nil, cfg)
	if status != StatusOK || !changed {
		t.Fatalf("got %s changed=%v, want %s changed=true", status, changed, NameOK)
	}
}

func TestFailThresholdMustBeConsecutive(t *testing.T) {
	cfg := DefaultConfig()
	cs := &checkState{}
	now := time.Now()

	for i := 1; i < cfg.FailThreshold; i++ {
		if status, _ := cs.record(now, errBoom, cfg); status == StatusFail {
			t.Fatalf("flipped to %s after %d failures, want %d", NameFail, i, cfg.FailThreshold)
		}
	}

	if status, changed := cs.record(now, errBoom, cfg); status != StatusFail || !changed {
		t.Fatalf("got %s changed=%v, want %s changed=true", status, changed, NameFail)
	}
}

func TestSingleFailureDoesNotUnseatOK(t *testing.T) {
	cfg := DefaultConfig()
	cs := &checkState{}
	now := time.Now()

	cs.record(now, nil, cfg)

	if status, changed := cs.record(now, errBoom, cfg); status != StatusOK || changed {
		t.Fatalf("one failure moved the check to %s", status)
	}
}

func TestRecoveryWaitsForMinUnready(t *testing.T) {
	cfg := DefaultConfig()
	cs := &checkState{}
	start := time.Now()

	failTo(t, cs, start, cfg)

	// A success inside the MinUnready window is not enough: an unstable
	// dependency must not be able to flap the node back into rotation.
	if status, _ := cs.record(start.Add(cfg.MinUnready-time.Second), nil, cfg); status != StatusFail {
		t.Fatalf("recovered before MinUnready elapsed, got %s", status)
	}

	if status, changed := cs.record(start.Add(cfg.MinUnready), nil, cfg); status != StatusOK || !changed {
		t.Fatalf("got %s changed=%v, want %s changed=true", status, changed, NameOK)
	}
}

func TestNotOKAlwaysCarriesAnError(t *testing.T) {
	// ReadyFunc's hard invariant leans on this one: engine calls err.Error()
	// unconditionally when the verdict is false, so a not-OK result with a
	// nil error is a panic waiting to happen.
	cfg := DefaultConfig()
	now := time.Now()

	never := &checkState{name: "never-ran"}

	stale := &checkState{name: "stale"}
	stale.record(now.Add(-cfg.StaleAfter-time.Second), nil, cfg)

	held := &checkState{name: "held-by-min-unready"}
	failTo(t, held, now, cfg)
	held.record(now, nil, cfg) // success inside the MinUnready window

	for _, cs := range []*checkState{never, stale, held} {
		res := cs.result(now, cfg)
		if res.Status == StatusOK {
			t.Errorf("%s: got %s, expected a not-OK state", cs.name, res.Status)

			continue
		}
		if res.Err == nil {
			t.Errorf("%s: status %s with nil Err", cs.name, res.Status)
		}
	}
}

func TestFailureCounterResetsOnSuccess(t *testing.T) {
	cfg := DefaultConfig()
	cs := &checkState{}
	now := time.Now()

	for i := 1; i < cfg.FailThreshold; i++ {
		cs.record(now, errBoom, cfg)
	}
	cs.record(now, nil, cfg)

	if status, _ := cs.record(now, errBoom, cfg); status == StatusFail {
		t.Fatalf("a success did not reset the consecutive-failure count")
	}
}

func TestResultIsUnknownBeforeFirstRun(t *testing.T) {
	cs := &checkState{name: "db"}

	if got := cs.result(time.Now(), DefaultConfig()).Status; got != StatusUnknown {
		t.Fatalf("got %s, want %s", got, NameUnknown)
	}
}

func TestStaleResultIsUnknownNotHealthy(t *testing.T) {
	cfg := DefaultConfig()
	cs := &checkState{name: "db"}
	now := time.Now()

	cs.record(now, nil, cfg)

	if got := cs.result(now.Add(cfg.StaleAfter), cfg).Status; got != StatusOK {
		t.Fatalf("went stale early: got %s, want %s", got, NameOK)
	}

	stale := cs.result(now.Add(cfg.StaleAfter+time.Second), cfg)
	if stale.Status != StatusUnknown {
		t.Fatalf("stale result reported as %s, want %s", stale.Status, NameUnknown)
	}
	if stale.Err == nil {
		t.Fatal("stale result carries no error")
	}
}

func TestResultCarriesLastError(t *testing.T) {
	cfg := DefaultConfig()
	cs := &checkState{name: "db"}
	now := time.Now()

	cs.record(now, errBoom, cfg)

	if got := cs.result(now, cfg).Err; !errors.Is(got, errBoom) {
		t.Fatalf("got %v, want %v", got, errBoom)
	}
}

func TestGroupNames(t *testing.T) {
	for group, want := range map[Group]string{
		GroupLiveness:      NameLiveness,
		GroupCritical:      NameCritical,
		GroupInformational: NameInformational,
		Group(200):         NameUnknown,
	} {
		if got := group.String(); got != want {
			t.Errorf("Group(%d) = %q, want %q", group, got, want)
		}
	}
}

func TestStatusNames(t *testing.T) {
	for status, want := range map[Status]string{
		StatusUnknown: NameUnknown,
		StatusOK:      NameOK,
		StatusFail:    NameFail,
		Status(200):   NameUnknown,
	} {
		if got := status.String(); got != want {
			t.Errorf("Status(%d) = %q, want %q", status, got, want)
		}
	}
}

func TestOnlyInformationalIsExcludedFromReadiness(t *testing.T) {
	for group, want := range map[Group]bool{
		GroupLiveness:      true,
		GroupCritical:      true,
		GroupInformational: false,
	} {
		if got := group.countsForReadiness(); got != want {
			t.Errorf("%s.countsForReadiness() = %v, want %v", group, got, want)
		}
	}
}
