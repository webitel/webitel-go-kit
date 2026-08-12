package health

import (
	"context"
	"fmt"
	"time"
)

// Names used in logs and in the /healthz body.
const (
	NameUnknown       = "unknown"
	NameOK            = "ok"
	NameFail          = "fail"
	NameLiveness      = "liveness"
	NameCritical      = "critical"
	NameInformational = "informational"
	NameNotReady      = "not_ready"
	NameDegraded      = "degraded"
	NameReady         = "ready"
	NameStopping      = "stopping"
)

// Check is a health check. It must respect ctx.
type Check func(context.Context) error

// Group is what a check was registered as.
type Group uint8

const (
	GroupLiveness      Group = iota // "is this process wedged?"; counts towards readiness
	GroupCritical                   // node-local failure: takes the node out of rotation
	GroupInformational              // only degrades; the node stays in rotation
)

var groupNames = [...]string{NameLiveness, NameCritical, NameInformational}

func (g Group) String() string {
	if int(g) >= len(groupNames) {
		return NameUnknown
	}

	return groupNames[g]
}

func (g Group) countsForReadiness() bool {
	return g != GroupInformational
}

// Status is a check's state once hysteresis and staleness are applied.
type Status uint8

const (
	StatusUnknown Status = iota // never ran, or older than Config.StaleAfter; never passes
	StatusOK
	StatusFail
)

var statusNames = [...]string{NameUnknown, NameOK, NameFail}

func (s Status) String() string {
	if int(s) >= len(statusNames) {
		return NameUnknown
	}

	return statusNames[s]
}

// State is the registry's verdict over every check, as carried by a Snapshot.
type State uint8

const (
	StateUnknown  State = iota // no checks, or a counting check without a usable result
	StateNotReady              // a liveness or critical check is failing
	StateDegraded              // only informational checks are not OK; stays in rotation
	StateReady                 // every check is OK
	StateStopping              // Drain or Stop was called; one-way
)

var stateNames = [...]string{NameUnknown, NameNotReady, NameDegraded, NameReady, NameStopping}

func (s State) String() string {
	if int(s) >= len(stateNames) {
		return NameUnknown
	}

	return stateNames[s]
}

// Ready reports whether the state serves traffic; degraded still counts.
func (s State) Ready() bool {
	return s == StateReady || s == StateDegraded
}

// CheckResult is one check's state when a snapshot was taken. Err is non-nil
// whenever Status is not StatusOK, and is for logs only.
type CheckResult struct {
	Name    string
	Group   Group
	Status  Status
	Since   time.Time // when Status last changed
	LastRun time.Time // when the last run completed
	Err     error
}

// checkState is guarded by the registry's mutex.
type checkState struct {
	name  string
	group Group
	fn    Check

	status  Status
	since   time.Time
	lastRun time.Time
	lastErr error
	fails   int
	ran     bool
}

// record folds one run's outcome in, applying hysteresis.
func (cs *checkState) record(now time.Time, err error, cfg Config) (Status, bool) {
	was := cs.status
	cs.ran = true
	cs.lastRun = now

	if err != nil {
		cs.lastErr = err
		cs.fails++
		if cs.fails >= cfg.FailThreshold {
			cs.set(StatusFail, now)
		}
	} else {
		cs.fails = 0
		if cs.status != StatusFail || now.Sub(cs.since) >= cfg.MinUnready {
			cs.set(StatusOK, now)
			cs.lastErr = nil
		}
	}

	return cs.status, cs.status != was
}

func (cs *checkState) set(status Status, now time.Time) {
	if cs.status == status {
		return
	}

	cs.status = status
	cs.since = now
}

// result reads the check's state, applying staleness at read time.
func (cs *checkState) result(now time.Time, cfg Config) CheckResult {
	res := CheckResult{
		Name:    cs.name,
		Group:   cs.group,
		Status:  cs.status,
		Since:   cs.since,
		LastRun: cs.lastRun,
		Err:     cs.lastErr,
	}

	switch {
	case !cs.ran:
		res.Status = StatusUnknown
		res.Err = fmt.Errorf("check %q has not completed a run yet", cs.name)
	case now.Sub(cs.lastRun) > cfg.StaleAfter:
		res.Status = StatusUnknown
		res.Err = fmt.Errorf("check %q is stale: last run %s ago",
			cs.name, now.Sub(cs.lastRun).Round(time.Millisecond))
	}

	return res
}
