package sdnotify

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/webitel/webitel-go-kit/infra/health"
)

func TestWatchdogPings(t *testing.T) {
	addr, ln := notifySocket(t)

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", passing)
	waitState(t, reg, health.StateReady)

	t.Setenv("WATCHDOG_USEC", "20000") // period 10ms
	t.Setenv("WATCHDOG_PID", "")

	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	pings := 0
	for range 20 {
		d := recv(t, ln, time.Second)
		if !strings.Contains(d, notifyWatchdog) {
			continue
		}
		if d != notifyWatchdog {
			t.Fatalf("a watchdog ping carries more than WATCHDOG=1: %q", d)
		}

		pings++
		if pings == 2 {
			return
		}
	}

	t.Fatalf("watchdog pings = %d, want at least 2", pings)
}

func TestWatchdogGatedOnSchedulerAlive(t *testing.T) {
	// D3: the ping tracks the registry's own scheduler, never a dependency, so
	// one database outage cannot restart the whole fleet at once.
	addr, ln := notifySocket(t)

	cfg := fastConfig()
	cfg.StaleAfter = 50 * time.Millisecond

	reg := newTestRegistry(t, cfg)
	reg.Critical("db", passing)
	waitState(t, reg, health.StateReady)

	t.Setenv("WATCHDOG_USEC", "20000")
	t.Setenv("WATCHDOG_PID", "")

	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := reg.Stop(ctx); err != nil {
		t.Fatalf("registry Stop: %v", err)
	}

	waitFor(t, "the scheduler to go stale", func() bool { return !reg.Snapshot().SchedulerAlive })

	recvAll(t, ln, 60*time.Millisecond, 2*time.Second) // drain what was in flight

	got := recvAll(t, ln, 60*time.Millisecond, 2*time.Second)
	for _, d := range got {
		if strings.Contains(d, notifyWatchdog) {
			t.Fatalf("a ping after the scheduler went stale: %q", got)
		}
	}
}

func TestWatchdogPIDMismatch(t *testing.T) {
	addr, ln := notifySocket(t)

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", passing)
	waitState(t, reg, health.StateReady)

	t.Setenv("WATCHDOG_USEC", "20000")
	t.Setenv("WATCHDOG_PID", strconv.Itoa(os.Getpid()+1))

	lg := &logCapture{}
	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond), WithLogger(lg.logger()))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	got := recvAll(t, ln, 60*time.Millisecond, 2*time.Second)
	for _, d := range got {
		if strings.Contains(d, notifyWatchdog) {
			t.Fatalf("a ping with a mismatched WATCHDOG_PID: %q", got)
		}
	}

	// A mismatch is silently disabled, not an error.
	if errs := lg.errs(); len(errs) != 0 {
		t.Fatalf("a PID mismatch logged errors: %v", errs)
	}
}

func TestWatchdogMalformedEnv(t *testing.T) {
	addr, ln := notifySocket(t)

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", passing)
	waitState(t, reg, health.StateReady)

	t.Setenv("WATCHDOG_USEC", "not-a-number")
	t.Setenv("WATCHDOG_PID", "")

	lg := &logCapture{}
	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond), WithLogger(lg.logger()))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if !strings.Contains(lg.text(), "WATCHDOG_USEC") {
		t.Fatalf("the unusable WATCHDOG_USEC was not logged: %s", lg.text())
	}

	// A malformed environment disables the ping and nothing else.
	got := recvAll(t, ln, 60*time.Millisecond, 2*time.Second)
	if want := notifyReady + "\nSTATUS=" + health.NameReady; countEqual(got, want) != 1 {
		t.Fatalf("datagrams = %q, want exactly one %q", got, want)
	}
	for _, d := range got {
		if strings.Contains(d, notifyWatchdog) {
			t.Fatalf("a ping with an unusable WATCHDOG_USEC: %q", got)
		}
	}
}

func TestWatchdogUsecOutOfRange(t *testing.T) {
	// WATCHDOG_USEC comes from outside this module. Unbounded, the conversion
	// to a Duration wraps: 18446744073709552 used to yield a 192ns period, a
	// spin loop dialling the notify socket millions of times a second.
	t.Setenv("WATCHDOG_PID", "")

	for _, v := range []string{"0", "-1", "9223372036854775807", "18446744073709552"} {
		t.Setenv("WATCHDOG_USEC", v)

		d, err := watchdogEnabled()
		if err == nil || d != 0 {
			t.Errorf("WATCHDOG_USEC=%s gave (%s, %v), want (0, an error)", v, d, err)
		}
	}
}

func TestWatchdogAbsent(t *testing.T) {
	addr, ln := notifySocket(t)

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", passing)
	waitState(t, reg, health.StateReady)

	t.Setenv("WATCHDOG_USEC", "")
	t.Setenv("WATCHDOG_PID", "")

	lg := &logCapture{}
	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond), WithLogger(lg.logger()))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	got := recvAll(t, ln, 60*time.Millisecond, 2*time.Second)
	for _, d := range got {
		if strings.Contains(d, notifyWatchdog) {
			t.Fatalf("a ping with no WATCHDOG_USEC: %q", got)
		}
	}
	if errs := lg.errs(); len(errs) != 0 {
		t.Fatalf("an absent watchdog logged errors: %v", errs)
	}
}
