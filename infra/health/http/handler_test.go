package healthhttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/webitel/webitel-go-kit/infra/health"
)

const (
	textCT = "text/plain; charset=utf-8"
	jsonCT = "application/json; charset=utf-8"
)

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
// config is a parameter because the /livez tests need opposite timings.
func newTestRegistry(t *testing.T, cfg health.Config) *health.Registry {
	t.Helper()

	reg := health.New(cfg, nil)
	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	stopAtCleanup(t, reg)

	return reg
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

// get drives a handler through a recorder. HEAD bodies need a real server; see
// TestHeadNoBody.
func get(h http.Handler, target string) *httptest.ResponseRecorder {
	return do(h, http.MethodGet, target)
}

func do(h http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))

	return rec
}

func assertResponse(t *testing.T, rec *httptest.ResponseRecorder, code int, body, contentType string) {
	t.Helper()

	if rec.Code != code {
		t.Fatalf("status = %d, want %d", rec.Code, code)
	}
	if got := rec.Body.String(); got != body {
		t.Fatalf("body = %q, want %q", got, body)
	}
	if got := rec.Header().Get("Content-Type"); got != contentType {
		t.Fatalf("Content-Type = %q, want %q", got, contentType)
	}
}

func failing(name string) health.Check {
	return func(context.Context) error { return errors.New(name + " is down") }
}

func passing(context.Context) error { return nil }

// emptyRegistry has no checks at all: unknown for readiness, alive for liveness.
func emptyRegistry(t *testing.T) *health.Registry {
	t.Helper()

	return newTestRegistry(t, fastConfig())
}

// okRegistry is a registry with one passing critical check, settled at ready.
func okRegistry(t *testing.T) *health.Registry {
	t.Helper()

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", passing)
	waitState(t, reg, health.StateReady)

	return reg
}

// notReadyRegistry has a failing critical check.
func notReadyRegistry(t *testing.T) *health.Registry {
	t.Helper()

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", failing("db"))
	waitState(t, reg, health.StateNotReady)

	return reg
}

// degradedRegistry has only an informational check failing.
func degradedRegistry(t *testing.T) *health.Registry {
	t.Helper()

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", passing)
	reg.Informational("s3", failing("s3"))
	waitState(t, reg, health.StateDegraded)

	return reg
}

// drainingRegistry is ready, then drained: stopping, and still not wedged.
func drainingRegistry(t *testing.T) *health.Registry {
	t.Helper()

	reg := okRegistry(t)
	reg.Drain()
	waitState(t, reg, health.StateStopping)

	return reg
}

// wedgedRegistry has a failing liveness check, past its threshold.
func wedgedRegistry(t *testing.T) *health.Registry {
	t.Helper()

	// FailThreshold 1: one run settles the check at StatusFail.
	reg := newTestRegistry(t, fastConfig())
	reg.Liveness("wedge", failing("the event loop"))
	waitState(t, reg, health.StateNotReady)

	return reg
}

func TestRouting(t *testing.T) {
	h := Handler(okRegistry(t))

	root := http.NewServeMux()
	root.Handle("/", h)

	exact := http.NewServeMux() // engine's style: one endpoint, one exact path
	exact.Handle("/readyz", h)

	prefix := http.NewServeMux()
	prefix.Handle("/health/", h)

	stripped := http.NewServeMux()
	stripped.Handle("/health/", http.StripPrefix("/health", h))

	// Stands in for gorilla's PathPrefix without adding the dependency: the
	// path reaches the handler unmodified.
	forwarded := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	})

	for _, tt := range []struct {
		name    string
		mount   http.Handler
		targets []string
	}{
		{"root", root, []string{"/livez", "/readyz", "/healthz"}},
		{"exact path", exact, []string{"/readyz"}},
		{"prefix", prefix, []string{"/health/livez", "/health/readyz", "/health/healthz"}},
		{"prefix stripped", stripped, []string{"/health/livez", "/health/readyz", "/health/healthz"}},
		{"prefix forwarded", forwarded, []string{"/health/livez", "/health/readyz", "/health/healthz"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, target := range tt.targets {
				if rec := get(tt.mount, target); rec.Code != http.StatusOK {
					t.Fatalf("%s: status = %d, want %d", target, rec.Code, http.StatusOK)
				}
			}
		})
	}
}

func TestRoutingUnmatched(t *testing.T) {
	h := Handler(okRegistry(t))

	for _, tt := range []struct {
		target string
		code   int
	}{
		{"/", http.StatusNotFound},
		{"/health/", http.StatusNotFound},
		{"/foo", http.StatusNotFound},
		{"/healthz/foo", http.StatusNotFound},
		{"/READYZ", http.StatusNotFound},
		{"/healthz%2Ffoo", http.StatusNotFound}, // arrives decoded; Base is foo
		{"/%68ealthz", http.StatusOK},           // arrives decoded; Base is healthz
		{"/readyz/", http.StatusOK},             // path.Base eats a trailing slash
	} {
		t.Run(tt.target, func(t *testing.T) {
			if rec := get(h, tt.target); rec.Code != tt.code {
				t.Fatalf("status = %d, want %d", rec.Code, tt.code)
			}
		})
	}
}

func TestReadyCode(t *testing.T) {
	for _, tt := range []struct {
		name  string
		setup func(*testing.T) *health.Registry
		state health.State
		code  int
	}{
		{"unknown", emptyRegistry, health.StateUnknown, http.StatusServiceUnavailable},
		{"not_ready", notReadyRegistry, health.StateNotReady, http.StatusServiceUnavailable},
		{"degraded", degradedRegistry, health.StateDegraded, http.StatusOK},
		{"ready", okRegistry, health.StateReady, http.StatusOK},
		{"stopping", drainingRegistry, health.StateStopping, http.StatusServiceUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reg := tt.setup(t)
			if got := reg.Snapshot().State; got != tt.state {
				t.Fatalf("state = %s, want %s", got, tt.state)
			}

			h := Handler(reg)
			for _, target := range []string{"/readyz", "/healthz"} {
				if rec := get(h, target); rec.Code != tt.code {
					t.Fatalf("%s: status = %d, want %d", target, rec.Code, tt.code)
				}
			}
		})
	}
}

func TestLivezEmptyIsAlive(t *testing.T) {
	// The design's inversion: nothing registered means nothing is wedged, even
	// though readiness has no verdict to give.
	h := Handler(emptyRegistry(t))

	assertResponse(t, get(h, "/livez"), http.StatusOK, NameAlive+"\n", textCT)
	assertResponse(t, get(h, "/readyz"), http.StatusServiceUnavailable, health.NameUnknown+"\n", textCT)
}

func TestLivezDrainingIsAlive(t *testing.T) {
	// A watchdog must not kill a process that is shutting down politely.
	h := Handler(drainingRegistry(t))

	assertResponse(t, get(h, "/livez"), http.StatusOK, NameAlive+"\n", textCT)
	assertResponse(t, get(h, "/readyz"), http.StatusServiceUnavailable, health.NameStopping+"\n", textCT)
}

func TestLivezLivenessFail(t *testing.T) {
	// The one case that really is wedged: a liveness check at StatusFail.
	assertResponse(t, get(Handler(wedgedRegistry(t)), "/livez"),
		http.StatusServiceUnavailable, NameWedged+"\n", textCT)
}

func TestLivezUnknownIsAlive(t *testing.T) {
	// A liveness check below its threshold is unknown: not ready, not wedged.
	// FailThreshold 100 keeps it there for the whole test instead of a ~20ms
	// window, and StaleAfter 10s keeps SchedulerAlive true, so the 200 is
	// decided by the liveness rule rather than by the scheduler.
	cfg := health.Config{
		Interval:      50 * time.Millisecond,
		Timeout:       10 * time.Millisecond,
		FailThreshold: 100,
		MinUnready:    time.Millisecond,
		StaleAfter:    10 * time.Second,
		DrainHold:     time.Millisecond,
	}

	reg := newTestRegistry(t, cfg)
	reg.Liveness("wedge", failing("the event loop"))
	waitFor(t, "the first run to complete", func() bool {
		checks := reg.Snapshot().Checks

		return len(checks) == 1 && !checks[0].LastRun.IsZero()
	})

	if got := reg.Snapshot().Checks[0].Status; got != health.StatusUnknown {
		t.Fatalf("check status = %s, want %s", got, health.NameUnknown)
	}

	h := Handler(reg)
	assertResponse(t, get(h, "/livez"), http.StatusOK, NameAlive+"\n", textCT)
	assertResponse(t, get(h, "/readyz"), http.StatusServiceUnavailable, health.NameUnknown+"\n", textCT)
}

func TestLivezSchedulerDead(t *testing.T) {
	// The opposite timing to TestLivezUnknownIsAlive, which is why
	// newTestRegistry takes a config: a short StaleAfter lets the scheduler be
	// observed to stop turning.
	cfg := fastConfig()
	cfg.StaleAfter = 50 * time.Millisecond

	reg := newTestRegistry(t, cfg)
	reg.Critical("db", passing)
	waitState(t, reg, health.StateReady)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := reg.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	waitFor(t, "the scheduler to go stale", func() bool { return !reg.Snapshot().SchedulerAlive })

	assertResponse(t, get(Handler(reg), "/livez"),
		http.StatusServiceUnavailable, NameWedged+"\n", textCT)
}

func TestVerboseStoppingBody(t *testing.T) {
	// The one-word /livez path never pairs 200 with the token stopping; the
	// verbose path does, because it renders the whole snapshot and the
	// snapshot's status is the readiness state, not the liveness verdict.
	// body.go says the same thing where detail() is defined.
	srv := startServer(t, drainingRegistry(t))

	code, body, header := fetch(t, "http://"+srv.Addr()+"/livez?verbose")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want %d", code, http.StatusOK)
	}
	if got := header.Get("Content-Type"); got != jsonCT {
		t.Fatalf("Content-Type = %q, want %q", got, jsonCT)
	}
	if !strings.HasPrefix(body, `{"status":"`+health.NameStopping+`"`) {
		t.Fatalf("body = %s, want the snapshot's stopping status", body)
	}
}

func TestBodyTokens(t *testing.T) {
	stateTokens := []string{
		health.NameUnknown, health.NameNotReady, health.NameDegraded,
		health.NameReady, health.NameStopping,
	}

	for _, tt := range []struct {
		name   string
		setup  func(*testing.T) *health.Registry
		livez  string
		readyz string
	}{
		{"alive", okRegistry, NameAlive + "\n", health.NameReady + "\n"},
		{"wedged", wedgedRegistry, NameWedged + "\n", health.NameNotReady + "\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := Handler(tt.setup(t))

			livez := get(h, "/livez").Body.String()
			if livez != tt.livez {
				t.Fatalf("/livez body = %q, want %q", livez, tt.livez)
			}
			if got := get(h, "/readyz").Body.String(); got != tt.readyz {
				t.Fatalf("/readyz body = %q, want %q", got, tt.readyz)
			}

			for _, token := range stateTokens {
				if livez == token+"\n" {
					t.Fatalf("/livez speaks the state token %q; liveness has its own vocabulary", token)
				}
			}
		})
	}
}

func TestResponseHeaders(t *testing.T) {
	ok, down := okRegistry(t), notReadyRegistry(t)

	for _, tt := range []struct {
		name        string
		reg         *health.Registry
		method      string
		target      string
		code        int
		contentType string
	}{
		{"livez 200", ok, http.MethodGet, "/livez", http.StatusOK, textCT},
		{"readyz 200", ok, http.MethodGet, "/readyz", http.StatusOK, textCT},
		{"healthz 200", ok, http.MethodGet, "/healthz", http.StatusOK, jsonCT},
		{"readyz 503", down, http.MethodGet, "/readyz", http.StatusServiceUnavailable, textCT},
		{"healthz 503", down, http.MethodGet, "/healthz", http.StatusServiceUnavailable, jsonCT},
		{"404", ok, http.MethodGet, "/nope", http.StatusNotFound, textCT},
		{"405", ok, http.MethodPost, "/readyz", http.StatusMethodNotAllowed, textCT},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(Handler(tt.reg), tt.method, tt.target)

			if rec.Code != tt.code {
				t.Fatalf("status = %d, want %d", rec.Code, tt.code)
			}
			if got := rec.Header().Get("Content-Type"); got != tt.contentType {
				t.Fatalf("Content-Type = %q, want %q", got, tt.contentType)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-store" {
				t.Fatalf("Cache-Control = %q, want no-store", got)
			}
			if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
			}
		})
	}
}

func TestMethodNotAllowed(t *testing.T) {
	// engine's gorilla/mux forwards POST /readyz to the handler at 200, so the
	// method gate is ours to enforce; the router gives us nothing.
	h := Handler(okRegistry(t))

	for _, method := range []string{
		http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions,
	} {
		for _, target := range []string{"/livez", "/readyz", "/healthz"} {
			t.Run(method+" "+target, func(t *testing.T) {
				rec := do(h, method, target)

				if rec.Code != http.StatusMethodNotAllowed {
					t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
				}
				if got := rec.Header().Get("Allow"); got != allowedMethods {
					t.Fatalf("Allow = %q, want %q", got, allowedMethods)
				}
			})
		}
	}
}

func TestHeadNoBody(t *testing.T) {
	// A real server: httptest.NewRecorder does not emulate net/http discarding
	// a HEAD body, so a recorder-based version of this test asserts nothing.
	srv := httptest.NewServer(Handler(okRegistry(t)))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodHead, srv.URL+"/readyz", nil)
	if err != nil {
		t.Fatalf("building the request: %v", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HEAD /readyz: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("reading the body: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if res.ContentLength <= 0 {
		t.Fatalf("Content-Length = %d, want the length of the body it would have sent", res.ContentLength)
	}
	if len(body) != 0 {
		t.Fatalf("body = %q, want no bytes at all", body)
	}
}

func TestVerboseIgnoredOnHandler(t *testing.T) {
	// A Handler has no address, so it can never satisfy the loopback rule.
	reg := okRegistry(t)
	queries := []string{"?verbose", "?verbose=", "?verbose=0", "?verbose=false"}

	for _, ep := range []string{epLivez, epReadyz, epHealthz} {
		want := get(Handler(reg), "/"+ep).Body.String()

		for _, query := range queries {
			for _, h := range []http.Handler{Handler(reg), singleEndpointHandler(t, reg, ep)} {
				if got := get(h, "/"+ep+query).Body.String(); got != want {
					t.Fatalf("/%s%s body = %q, want %q", ep, query, got, want)
				}
			}
		}

		// /healthz is JSON either way; the other two must stay one-word.
		if strings.HasPrefix(want, "{") != (ep == epHealthz) {
			t.Fatalf("/%s body = %q, want the one-word body", ep, want)
		}
	}
}

func TestNilRegistry(t *testing.T) {
	h := Handler(nil)

	assertResponse(t, get(h, "/livez"), http.StatusServiceUnavailable, NameWedged+"\n", textCT)
	assertResponse(t, get(h, "/readyz"), http.StatusServiceUnavailable, health.NameUnknown+"\n", textCT)
	assertResponse(t, get(h, "/healthz"), http.StatusServiceUnavailable,
		`{"status":"unknown","checks":[]}`, jsonCT)
}

// assertAgrees pins the /readyz code to ReadyFunc's bool in one state.
func assertAgrees(t *testing.T, reg *health.Registry, want health.State) {
	t.Helper()

	if got := reg.Snapshot().State; got != want {
		t.Fatalf("state = %s, want %s", got, want)
	}

	ok, err := reg.ReadyFunc()()
	code := get(Handler(reg), "/readyz").Code

	if ok != (code == http.StatusOK) {
		t.Fatalf("%s: ReadyFunc = %v but /readyz = %d", want, ok, code)
	}
	if !ok && err == nil {
		t.Fatalf("%s: a false verdict with a nil error would panic engine", want)
	}
}

func TestVerdictAgreesWithReadyFunc(t *testing.T) {
	// The transport must never drift from service discovery. The steps are
	// ordered: MinUnready gates the way back to ready, and StateStopping is
	// one-way, so Drain comes last.
	var dbDown, s3Down atomic.Bool

	reg := health.New(fastConfig(), nil)
	reg.Critical("db", func(context.Context) error {
		if dbDown.Load() {
			return errors.New("db is down")
		}

		return nil
	})
	reg.Informational("s3", func(context.Context) error {
		if s3Down.Load() {
			return errors.New("s3 is down")
		}

		return nil
	})

	// Cold, before any run: asserted before Start, because the first run lands
	// within a millisecond of it.
	assertAgrees(t, reg, health.StateUnknown)

	if err := reg.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	stopAtCleanup(t, reg)

	waitState(t, reg, health.StateReady)
	assertAgrees(t, reg, health.StateReady)

	dbDown.Store(true)
	waitState(t, reg, health.StateNotReady)
	assertAgrees(t, reg, health.StateNotReady)

	dbDown.Store(false)
	waitState(t, reg, health.StateReady) // only once MinUnready has elapsed
	assertAgrees(t, reg, health.StateReady)

	s3Down.Store(true)
	waitState(t, reg, health.StateDegraded)
	assertAgrees(t, reg, health.StateDegraded)

	reg.Drain()
	waitState(t, reg, health.StateStopping)
	assertAgrees(t, reg, health.StateStopping)
}

func TestConcurrentScrape(t *testing.T) {
	// The assertion is that nothing panics and -race stays silent.
	var dbDown atomic.Bool

	reg := newTestRegistry(t, fastConfig())
	reg.Critical("db", func(context.Context) error {
		if dbDown.Load() {
			return errors.New("db is down")
		}

		return nil
	})

	h := Handler(reg)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for range 50 {
				for _, target := range []string{"/livez", "/readyz", "/healthz"} {
					get(h, target)
				}
			}
		}()
	}

	for range 10 {
		dbDown.Store(!dbDown.Load())
		time.Sleep(time.Millisecond)
	}
	reg.Drain()

	wg.Wait()
}
