package sdnotify

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/webitel/webitel-go-kit/infra/health"
)

// socketNonce keeps abstract socket names unique: -count=3 runs three
// iterations in one process, so a bare PID would collide with itself.
var socketNonce atomic.Uint64

// notifySocket returns a receiving unixgram socket on a path short enough for
// darwin's 104-byte sun_path. t.TempDir() is unusable here: on darwin it is
// already 80 bytes for a short test name and over 125 for a subtest, and the
// bind then fails with a bare "invalid argument".
func notifySocket(t *testing.T) (string, *net.UnixConn) {
	t.Helper()

	// Hermetic by default: a runner that exports a watchdog environment would
	// otherwise break every "want nothing" assertion in this package, and no
	// quiet window would ever elapse. The watchdog tests set their own values
	// after calling this.
	t.Setenv("WATCHDOG_USEC", "")
	t.Setenv("WATCHDOG_PID", "")

	// os.TempDir honours TMPDIR, which a sandbox may redirect when /tmp is
	// read-only. Darwin's per-user TMPDIR is far too long for sun_path, so /tmp
	// stays the fallback and the length guard below still has the last word.
	base := os.TempDir()
	if len(base) > 40 {
		base = "/tmp"
	}

	dir, err := os.MkdirTemp(base, "h")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	addr := filepath.Join(dir, "n.sock")
	if len(addr) > 100 {
		t.Fatalf("socket path is %d bytes, too long for this platform: %s", len(addr), addr)
	}

	ln, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: addr, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	return addr, ln
}

// rebind destroys the receiving socket file and listens on the same path again.
// The os.Remove is mandatory: net.ListenUnixgram has no unlink-on-close — that
// lives on UnixListener, the stream type — so the file outlives Close and a
// second bind fails with "address already in use".
func rebind(t *testing.T, addr string) *net.UnixConn {
	t.Helper()

	if err := os.Remove(addr); err != nil {
		t.Fatal(err)
	}

	ln, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: addr, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	return ln
}

// recv reads one datagram.
func recv(t *testing.T, ln *net.UnixConn, timeout time.Duration) string {
	t.Helper()

	if err := ln.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 4096)
	n, err := ln.Read(buf)
	if err != nil {
		t.Fatalf("waiting %s for a datagram: %v", timeout, err)
	}

	return string(buf[:n])
}

// recvAll reads datagrams until quiet elapses with none arriving. The budget is
// not optional: against a live watchdog ticker no quiet window ever elapses, and
// a test that hangs until go test's ten-minute timeout is strictly worse than
// one that fails legibly in a second.
func recvAll(t *testing.T, ln *net.UnixConn, quiet, budget time.Duration) []string {
	t.Helper()

	var got []string
	end := time.Now().Add(budget)

	for {
		if time.Now().After(end) {
			t.Fatalf("no quiet window of %s within %s; got %d datagrams: %q", quiet, budget, len(got), got)
		}
		if err := ln.SetReadDeadline(time.Now().Add(quiet)); err != nil {
			t.Fatal(err)
		}

		buf := make([]byte, 4096)
		n, err := ln.Read(buf)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				return got
			}

			t.Fatal(err)
		}

		got = append(got, string(buf[:n]))
	}
}

// countEqual counts datagrams equal to want.
func countEqual(got []string, want string) int {
	n := 0
	for _, d := range got {
		if d == want {
			n++
		}
	}

	return n
}

// fastConfig is a starting point for tests that just need a registry which
// settles quickly. Tests with their own timing requirements override it.
func fastConfig() health.Config {
	return health.Config{
		Interval:      10 * time.Millisecond,
		Timeout:       5 * time.Millisecond,
		FailThreshold: 1,
		MinUnready:    time.Millisecond,
		StaleAfter:    500 * time.Millisecond,
		DrainHold:     time.Millisecond,
	}
}

func stopAtCleanup(t *testing.T, reg *health.Registry) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		if err := reg.Stop(ctx); err != nil {
			t.Errorf("Stop: %v", err)
		}
	})
}

// newTestRegistry returns a started registry, stopped again at cleanup. The
// config is a parameter because the watchdog tests need a shorter StaleAfter.
func newTestRegistry(t *testing.T, cfg health.Config) *health.Registry {
	t.Helper()

	reg := health.New(cfg, nil)
	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	stopAtCleanup(t, reg)

	return reg
}

// newNotifier sets NOTIFY_SOCKET, builds a Notifier and stops it at cleanup.
// The environment must be set before New, which reads it once.
func newNotifier(t *testing.T, reg *health.Registry, addr string, opts ...Option) *Notifier {
	t.Helper()

	t.Setenv("NOTIFY_SOCKET", addr)

	n := New(reg, opts...)
	if n == nil {
		t.Fatal("New returned nil with NOTIFY_SOCKET set")
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		if err := n.Stop(ctx); err != nil {
			t.Errorf("Stop: %v", err)
		}
	})

	return n
}

// waitState polls until the registry reports want. Never sleep a fixed
// duration waiting for a state: -race -count=3 turns that into a flake.
func waitState(t *testing.T, reg *health.Registry, want health.State) {
	t.Helper()

	var got health.State
	for range 2000 {
		got = reg.Snapshot().State
		if got == want {
			return
		}

		time.Sleep(time.Millisecond)
	}

	t.Fatalf("state = %s, want %s", got, want)
}

// waitFor polls cond for up to two seconds.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()

	for range 2000 {
		if cond() {
			return
		}

		time.Sleep(time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", what)
}

func failing(name string) health.Check {
	return func(context.Context) error { return errors.New(name + " is down") }
}

func passing(context.Context) error { return nil }

// toggle is a check a test can flip without racing the registry's goroutines.
type toggle struct {
	down atomic.Bool
	err  error
}

func newToggle(err error) *toggle {
	return &toggle{err: err}
}

func (tg *toggle) check(context.Context) error {
	if tg.down.Load() {
		return tg.err
	}

	return nil
}

func (tg *toggle) fail() { tg.down.Store(true) }

func (tg *toggle) pass() { tg.down.Store(false) }

// logCapture collects records so a test can assert on what was logged. The
// mutex is genuinely needed and test-only: records are appended from the loop
// goroutine and read by the test.
type logCapture struct {
	mu      sync.Mutex
	records []slog.Record
}

func (c *logCapture) Enabled(context.Context, slog.Level) bool { return true }

func (c *logCapture) Handle(_ context.Context, r slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.records = append(c.records, r.Clone())

	return nil
}

func (c *logCapture) WithAttrs([]slog.Attr) slog.Handler { return c }

func (c *logCapture) WithGroup(string) slog.Handler { return c }

func (c *logCapture) logger() *slog.Logger { return slog.New(c) }

// errs returns every attribute named err whose value is an error.
func (c *logCapture) errs() []error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var out []error
	for _, r := range c.records {
		r.Attrs(func(a slog.Attr) bool {
			if err, ok := a.Value.Resolve().Any().(error); ok && a.Key == "err" {
				out = append(out, err)
			}

			return true
		})
	}

	return out
}

// text returns every message and attribute flattened.
func (c *logCapture) text() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	var b strings.Builder
	for _, r := range c.records {
		b.WriteString(r.Message)
		r.Attrs(func(a slog.Attr) bool {
			fmt.Fprintf(&b, " %s=%v", a.Key, a.Value)

			return true
		})
		b.WriteString("\n")
	}

	return b.String()
}
