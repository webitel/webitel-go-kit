package prometheus

import (
	"context"
	"errors"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Run under -race. Most of these assert only "no race, no panic, no
// deadlock, no leak" -- the detector and the test timeout are the
// assertions.

// settled polls runtime.NumGoroutine() to a deadline instead of sleeping a
// fixed amount, so it does not depend on how fast the machine is.
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

// TestRaceServerCyclesDoNotLeakGoroutines drives 25 create/scrape/shutdown
// cycles. It doubles as the port-released proof: 25 successive binds cannot
// all succeed if a prior cycle's port leaked.
func TestRaceServerCyclesDoNotLeakGoroutines(t *testing.T) {
	base := runtime.NumGoroutine()

	for i := 0; i < 25; i++ {
		r, err := newReader(context.Background(), "127.0.0.1", "0")
		if err != nil {
			t.Fatalf("cycle %d: newReader: %v", i, err)
		}

		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))

		// The scrape must SUCCEED, not merely be attempted. Swallowing this
		// error would let all 25 scrapes fail while the test still passed,
		// leaving it a goroutine test that no longer proves the endpoint on
		// each freshly bound port actually served.
		resp, err := http.Get("http://" + r.addr() + metricsPath)
		if err != nil {
			t.Fatalf("cycle %d: scrape: %v", i, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel() // deferred: the t.Fatalf path must cancel too
			if err := mp.Shutdown(ctx); err != nil {
				t.Fatalf("cycle %d: Shutdown: %v", i, err)
			}
		}()
	}

	// goroutineSlack absorbs runtime/HTTP-transport goroutines still unwinding
	// from earlier tests; it is a tolerance, not a target. A real per-cycle
	// leak would be ~25, far outside it.
	const goroutineSlack = 4
	if got := settled(t, base+goroutineSlack); got > base+goroutineSlack {
		t.Fatalf("goroutines %d -> %d after 25 reader cycles", base, got)
	}
}

// TestRaceRequestsInFlightAcrossShutdown drives GETs concurrently with
// Shutdown. Only two outcomes are legal: full data, or a refused connection.
// A connection error is expected and skipped -- the endpoint going away is
// the point -- but any response that DOES come back HTTP 200 must be
// inspected: an empty body on 200 is exactly the reader-first bug, and it is
// not a Go error, so a bare continue-on-error loop would sail past it.
func TestRaceRequestsInFlightAcrossShutdown(t *testing.T) {
	r, err := newReader(context.Background(), "127.0.0.1", "0")
	if err != nil {
		t.Fatalf("newReader: %v", err)
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))
	addr := "http://" + r.addr() + metricsPath

	var wg sync.WaitGroup
	stop := make(chan struct{})
	var badResponses atomic.Int64
	ready := make(chan struct{})
	var readyOnce sync.Once

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

				resp, err := c.Get(addr)
				if err != nil {
					continue // the endpoint going away is the point
				}

				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					if strings.Contains(string(body), "target_info") {
						readyOnce.Do(func() { close(ready) })
					} else {
						badResponses.Add(1)
					}
				}
			}
		}()
	}

	// Synchronize on an OBSERVED successful scrape rather than on the clock.
	// A fixed sleep that is too short on a loaded CI runner means no request
	// ever overlaps the shutdown: badResponses is trivially 0 and the test
	// passes while proving nothing -- and it never fails, so the erosion is
	// invisible.
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("no goroutine completed a scrape before shutdown; the race window never opened")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := mp.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	cancel()

	close(stop)
	wg.Wait()

	if n := badResponses.Load(); n > 0 {
		t.Fatalf("%d responses were HTTP 200 with an empty/incomplete body -- the reader-first bug", n)
	}
}

// TestRaceConcurrentShutdown fires N goroutines calling Shutdown at once on
// the same reader; the sync.Once must mean the underlying stop/reader
// shutdown runs exactly once, with no panic and no data race.
func TestRaceConcurrentShutdown(t *testing.T) {
	var stopCalls atomic.Int64

	// A sentinel first-call error: with stop returning nil, an implementation
	// that handed every racing caller a FRESH nil -- rather than the memoized
	// result -- would pass on stopCalls alone. Collecting the returned errors
	// is what closes that gap concurrently.
	errStop := errors.New("sentinel: concurrent stop failed")

	r := fakeReader(t, sdkmetric.NewManualReader(), func(context.Context) error {
		stopCalls.Add(1)
		return errStop
	})

	const callers = 8
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err := r.Shutdown(ctx)
			mu.Lock()
			errs = append(errs, err)
			mu.Unlock()
		}()
	}
	wg.Wait()

	if got := stopCalls.Load(); got != 1 {
		t.Fatalf("stop ran %d times across concurrent Shutdown calls, want exactly 1", got)
	}

	if len(errs) != callers {
		t.Fatalf("collected %d results, want %d", len(errs), callers)
	}
	for i, err := range errs {
		if !errors.Is(err, errStop) {
			t.Fatalf("caller %d got %v, want the memoized %v", i, err, errStop)
		}
		if err != errs[0] {
			t.Fatalf("caller %d got a different error value than caller 0 -- memoization is broken", i)
		}
	}
}
