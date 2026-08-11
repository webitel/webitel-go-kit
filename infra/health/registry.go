package health

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

// Registry runs checks in the background, one goroutine per check, and caches
// the last result. Every method is safe for concurrent use and on a nil
// *Registry. The zero value is not usable; use New.
type Registry struct {
	cfg Config
	log *slog.Logger

	// mu guards everything below, including every checkState, and is never
	// held while a check runs.
	mu        sync.Mutex
	checks    map[string]*checkState
	ctx       context.Context // nil until Start
	cancel    context.CancelFunc
	started   bool
	startedAt time.Time
	stopped   bool
	draining  bool
	drainAt   time.Time
	stopDone  chan struct{} // closed when the first Stop call finishes

	wg sync.WaitGroup
}

// New builds a registry. A nil logger discards.
func New(cfg Config, log *slog.Logger) *Registry {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}

	cfg = cfg.withDefaults()
	if cfg.Timeout >= cfg.Interval {
		log.Warn("health: Timeout is not smaller than Interval, checks cannot keep to schedule",
			"timeout", cfg.Timeout, "interval", cfg.Interval)
	}

	return &Registry{
		cfg:      cfg,
		log:      log,
		checks:   make(map[string]*checkState),
		stopDone: make(chan struct{}),
	}
}

// Liveness registers a check that answers "is this process wedged?".
func (r *Registry) Liveness(name string, fn Check) { r.register(name, GroupLiveness, fn) }

// Critical registers a check whose failure takes the node out of rotation.
func (r *Registry) Critical(name string, fn Check) { r.register(name, GroupCritical, fn) }

// Informational registers a check whose failure only degrades the node.
func (r *Registry) Informational(name string, fn Check) { r.register(name, GroupInformational, fn) }

// register adds a check, starting its goroutine at once when called after Start.
// Bad input is logged and dropped: registration has no error to return.
func (r *Registry) register(name string, group Group, fn Check) {
	if r == nil {
		return
	}
	if name == "" || fn == nil {
		r.log.Error("health: ignoring check with empty name or nil function", "check", name)

		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stopped {
		r.log.Error("health: ignoring check registered after Stop", "check", name)

		return
	}
	if _, exists := r.checks[name]; exists {
		r.log.Error("health: ignoring duplicate check name", "check", name)

		return
	}

	cs := &checkState{name: name, group: group, fn: fn}
	r.checks[name] = cs

	// Engine brings dependencies up one at a time, so this must work after Start.
	if r.started {
		r.wg.Add(1)
		go r.run(r.ctx, cs)
	}
}

// Start begins running every registered check. It does not block.
func (r *Registry) Start(ctx context.Context) error {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stopped {
		return errors.New("health: registry is stopped")
	}
	if r.started {
		return errors.New("health: registry is already started")
	}

	r.ctx, r.cancel = context.WithCancel(ctx)
	r.started = true
	r.startedAt = time.Now()

	for _, cs := range r.checks {
		r.wg.Add(1)
		go r.run(r.ctx, cs)
	}

	r.log.Info("health: started", "checks", len(r.checks), "interval", r.cfg.Interval)

	return nil
}

// Drain switches to not-ready, one way. It returns at once; the DrainHold wait
// happens in Stop.
func (r *Registry) Drain() {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.draining {
		return
	}

	r.draining = true
	r.drainAt = time.Now()
	r.log.Info("health: draining", "hold", r.cfg.DrainHold)
}

// Stop halts the scheduler, bounded by ctx. After Drain it first waits out the
// rest of DrainHold so service discovery sees us as not-ready.
func (r *Registry) Stop(ctx context.Context) error {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	if r.stopped {
		done := r.stopDone
		r.mu.Unlock()

		// Another Stop got there first; wait for it, bounded by our ctx.
		select {
		case <-done:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	r.stopped = true
	cancel, started, draining, drainAt := r.cancel, r.started, r.draining, r.drainAt
	r.mu.Unlock()

	defer close(r.stopDone)

	// The hold only matters if checks actually ran: a never-started registry
	// was never ready, so there is nothing for service discovery to notice.
	if draining && started {
		if err := wait(ctx, r.cfg.DrainHold-time.Since(drainAt)); err != nil {
			r.log.Warn("health: drain hold cut short", "err", err)
		}
	}

	if cancel != nil {
		cancel()
	}
	if !started {
		return nil
	}

	return r.join(ctx)
}

// results returns every check's state, sorted by name.
func (r *Registry) results() []CheckResult {
	return r.Snapshot().Checks
}

// run is one check's whole life: run, wait one Interval, repeat. The wait starts
// after the run, so a slow check is not re-run back to back as a Ticker would.
func (r *Registry) run(ctx context.Context, cs *checkState) {
	defer r.wg.Done()

	for {
		if ctx.Err() != nil {
			return
		}

		r.runOnce(ctx, cs)

		select {
		case <-ctx.Done():
			return
		case <-time.After(r.cfg.Interval):
		}
	}
}

func (r *Registry) runOnce(ctx context.Context, cs *checkState) {
	runCtx, cancel := context.WithTimeout(ctx, r.cfg.Timeout)
	defer cancel()

	err := r.call(runCtx, cs)

	if ctx.Err() != nil {
		return // shutting down, not the check's fault
	}
	if err == nil && errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		// Ignored its context and finished late. Late is not passing.
		err = fmt.Errorf("check %q passed only after its %s timeout", cs.name, r.cfg.Timeout)
	}

	r.record(cs, err)
}

// call runs the check; a panic becomes that check's error.
func (r *Registry) call(ctx context.Context, cs *checkState) (err error) {
	defer func() {
		if p := recover(); p != nil {
			r.log.Error("health: check panicked",
				"check", cs.name, "panic", p, "stack", string(debug.Stack()))
			err = fmt.Errorf("check %q panicked: %v", cs.name, p)
		}
	}()

	return cs.fn(ctx)
}

func (r *Registry) record(cs *checkState, err error) {
	r.mu.Lock()
	status, changed := cs.record(time.Now(), err, r.cfg)
	r.mu.Unlock()

	if !changed {
		return
	}

	if status == StatusFail {
		r.log.Warn("health: check is failing", "check", cs.name, "group", cs.group, "err", err)

		return
	}

	r.log.Info("health: check recovered", "check", cs.name, "group", cs.group, "status", status)
}

// join waits for the check goroutines, bounded by ctx.
func (r *Registry) join(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		r.log.Info("health: stopped")

		return nil
	case <-ctx.Done():
		return fmt.Errorf("health: stopped with checks still running: %w", ctx.Err())
	}
}

// wait sleeps for d, or until ctx is done.
func wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
