package health

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"
)

const startAbortTimeout = 5 * time.Second

// Registry runs checks in the background and caches the last result. Every
// method is safe for concurrent use and on a nil *Registry. Use New.
type Registry struct {
	cfg        Config
	log        *slog.Logger
	transports []Transport
	// mu guards everything below and is never held while a check runs.
	mu        sync.Mutex
	checks    []*checkState   // sorted by name
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
func New(cfg Config, log *slog.Logger, opts ...Option) *Registry {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}

	cfg = cfg.withDefaults()
	if cfg.Timeout >= cfg.Interval {
		log.Warn("health: Timeout is not smaller than Interval, checks cannot keep to schedule",
			"timeout", cfg.Timeout, "interval", cfg.Interval)
	}

	r := &Registry{
		cfg:      cfg,
		log:      log,
		stopDone: make(chan struct{}),
	}
	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Liveness registers a check that answers "is this process wedged?".
func (r *Registry) Liveness(name string, fn Check) { r.register(name, GroupLiveness, fn) }

// Critical registers a check whose failure takes the node out of rotation.
func (r *Registry) Critical(name string, fn Check) { r.register(name, GroupCritical, fn) }

// Informational registers a check whose failure only degrades the node.
func (r *Registry) Informational(name string, fn Check) { r.register(name, GroupInformational, fn) }

// register adds a check, starting its goroutine at once when already running.
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
	at, exists := slices.BinarySearchFunc(r.checks, name, func(cs *checkState, name string) int {
		return strings.Compare(cs.name, name)
	})
	if exists {
		r.log.Error("health: ignoring duplicate check name", "check", name)

		return
	}

	cs := &checkState{name: name, group: group, fn: fn}
	r.checks = slices.Insert(r.checks, at, cs)

	if r.started {
		r.wg.Add(1)
		go r.run(r.ctx, cs)
	}
}

// Start begins running every registered check, then starts each transport in
// order. It does not block. When a transport fails to start, the transports
// already started are stopped and the checks are halted; the registry is then
// stopped for good and the caller must exit or build a new one. The returned
// error joins the start failure with any failure of that rollback.
func (r *Registry) Start(ctx context.Context) error {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return errors.New("health: registry is stopped")
	}
	if r.started {
		r.mu.Unlock()
		return errors.New("health: registry is already started")
	}

	r.ctx, r.cancel = context.WithCancel(ctx)
	r.started = true
	r.startedAt = time.Now()

	for _, cs := range r.checks {
		r.wg.Add(1)
		go r.run(r.ctx, cs)
	}

	runCtx, checks := r.ctx, len(r.checks)
	r.mu.Unlock()

	r.log.Info("health: started", "checks", checks, "interval", r.cfg.Interval)

	for i, t := range r.transports {
		if err := r.startTransport(runCtx, t); err != nil {
			r.log.Error("health: transport failed to start, rolling back", "err", err)

			return r.abortStart(ctx, i, err)
		}
	}

	return nil
}

func (r *Registry) startTransport(ctx context.Context, t Transport) error {
	r.mu.Lock()
	stopped := r.stopped
	r.mu.Unlock()

	if stopped {
		return errors.New("health: registry was stopped while starting")
	}

	return t.Start(ctx, r)
}

func (r *Registry) abortStart(ctx context.Context, started int, cause error) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), startAbortTimeout)
	defer cancel()

	return errors.Join(cause, r.stop(ctx, r.transports[:started]))
}

// Drain switches to not-ready, one way. The DrainHold wait happens in Stop.
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

// Shutdown drains, then stops the transports and the registry, in the order the
// registry requires: Drain first so the verdict flips before anything is torn
// down.
//
// ctx must carry at least DrainHold of budget: Drain returns immediately and
// the hold is absorbed inside Stop. A shorter context cuts the hold short and
// service discovery may never observe the node leaving rotation.
func (r *Registry) Shutdown(ctx context.Context) error {
	r.Drain()

	return r.Stop(ctx)
}

// Stop stops every transport in reverse start order, waits out any DrainHold,
// then halts the scheduler, all bounded by ctx. Every transport is stopped even
// if an earlier one fails; the errors are joined.
func (r *Registry) Stop(ctx context.Context) error {
	if r == nil {
		return nil
	}

	return r.stop(ctx, r.transports)
}

func (r *Registry) stop(ctx context.Context, up []Transport) error {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	if r.stopped {
		done := r.stopDone
		r.mu.Unlock()

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

	var errs []error
	for _, u := range slices.Backward(up) {
		if err := u.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	// A never-started registry was never ready, so there is nothing to hold for.
	if draining && started {
		if err := wait(ctx, r.cfg.DrainHold-time.Since(drainAt)); err != nil {
			r.log.Warn("health: drain hold cut short", "err", err)
		}
	}

	if cancel != nil {
		cancel()
	}
	if started {
		if err := r.join(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// results returns every check's state, sorted by name.
func (r *Registry) results() []CheckResult {
	return r.Snapshot().Checks
}

// run is one check's whole life. The wait starts after the run, so a slow check
// is not re-run back to back as a Ticker would.
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

	start := time.Now()
	err := r.call(runCtx, cs)
	elapsed := time.Since(start)

	if ctx.Err() != nil {
		return // shutting down, not the check's fault
	}
	if err == nil && errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		// Finished late: that is not passing.
		err = fmt.Errorf("check %q passed only after its %s timeout", cs.name, r.cfg.Timeout)
	}

	r.record(cs, elapsed, err)
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

func (r *Registry) record(cs *checkState, elapsed time.Duration, err error) {
	r.mu.Lock()
	status, changed := cs.record(time.Now(), elapsed, err, r.cfg)
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
