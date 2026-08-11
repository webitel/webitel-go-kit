package sdnotify

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/webitel/webitel-go-kit/infra/health"
)

// Notifier reports the registry's state to systemd. A nil *Notifier is a no-op.
type Notifier struct {
	reg  *health.Registry
	addr string
	log  *slog.Logger

	writeTimeout time.Duration
	poll         time.Duration
	wdPeriod     time.Duration // WATCHDOG_USEC/2, 0 when disabled
	startTimeout time.Duration

	cancel context.CancelFunc
	done   chan struct{}

	startOnce sync.Once
	stopOnce  sync.Once
	stopErr   error // written under stopOnce, so every caller sees the same verdict
}

// loopState lives on the loop goroutine's stack, so it needs no mutex.
type loopState struct {
	readySent  bool
	haveStatus bool
	lastState  health.State
}

// New builds a Notifier, or nil when NOTIFY_SOCKET is unset. The environment is
// read once, here.
func New(r *health.Registry, opts ...Option) *Notifier {
	addr := os.Getenv("NOTIFY_SOCKET")
	if addr == "" {
		return nil
	}

	o := newOptions(opts)

	// A malformed watchdog environment disables the ping and nothing else.
	interval, err := watchdogEnabled()
	if err != nil {
		o.log.Warn("health/sdnotify: the watchdog environment is unusable, pings are disabled", "err", err)
	}

	return &Notifier{
		reg:          r,
		addr:         addr,
		log:          o.log,
		writeTimeout: o.writeTimeout,
		poll:         o.poll,
		wdPeriod:     interval / 2,
		startTimeout: o.startTimeout,
	}
}

// Start begins notifying. It does not block and a second call is a no-op. The
// error is always nil — sd_notify(3) says to ignore notify failures — and the
// signature keeps symmetry with health.Registry.Start.
func (n *Notifier) Start(ctx context.Context) error {
	if n == nil {
		return nil
	}

	first := false
	n.startOnce.Do(func() {
		first = true
		ctx, n.cancel = context.WithCancel(ctx)
		n.done = make(chan struct{})

		go n.loop(ctx)
	})

	if !first {
		n.log.Debug("health/sdnotify: already started")
	}

	return nil
}

// Stop sends STOPPING=1 and halts the loop. Only the loop wait is bounded by
// ctx; the STOPPING=1 write happens first regardless, bounded by
// WithWriteTimeout — budget it on top of ctx. A second call repeats the first
// verdict. It does not Drain: the consumer Drains first, then Stops.
func (n *Notifier) Stop(ctx context.Context) error {
	if n == nil {
		return nil
	}

	n.stopOnce.Do(func() {
		// Burning the start once makes a Start after Stop a no-op, and orders a
		// concurrent Start against this Stop: sync.Once supplies the
		// happens-before edge that lets cancel and done be read without a mutex.
		n.startOnce.Do(func() {})

		// Before the cancel, so a signal handler with a dead ctx still notifies.
		// A poll tick racing this can emit one more STATUS=, byte-identical to
		// this one, because the consumer has already Drained.
		n.send(notifyStopping + "\nSTATUS=" + health.NameStopping)

		if n.cancel == nil {
			return // Start was never called
		}

		n.cancel()
		n.stopErr = n.join(ctx)
	})

	return n.stopErr
}

// loop is the only goroutine; each of its three timers is a distinct trigger.
func (n *Notifier) loop(ctx context.Context) {
	defer close(n.done)

	var ls loopState

	poll := time.NewTicker(n.poll)
	defer poll.Stop()

	// A nil channel blocks forever in select, which is exactly "disabled".
	var wdC <-chan time.Time
	if n.wdPeriod > 0 {
		wd := time.NewTicker(n.wdPeriod)
		defer wd.Stop()
		wdC = wd.C
	}

	var startC <-chan time.Time
	if n.startTimeout > 0 {
		st := time.NewTimer(n.startTimeout)
		defer st.Stop()
		startC = st.C
	}

	// Announcing readiness on an already-dead context is a lie to systemd.
	if ctx.Err() != nil {
		return
	}

	n.sync(&ls, n.reg.Snapshot())

	for {
		select {
		case <-ctx.Done():
			return
		case <-poll.C:
			n.sync(&ls, n.reg.Snapshot())
		case <-wdC:
			n.ping()
		case <-startC:
			n.forceReady(&ls)
		}
	}
}

// sync emits READY=1 once and STATUS= on a state change, from the one snapshot
// it is given — a second Snapshot() could straddle a change and pair the two
// from different instants. State.Ready() is the trigger, as in ReadyFunc.
func (n *Notifier) sync(ls *loopState, s health.Snapshot) {
	text := statusText(s)

	if !ls.readySent && s.State.Ready() {
		n.send(notifyReady + "\nSTATUS=" + text)
		ls.readySent, ls.haveStatus, ls.lastState = true, true, s.State

		return
	}

	// haveStatus, not a zero lastState: State(0) is the real value StateUnknown.
	if !ls.haveStatus || s.State != ls.lastState {
		n.send("STATUS=" + text)
		ls.haveStatus, ls.lastState = true, s.State
	}
}

// ping proves the process is not wedged. SchedulerAlive is the sole gate, so a
// dependency outage degrades readiness without restarting the whole fleet.
func (n *Notifier) ping() {
	if !n.reg.Snapshot().SchedulerAlive {
		n.log.Debug("health/sdnotify: skipping the watchdog ping, the scheduler is not turning")

		return
	}

	n.send(notifyWatchdog)
}

// forceReady is the WithStartTimeout fallback: better to come up visibly degraded
// than hang in activating. Clearing haveStatus makes the next poll restate the truth.
func (n *Notifier) forceReady(ls *loopState) {
	if ls.readySent {
		return
	}

	n.send(notifyReady + "\nSTATUS=" + statusStarting)
	ls.readySent, ls.haveStatus = true, false
}

// send writes one datagram, dialling afresh. Failures are logged and swallowed:
// the next tick retries, and a genuinely broken socket means systemd kills us
// anyway, which is the correct outcome.
func (n *Notifier) send(state string) {
	if err := notify(n.addr, state, n.writeTimeout); err != nil {
		n.log.Warn("health/sdnotify: sending a notification failed", "err", err, "state", state)
	}
}

// join waits for the loop goroutine, bounded by ctx.
func (n *Notifier) join(ctx context.Context) error {
	// A non-blocking first look: with both ready, select picks at random, and an
	// already-exited loop is never a failure.
	select {
	case <-n.done:
		n.log.Info("health/sdnotify: stopped")

		return nil
	default:
	}

	select {
	case <-n.done:
		n.log.Info("health/sdnotify: stopped")

		return nil
	case <-ctx.Done():
		return fmt.Errorf("health/sdnotify: stopped with the loop still running: %w", ctx.Err())
	}
}
