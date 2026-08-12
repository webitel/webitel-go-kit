package sdnotify

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/webitel/webitel-go-kit/infra/health"
)

// sentinel is the check error text that must never reach a datagram.
const sentinel = "SENTINEL-9f2a41-must-not-leak"

func check(name string, group health.Group, status health.Status) health.CheckResult {
	return health.CheckResult{Name: name, Group: group, Status: status}
}

func TestStatusTextTable(t *testing.T) {
	cases := []struct {
		name string
		snap health.Snapshot
		want string
	}{
		{
			name: "ready",
			snap: health.Snapshot{State: health.StateReady, Checks: []health.CheckResult{
				check("db", health.GroupCritical, health.StatusOK),
			}},
			want: "ready",
		},
		{
			name: "degraded",
			snap: health.Snapshot{State: health.StateDegraded, Checks: []health.CheckResult{
				check("db", health.GroupCritical, health.StatusOK),
				check("s3", health.GroupInformational, health.StatusFail),
			}},
			want: "degraded: informational [s3]",
		},
		{
			name: "not ready",
			snap: health.Snapshot{State: health.StateNotReady, Checks: []health.CheckResult{
				check("db", health.GroupCritical, health.StatusFail),
			}},
			want: "not_ready: failing [db]",
		},
		{
			name: "not ready and informational",
			snap: health.Snapshot{State: health.StateNotReady, Checks: []health.CheckResult{
				check("db", health.GroupCritical, health.StatusFail),
				check("s3", health.GroupInformational, health.StatusFail),
			}},
			want: "not_ready: failing [db]; informational [s3]",
		},
		{
			name: "no checks",
			snap: health.Snapshot{State: health.StateUnknown},
			want: "unknown: no checks registered",
		},
		{
			name: "pending",
			snap: health.Snapshot{State: health.StateUnknown, Checks: []health.CheckResult{
				check("db", health.GroupCritical, health.StatusUnknown),
			}},
			want: "unknown: pending [db]",
		},
		{
			name: "informational pending",
			snap: health.Snapshot{State: health.StateDegraded, Checks: []health.CheckResult{
				check("db", health.GroupCritical, health.StatusOK),
				check("s3", health.GroupInformational, health.StatusUnknown),
			}},
			want: "degraded: pending [s3]",
		},
		{
			name: "stopping",
			snap: health.Snapshot{State: health.StateStopping, Checks: []health.CheckResult{
				check("db", health.GroupCritical, health.StatusFail),
			}},
			want: "stopping",
		},
	}

	for _, c := range cases {
		if got := statusText(c.snap); got != c.want {
			t.Errorf("%s: statusText = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestStatusTextSanitisesNames(t *testing.T) {
	s := health.Snapshot{State: health.StateNotReady, Checks: []health.CheckResult{
		check("bad\nREADY=1\tx", health.GroupCritical, health.StatusFail),
	}}

	got := statusText(s)
	if want := "not_ready: failing [bad_READY=1_x]"; got != want {
		t.Fatalf("statusText = %q, want %q", got, want)
	}
	if strings.Count(got, "\n") != 0 {
		t.Fatalf("statusText = %q, want a single line", got)
	}
}

func TestStatusTextIsCapped(t *testing.T) {
	// systemd drops an oversized datagram whole, taking the READY=1 riding with
	// it. The names are multi-byte on purpose: the cut lands mid-rune here, so
	// this also exercises the rune-boundary walk-back.
	s := health.Snapshot{State: health.StateNotReady}
	for range 400 {
		s.Checks = append(s.Checks, check(strings.Repeat("日", 20), health.GroupCritical, health.StatusFail))
	}

	got := statusText(s)
	if len(got) > maxStatusBytes {
		t.Fatalf("statusText is %d bytes, want at most %d", len(got), maxStatusBytes)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("a capped line does not end in an ellipsis: %q", got)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("the cap cut a rune in half: %q", got)
	}
}

func TestStatusIsOneLineOnTheWire(t *testing.T) {
	// The end-to-end form: a hostile check name must not become a second
	// systemd assignment in the datagram.
	addr, ln := notifySocket(t)

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("bad\nREADY=1\nWATCHDOG=1", failing("bad"))
	waitState(t, reg, health.StateNotReady)

	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// The registry is not_ready, so this datagram is STATUS= alone: exactly one
	// line. A looser bound would let a sanitiser regression inject MAINPID= or
	// ERRNO= and still pass.
	if got := recv(t, ln, time.Second); strings.Count(got, "\n") != 0 {
		t.Fatalf("datagram %q contains %d newlines, want 0", got, strings.Count(got, "\n"))
	}
}

func TestNoErrorTextLeaks(t *testing.T) {
	// D1/D8: the property this transport must never break. Every state is
	// driven, and the vacuity guard below is what makes the result mean
	// something — a healthy registry would pass this test trivially.
	addr, ln := notifySocket(t)

	lg := &logCapture{}

	reg := health.New(fastConfig(), lg.logger())
	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("registry Start: %v", err)
	}
	stopAtCleanup(t, reg)

	n := newNotifier(t, reg, addr, WithPollInterval(10*time.Millisecond))
	if err := n.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// unknown: an empty registry has no verdict.
	seen := recvAll(t, ln, 60*time.Millisecond, 2*time.Second)

	live, db, s3 := newToggle(errors.New(sentinel)), newToggle(errors.New(sentinel)), newToggle(errors.New(sentinel))
	for _, tg := range []*toggle{live, db, s3} {
		tg.fail()
	}

	reg.Liveness("live", live.check)
	reg.Critical("db", db.check)
	reg.Informational("s3", s3.check)
	waitState(t, reg, health.StateNotReady)

	// The vacuity guard. deriveState is first-match-wins, so the state proves
	// only that *one* counting check is down — wait on the property actually
	// being asserted instead, or a check still in its first run reads unknown.
	waitFor(t, "all three checks to be failing", func() bool {
		cs := reg.Snapshot().Checks
		if len(cs) != 3 {
			return false
		}
		for _, c := range cs {
			if c.Status != health.StatusFail {
				return false
			}
		}

		return true
	})

	if !strings.Contains(lg.text(), sentinel) {
		t.Fatal("the sentinel is not in the registry log either, so this test proves nothing")
	}

	seen = append(seen, recvAll(t, ln, 60*time.Millisecond, 2*time.Second)...)

	// degraded: only the informational check is still red.
	live.pass()
	db.pass()
	waitState(t, reg, health.StateDegraded)
	seen = append(seen, recvAll(t, ln, 60*time.Millisecond, 2*time.Second)...)

	s3.pass()
	waitState(t, reg, health.StateReady)
	seen = append(seen, recvAll(t, ln, 60*time.Millisecond, 2*time.Second)...)

	db.fail()
	waitState(t, reg, health.StateNotReady)
	seen = append(seen, recvAll(t, ln, 60*time.Millisecond, 2*time.Second)...)

	reg.Drain()
	waitState(t, reg, health.StateStopping)
	seen = append(seen, recvAll(t, ln, 60*time.Millisecond, 2*time.Second)...)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := n.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	seen = append(seen, recvAll(t, ln, 60*time.Millisecond, 2*time.Second)...)

	if len(seen) == 0 {
		t.Fatal("no datagrams captured at all")
	}
	for _, d := range seen {
		if strings.Contains(d, sentinel) {
			t.Fatalf("a datagram leaks a check error: %q", d)
		}
	}
}
