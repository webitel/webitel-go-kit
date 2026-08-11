package sdnotify

import (
	"context"
	"errors"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/webitel/webitel-go-kit/infra/health"
)

func TestNewWithoutSocket(t *testing.T) {
	// Explicit rather than ambient: a CI runner that set NOTIFY_SOCKET would
	// otherwise break this silently.
	t.Setenv("NOTIFY_SOCKET", "")

	n := New(health.New(fastConfig(), nil))
	if n != nil {
		t.Fatal("New returned a Notifier with NOTIFY_SOCKET unset")
	}

	ctx := context.Background()
	if err := n.Start(ctx); err != nil {
		t.Fatalf("Start on a nil Notifier: %v", err)
	}
	if err := n.Stop(ctx); err != nil {
		t.Fatalf("Stop on a nil Notifier: %v", err)
	}
}

func TestReadyAndStopEndToEnd(t *testing.T) {
	addr, ln := notifySocket(t)

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", passing)
	waitState(t, reg, health.StateReady)

	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond))

	ctx := context.Background()
	if err := n.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if got, want := recv(t, ln, time.Second), notifyReady+"\nSTATUS="+health.NameReady; got != want {
		t.Fatalf("ready datagram = %q, want %q", got, want)
	}

	if err := n.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if got, want := recv(t, ln, time.Second), notifyStopping+"\nSTATUS="+health.NameStopping; got != want {
		t.Fatalf("stop datagram = %q, want %q", got, want)
	}
}

func TestReadySentOnceOnReady(t *testing.T) {
	addr, ln := notifySocket(t)

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", passing)
	waitState(t, reg, health.StateReady)

	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	got := recvAll(t, ln, 50*time.Millisecond, 2*time.Second)

	ready := 0
	for _, d := range got {
		if strings.Contains(d, notifyReady) {
			ready++
		}
	}
	if ready != 1 {
		t.Fatalf("READY=1 datagrams = %d, want 1; got %q", ready, got)
	}
}

func TestReadyOnDegraded(t *testing.T) {
	// Degraded means every critical check is green, which is literally the
	// design's own READY=1 trigger. ReadyFunc and /readyz already agree.
	addr, ln := notifySocket(t)

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", passing)
	reg.Informational("s3", failing("s3"))
	waitState(t, reg, health.StateDegraded)

	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	want := notifyReady + "\nSTATUS=" + health.NameDegraded + ": informational [s3]"
	if got := recv(t, ln, time.Second); got != want {
		t.Fatalf("datagram = %q, want %q", got, want)
	}
}

func TestNoReadyBeforeFirstRound(t *testing.T) {
	// A check that has not finished its first run: the cold start, where every
	// counting check is unknown and there is no verdict to report yet.
	addr, ln := notifySocket(t)

	cfg := fastConfig()
	cfg.Interval = time.Second
	cfg.Timeout = 900 * time.Millisecond

	reg := newTestRegistry(t, cfg)
	reg.Critical("db", func(ctx context.Context) error {
		<-ctx.Done()

		return ctx.Err()
	})

	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	got := recvAll(t, ln, 50*time.Millisecond, 2*time.Second)
	if len(got) == 0 {
		t.Fatal("no STATUS= datagram at all")
	}
	for _, d := range got {
		if strings.Contains(d, notifyReady) {
			t.Fatalf("READY=1 before the first round: %q", got)
		}
	}
}

func TestStatusOnChange(t *testing.T) {
	addr, ln := notifySocket(t)

	tg := newToggle(errors.New("db is down"))

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", tg.check)
	waitState(t, reg, health.StateReady)

	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if got, want := recv(t, ln, time.Second), notifyReady+"\nSTATUS="+health.NameReady; got != want {
		t.Fatalf("ready datagram = %q, want %q", got, want)
	}
	if quiet := recvAll(t, ln, 50*time.Millisecond, 2*time.Second); len(quiet) != 0 {
		t.Fatalf("steady state sent %q, want nothing", quiet)
	}

	tg.fail()
	waitState(t, reg, health.StateNotReady)

	want := "STATUS=" + health.NameNotReady + ": failing [db]"
	if got := recv(t, ln, time.Second); got != want {
		t.Fatalf("change datagram = %q, want %q", got, want)
	}
	if quiet := recvAll(t, ln, 50*time.Millisecond, 2*time.Second); len(quiet) != 0 {
		t.Fatalf("steady state sent %q, want nothing", quiet)
	}
}

func TestRedialAfterRebind(t *testing.T) {
	// The executable form of "no conn field on Notifier": a cached unixgram
	// conn is permanently dead once the receiver re-binds, while a fresh dial
	// per message recovers with no reconnect code at all.
	addr, ln := notifySocket(t)

	tg := newToggle(errors.New("db is down"))

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", tg.check)
	waitState(t, reg, health.StateReady)

	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	recv(t, ln, time.Second) // READY=1

	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	ln2 := rebind(t, addr)

	tg.fail()
	waitState(t, reg, health.StateNotReady)

	want := "STATUS=" + health.NameNotReady + ": failing [db]"
	if got := recv(t, ln2, 2*time.Second); got != want {
		t.Fatalf("datagram after the rebind = %q, want %q", got, want)
	}
}

func TestWriteDeadline(t *testing.T) {
	// This exercises the deadline plumbing, not real backpressure: on darwin an
	// undrained unixgram receiver returns ENOBUFS immediately rather than
	// blocking, so a backpressure test would pass on Linux and fail here.
	addr, _ := notifySocket(t)

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", passing)
	waitState(t, reg, health.StateReady)

	lg := &logCapture{}
	n := newNotifier(t, reg, addr,
		WithPollInterval(10*time.Millisecond),
		WithWriteTimeout(time.Nanosecond),
		WithLogger(lg.logger()))

	ctx := context.Background()
	if err := n.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	waitFor(t, "a deadline error to be logged", func() bool {
		for _, err := range lg.errs() {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				return true
			}
		}

		return false
	})

	// A notify failure is never fatal.
	if err := n.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestLifecycleEdges(t *testing.T) {
	addr, ln := notifySocket(t)

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", passing)
	waitState(t, reg, health.StateReady)

	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond))

	ctx := context.Background()

	// Stop before Start still tells systemd we are going away.
	if err := n.Stop(ctx); err != nil {
		t.Fatalf("Stop before Start: %v", err)
	}
	if got, want := recv(t, ln, time.Second), notifyStopping+"\nSTATUS="+health.NameStopping; got != want {
		t.Fatalf("stop datagram = %q, want %q", got, want)
	}

	// The burnt startOnce makes every later Start a no-op.
	if err := n.Start(ctx); err != nil {
		t.Fatalf("Start after Stop: %v", err)
	}
	if err := n.Start(ctx); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	if err := n.Stop(ctx); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
	if got := recvAll(t, ln, 50*time.Millisecond, 2*time.Second); len(got) != 0 {
		t.Fatalf("a no-op Start sent %q, want nothing", got)
	}

	// Registry.Snapshot is nil-safe, so a nil registry must not panic.
	t.Setenv("NOTIFY_SOCKET", addr)

	empty := New(nil, WithPollInterval(10*time.Millisecond))
	if empty == nil {
		t.Fatal("New returned nil with NOTIFY_SOCKET set")
	}
	if err := empty.Start(ctx); err != nil {
		t.Fatalf("Start with a nil registry: %v", err)
	}
	if err := empty.Stop(ctx); err != nil {
		t.Fatalf("Stop with a nil registry: %v", err)
	}
}

func TestAbstractSocket(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("abstract unix sockets are Linux-only; on darwin '@' is an ordinary filename")
	}

	// This is the one socket test that does not go through notifySocket, so it
	// neutralises the ambient watchdog environment itself.
	t.Setenv("WATCHDOG_USEC", "")
	t.Setenv("WATCHDOG_PID", "")

	// A bare PID is not enough: -count=3 runs three iterations in one process.
	name := "@webitel-health-" + strconv.Itoa(os.Getpid()) + "-" +
		strconv.FormatUint(socketNonce.Add(1), 10)

	ln, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: name, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", passing)
	waitState(t, reg, health.StateReady)

	n := newNotifier(t, reg, name, WithPollInterval(10*time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if got, want := recv(t, ln, time.Second), notifyReady+"\nSTATUS="+health.NameReady; got != want {
		t.Fatalf("datagram = %q, want %q", got, want)
	}
}

func TestStartTimeoutFallback(t *testing.T) {
	addr, ln := notifySocket(t)

	tg := newToggle(errors.New("db is down"))
	tg.fail()

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", tg.check)
	waitState(t, reg, health.StateNotReady)

	n := newNotifier(t, reg, addr,
		WithPollInterval(10*time.Millisecond),
		WithStartTimeout(20*time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	fallback := notifyReady + "\nSTATUS=" + statusStarting
	got := recvAll(t, ln, 60*time.Millisecond, 2*time.Second)
	if c := countEqual(got, fallback); c != 1 {
		t.Fatalf("fallback datagrams = %d, want 1; got %q", c, got)
	}

	// Becoming genuinely ready afterwards reports STATUS=ready with no second
	// READY=1.
	tg.pass()
	waitState(t, reg, health.StateReady)

	rest := recvAll(t, ln, 60*time.Millisecond, 2*time.Second)
	for _, d := range rest {
		if strings.Contains(d, notifyReady) {
			t.Fatalf("a second READY=1 after the fallback: %q", rest)
		}
	}
	if want := "STATUS=" + health.NameReady; countEqual(rest, want) != 1 {
		t.Fatalf("datagrams = %q, want exactly one %q", rest, want)
	}
}

func TestStartTimeoutFallbackRestatesTruth(t *testing.T) {
	// The wedged node the fallback exists for: the state never changes again,
	// so "starting degraded" must not be the last thing systemd ever hears.
	addr, ln := notifySocket(t)

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", failing("db"))
	waitState(t, reg, health.StateNotReady)

	n := newNotifier(t, reg, addr,
		WithPollInterval(10*time.Millisecond),
		WithStartTimeout(30*time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	got := recvAll(t, ln, 200*time.Millisecond, 3*time.Second)
	want := "STATUS=" + health.NameNotReady + ": failing [db]"
	if last := got[len(got)-1]; last != want {
		t.Fatalf("last datagram = %q, want %q; got %q", last, want, got)
	}
}

func TestStartOnADeadContextSendsNothing(t *testing.T) {
	addr, ln := notifySocket(t)

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", passing)
	waitState(t, reg, health.StateReady)

	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := n.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if got := recvAll(t, ln, 60*time.Millisecond, 2*time.Second); len(got) != 0 {
		t.Fatalf("Start on a dead ctx announced %q, want nothing", got)
	}
}

func TestStopAfterTheLoopExitedIsNotAnError(t *testing.T) {
	// Both done and ctx.Done are ready here, and select picks at random: a loop
	// that has provably already exited must never be reported as still running.
	addr, _ := notifySocket(t)

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", passing)

	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	if err := n.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	cancel()
	<-n.done

	if err := n.Stop(ctx); err != nil {
		t.Fatalf("Stop after the loop exited: %v", err)
	}
}

func TestStopRepeatsTheFirstVerdict(t *testing.T) {
	// A second caller blocks inside stopOnce and must not report success when
	// the first caller left the loop running.
	addr, _ := notifySocket(t)

	t.Setenv("NOTIFY_SOCKET", addr)

	n := New(nil)
	if n == nil {
		t.Fatal("New returned nil with NOTIFY_SOCKET set")
	}

	// Stand in for a loop that will not exit: done never closes, so join can
	// only report the ctx deadline.
	n.cancel, n.done = func() {}, make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	first := n.Stop(ctx)
	if first == nil {
		t.Fatal("Stop with a live loop and a dead ctx returned nil")
	}
	if second := n.Stop(ctx); second == nil {
		t.Fatalf("a second Stop returned nil, want %v", first)
	}
}

func TestStartTimeoutDisabled(t *testing.T) {
	// The fallback really does ship off: without the option there is none.
	addr, ln := notifySocket(t)

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", failing("db"))
	waitState(t, reg, health.StateNotReady)

	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	got := recvAll(t, ln, 60*time.Millisecond, 2*time.Second)
	if len(got) == 0 {
		t.Fatal("no STATUS= datagram at all")
	}
	for _, d := range got {
		if strings.Contains(d, notifyReady) {
			t.Fatalf("READY=1 with WithStartTimeout unset: %q", got)
		}
	}
}
