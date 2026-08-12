package healthhttp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/webitel/webitel-go-kit/infra/health"
)

// Run under -race. Most of these assert only "no race, no panic, no deadlock,
// no leak" — the detector and the test timeout are the assertions.

func racyConfig() health.Config {
	return health.Config{
		Interval:      time.Millisecond,
		Timeout:       500 * time.Microsecond,
		FailThreshold: 1,
		MinUnready:    time.Microsecond,
		StaleAfter:    50 * time.Millisecond,
		DrainHold:     time.Millisecond,
	}
}

func racyRegistry(t *testing.T) *health.Registry {
	t.Helper()

	reg := health.New(racyConfig(), nil)
	reg.Critical("db", func(context.Context) error { return nil })
	reg.Informational("s3", func(context.Context) error { return nil })

	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		reg.Stop(ctx)
	})

	return reg
}

func settled(t *testing.T, want int) int {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	got := runtime.NumGoroutine()
	for time.Now().Before(deadline) {
		got = runtime.NumGoroutine()
		if got <= want {
			return got
		}
		time.Sleep(2 * time.Millisecond)
	}

	return got
}

// TestRaceSecondStartIsRefused is the regression test for the shipped data race
// found in review: a second Start bound another port and rewrote the verbose
// grant while requests were already in flight on the first listener.
func TestRaceSecondStartIsRefused(t *testing.T) {
	reg := racyRegistry(t)
	srv := NewServer(reg, "127.0.0.1:0")

	if err := srv.Start(); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	t.Cleanup(func() { srv.Stop(context.Background()) })

	first := srv.Addr()

	if err := srv.Start(); err == nil {
		t.Fatal("second Start succeeded; it must be refused, not bind another port")
	}
	if got := srv.Addr(); got != first {
		t.Fatalf("Addr moved from %q to %q after a refused Start", first, got)
	}
}

func TestRaceConcurrentStartsBindOnePort(t *testing.T) {
	reg := racyRegistry(t)

	for round := 0; round < 10; round++ {
		srv := NewServer(reg, "127.0.0.1:0")

		var mu sync.Mutex
		var wins int

		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := srv.Start(); err == nil {
					mu.Lock()
					wins++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		if wins != 1 {
			t.Fatalf("round %d: %d concurrent Starts succeeded, want exactly 1", round, wins)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		srv.Stop(ctx)
		cancel()
	}
}

func TestRaceRequestsWhileStartingAndStopping(t *testing.T) {
	// Requests in flight while the lifecycle moves under them.
	reg := racyRegistry(t)

	for round := 0; round < 5; round++ {
		srv := NewServer(reg, "127.0.0.1:0")
		if err := srv.Start(); err != nil {
			t.Fatalf("Start: %v", err)
		}
		base := "http://" + srv.Addr()

		var wg sync.WaitGroup
		stop := make(chan struct{})
		for i := 0; i < 6; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				c := &http.Client{Timeout: time.Second}
				for {
					select {
					case <-stop:
						return
					default:
					}
					for _, p := range []string{"/livez", "/readyz", "/healthz", "/readyz?verbose"} {
						resp, err := c.Get(base + p)
						if err != nil {
							continue // the server is going away; that is the point
						}
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					}
					_ = srv.Addr()
				}
			}()
		}

		time.Sleep(20 * time.Millisecond)
		close(stop)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := srv.Stop(ctx); err != nil {
			t.Fatalf("round %d: Stop: %v", round, err)
		}
		cancel()
		wg.Wait()
	}
}

func TestRaceStopBeforeStartIsANoOp(t *testing.T) {
	reg := racyRegistry(t)
	srv := NewServer(reg, "127.0.0.1:0")

	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("Stop before Start: %v", err)
	}
	if got := srv.Addr(); got != "127.0.0.1:0" {
		t.Fatalf("Addr = %q before Start, want the configured address", got)
	}
}

func TestRaceConcurrentStopsAreSafe(t *testing.T) {
	reg := racyRegistry(t)

	for round := 0; round < 10; round++ {
		srv := NewServer(reg, "127.0.0.1:0")
		if err := srv.Start(); err != nil {
			t.Fatalf("Start: %v", err)
		}

		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				srv.Stop(ctx)
			}()
		}
		wg.Wait()
	}
}

func TestRaceServerCyclesDoNotLeakGoroutines(t *testing.T) {
	reg := racyRegistry(t)
	base := runtime.NumGoroutine()

	for i := 0; i < 25; i++ {
		srv := NewServer(reg, "127.0.0.1:0")
		if err := srv.Start(); err != nil {
			t.Fatalf("cycle %d: Start: %v", i, err)
		}

		resp, err := http.Get("http://" + srv.Addr() + "/readyz")
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := srv.Stop(ctx); err != nil {
			t.Fatalf("cycle %d: Stop: %v", i, err)
		}
		cancel()
	}

	if got := settled(t, base+4); got > base+4 {
		t.Fatalf("goroutines %d -> %d after 25 server cycles", base, got)
	}
}

func TestRaceHandlerUnderLoadWhileRegistryChurns(t *testing.T) {
	// The core contract: transports only read a snapshot, so scraping hard must
	// never disturb — or be disturbed by — the scheduler.
	reg := health.New(racyConfig(), nil)
	reg.Critical("db", func(context.Context) error { return nil })
	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		reg.Stop(ctx)
	})

	h := Handler(reg)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				for _, p := range []string{"/livez", "/readyz", "/healthz"} {
					rec := httptest.NewRecorder()
					h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
					if rec.Code != http.StatusOK && rec.Code != http.StatusServiceUnavailable {
						panic("unexpected status " + strconv.Itoa(rec.Code) + " on " + p)
					}
					if strings.Contains(rec.Body.String(), "sentinel") {
						panic("check error text leaked into the body: " + rec.Body.String())
					}
				}
			}
		}()
	}

	// Churn the registry underneath the readers.
	for i := 0; i < 40; i++ {
		reg.Informational("late"+strconv.Itoa(i), func(context.Context) error {
			return errSentinel
		})
		time.Sleep(time.Millisecond)
	}
	reg.Drain()

	time.Sleep(10 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestRaceNilServerAndNilRegistryUnderConcurrency(t *testing.T) {
	var srv *Server

	var reg *health.Registry
	h := Handler(reg)

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_ = srv.Start()
			_ = srv.Stop(context.Background())
			_ = srv.Addr()

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if rec.Code != http.StatusServiceUnavailable {
				panic("nil registry served " + strconv.Itoa(rec.Code) + ", want 503")
			}
		}()
	}
	wg.Wait()
}

// TestOnlyInformationalChecksAreNotReady is the regression test for the spec
// violation found in review: a registry holding only informational checks
// reported Ready — and answered /readyz 200 — before any check had run.
func TestOnlyInformationalChecksAreNotReady(t *testing.T) {
	reg := health.New(racyConfig(), nil)
	reg.Informational("rabbit", func(context.Context) error { return nil })
	reg.Informational("s3", func(context.Context) error { return nil })

	// Deliberately not started: nothing has run, so nothing can be ready.
	rec := httptest.NewRecorder()
	Handler(reg).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("/readyz = %d with only informational checks and no run, want 503; body %q",
			rec.Code, rec.Body.String())
	}

	ok, err := reg.ReadyFunc()()
	if ok {
		t.Fatal("verdict is true with only informational checks and no completed run")
	}
	if err == nil {
		t.Fatal("false verdict with a nil error")
	}
	if !strings.Contains(err.Error(), "liveness or critical") {
		t.Fatalf("got %q, want an error naming the missing counting checks", err)
	}
}

var errSentinel = &sentinelError{}

type sentinelError struct{}

func (*sentinelError) Error() string { return "sentinel-dsn-secret" }
