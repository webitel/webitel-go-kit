package health

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Snapshot is one consistent view of the registry, taken under a single lock.
type Snapshot struct {
	State          State         // verdict over liveness and critical; informational only degrades
	Checks         []CheckResult // sorted by name; Err is for logs, never for bodies
	SchedulerAlive bool          // the scheduler itself is still turning
	Draining       bool          // Drain was called; one-way
	Time           time.Time     // the single now used for staleness, grace, and this stamp
}

// Snapshot returns that view, taken at a single instant.
func (r *Registry) Snapshot() Snapshot {
	if r == nil {
		return Snapshot{State: StateUnknown, Time: time.Now()}
	}

	r.mu.Lock()
	now := time.Now()
	draining, stopped := r.draining, r.stopped
	started, startedAt := r.started, r.startedAt
	checks := make([]CheckResult, 0, len(r.checks))
	for _, cs := range r.checks {
		checks = append(checks, cs.result(now, r.cfg))
	}
	r.mu.Unlock()

	slices.SortFunc(checks, func(a, b CheckResult) int { return strings.Compare(a.Name, b.Name) })

	return Snapshot{
		State:          deriveState(checks, draining, stopped),
		Checks:         checks,
		SchedulerAlive: schedulerAlive(now, checks, started, startedAt, r.cfg.StaleAfter),
		Draining:       draining,
		Time:           now,
	}
}

// ReadyFunc returns the readiness verdict for service discovery. A false
// verdict always carries a non-nil error.
func (r *Registry) ReadyFunc() func() (bool, error) {
	return func() (bool, error) {
		s := r.Snapshot()
		if s.State.Ready() {
			return true, nil
		}

		return false, readyErr(s)
	}
}

// readyErr builds the verdict error from state tokens and check names only.
func readyErr(s Snapshot) error {
	if s.State == StateStopping {
		return errors.New("health: stopping: shutting down")
	}
	if len(s.Checks) == 0 {
		return errors.New("health: unknown: no checks registered")
	}

	var failing, unknown []string
	for _, c := range s.Checks {
		if !c.Group.countsForReadiness() {
			continue
		}
		if c.Status == StatusFail {
			failing = append(failing, c.Name)
		}
		if c.Status == StatusUnknown {
			unknown = append(unknown, c.Name)
		}
	}

	var parts []string
	if len(failing) > 0 {
		parts = append(parts, fmt.Sprintf("failing [%s]", strings.Join(failing, " ")))
	}
	if len(unknown) > 0 {
		parts = append(parts, fmt.Sprintf("unknown [%s]", strings.Join(unknown, " ")))
	}

	if len(parts) == 0 {
		return errors.New("health: unknown: no liveness or critical checks registered")
	}

	return fmt.Errorf("health: %s: %s", s.State, strings.Join(parts, "; "))
}

// deriveState is the readiness verdict, ordered and first-match-wins.
func deriveState(checks []CheckResult, draining, stopped bool) State {
	if draining || stopped {
		return StateStopping
	}
	if len(checks) == 0 {
		return StateUnknown
	}

	var counting, failing, unknown, degraded bool
	for _, c := range checks {
		if c.Group.countsForReadiness() {
			counting = true
			failing = failing || c.Status == StatusFail
			unknown = unknown || c.Status == StatusUnknown
		} else if c.Status != StatusOK {
			degraded = true
		}
	}

	// Nothing answers "can this node take traffic?", so no verdict is earned.
	if !counting {
		return StateUnknown
	}
	if failing {
		return StateNotReady
	}
	if unknown {
		return StateUnknown
	}
	if degraded {
		return StateDegraded
	}

	return StateReady
}

// schedulerAlive says whether the scheduler is still turning.
func schedulerAlive(now time.Time, checks []CheckResult, started bool, startedAt time.Time, staleAfter time.Duration) bool {
	if len(checks) == 0 {
		return true
	}
	if started && now.Sub(startedAt) <= staleAfter {
		return true
	}

	for _, c := range checks {
		if !c.LastRun.IsZero() && now.Sub(c.LastRun) <= staleAfter {
			return true
		}
	}

	return false
}
