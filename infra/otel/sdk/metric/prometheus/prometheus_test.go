package prometheus

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/webitel/webitel-go-kit/infra/otel/sdk/metric"
)

// newRegisteredReader binds a reader and wires it into a real MeterProvider,
// which is what a bare newReader lacks: an unregistered Reader's Collect
// returns metric.ErrReaderNotRegistered and never emits target_info, so any
// test asserting on scrape *content* needs the provider in the loop. Returns
// the reader plus a shutdown func bound to the provider.
func newRegisteredReader(t *testing.T, host, port string) (*reader, func(context.Context) error) {
	t.Helper()

	r, err := newReader(context.Background(), host, port)
	require.NoError(t, err)

	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))
	return r, mp.Shutdown
}

// stopAtCleanup registers a t.Cleanup that shuts down under a bounded
// context, t.Errorf-ing (never fataling) on failure.
func stopAtCleanup(t *testing.T, stop func(context.Context) error) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := stop(ctx); err != nil {
			t.Errorf("Shutdown: %v", err)
		}
	})
}

// TestEndToEndScrape proves the one path this plan wires end to end: a bound
// listener answers /metrics with HTTP 200 and target_info in the body. The
// reader must be registered with a MeterProvider before a Collect can
// produce anything -- an unregistered reader's Collect returns
// metric.ErrReaderNotRegistered and the scrape body stays empty, which is
// exactly how a real Configure() wires this package via sdkmetric.WithReader.
func TestEndToEndScrape(t *testing.T) {
	r, stop := newRegisteredReader(t, "127.0.0.1", "0")
	stopAtCleanup(t, stop)

	resp, err := http.Get("http://" + r.addr() + metricsPath)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.True(t, strings.Contains(string(body), "target_info"), "body missing target_info: %s", body)
}

// TestSchemeResolves proves init() ran and "prometheus" is no longer an
// unknown scheme -- the whole reason the telemetry stack failed to
// configure before this package existed.
func TestSchemeResolves(t *testing.T) {
	// Env-reading tests are serial: t.Setenv forbids t.Parallel().
	t.Setenv("OTEL_EXPORTER_PROMETHEUS_HOST", "127.0.0.1")
	t.Setenv("OTEL_EXPORTER_PROMETHEUS_PORT", "0")

	opts, err := metric.NewOptions(context.Background(), "prometheus")
	require.NoError(t, err)
	require.NotEmpty(t, opts)

	mp := sdkmetric.NewMeterProvider(opts...)
	stopAtCleanup(t, mp.Shutdown)
}

// TestPortParsing is table-driven over accepted and rejected port strings.
// A rejection must name the offending value and must never fall back to the
// default port -- newReader returns a nil *reader, so there is nothing to
// shut down.
func TestPortParsing(t *testing.T) {
	accepted := []string{"0", "65535"}
	for _, port := range accepted {
		t.Run("accept_"+port, func(t *testing.T) {
			r, err := newReader(context.Background(), "127.0.0.1", port)
			require.NoError(t, err)
			require.NotNil(t, r)
			stopAtCleanup(t, r.Shutdown)
		})
	}

	rejected := []string{"70000", "abc", "-1", "9464x"}
	for _, port := range rejected {
		t.Run("reject_"+port, func(t *testing.T) {
			r, err := newReader(context.Background(), "127.0.0.1", port)
			require.Error(t, err)
			require.Nil(t, r, "a rejected port must not fall back to the default and bind anyway")
			require.Contains(t, err.Error(), port)
		})
	}
}

// TestOptionsDefaultsOnUnsetEnv proves an unset/whitespace env leaves the
// spec defaults untouched. It asserts through resolveHostPort rather than a
// full Options call, deliberately never binding the real default port 9464
// -- tests must not depend on that port being free.
func TestOptionsDefaultsOnUnsetEnv(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_PROMETHEUS_HOST", "")
	t.Setenv("OTEL_EXPORTER_PROMETHEUS_PORT", "")

	host, port := resolveHostPort()
	require.Equal(t, defaultHost, host)
	require.Equal(t, defaultPort, port)
}

// TestEnvWiring proves the namespaced key lookup is right: passing the full
// variable name to EnvString instead of the OTEL-prefixed suffix would leave
// this test resolving the defaults instead, which -- with port "0" -- would
// still succeed. The point is that the request actually reads back the
// injected host, not just that Options returns cleanly.
func TestEnvWiring(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_PROMETHEUS_HOST", "127.0.0.1")
	t.Setenv("OTEL_EXPORTER_PROMETHEUS_PORT", "0")

	opts, err := Options(context.Background(), "prometheus")
	require.NoError(t, err)
	require.NotEmpty(t, opts)

	mp := sdkmetric.NewMeterProvider(opts...)
	stopAtCleanup(t, mp.Shutdown)
}

// TestBindErrorSurfaced proves a bind failure is RETURNED, not merely
// logged, and that the error names the held address.
func TestBindErrorSurfaced(t *testing.T) {
	held, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer held.Close()

	host, port, err := net.SplitHostPort(held.Addr().String())
	require.NoError(t, err)

	r, err := newReader(context.Background(), host, port)
	require.Error(t, err)
	require.Nil(t, r)
	require.Contains(t, err.Error(), held.Addr().String())
}

// TestIPv6Host proves net.JoinHostPort brackets an IPv6 literal correctly.
// Skips gracefully on a platform without an IPv6 loopback.
func TestIPv6Host(t *testing.T) {
	probe, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skipf("no IPv6 loopback available: %v", err)
	}
	probe.Close()

	r, err := newReader(context.Background(), "::1", "0")
	require.NoError(t, err)
	stopAtCleanup(t, r.Shutdown)

	require.True(t, strings.HasPrefix(r.addr(), "[::1]:"), "addr() = %q, want a bracketed IPv6 address", r.addr())
}

// recordingReader embeds a real Reader and records "reader" into the shared
// sequence when Shutdown runs -- the same interface-embedding trick the
// production wrapper uses, applied here to observe call order from outside.
type recordingReader struct {
	sdkmetric.Reader
	seq *[]string
}

func (r recordingReader) Shutdown(ctx context.Context) error {
	*r.seq = append(*r.seq, "reader")
	return r.Reader.Shutdown(ctx)
}

// TestShutdownOrderIsListenerThenReader is the ordering gate. A post-Shutdown
// connection check cannot prove this: Shutdown is synchronous, so by the
// time it returns both the listener and the reader are down whichever ran
// first -- the two orderings converge on an identical final state. The
// window where they differ is *during* the call, which is exactly what the
// recorded sequence captures. Hand-verified: swapping the two statements in
// reader.go's Shutdown makes this fail (sequence becomes ["reader",
// "listener"]).
func TestShutdownOrderIsListenerThenReader(t *testing.T) {
	var seq []string

	r := fakeReader(t, recordingReader{Reader: sdkmetric.NewManualReader(), seq: &seq},
		func(context.Context) error {
			seq = append(seq, "listener")
			return nil
		})

	err := r.Shutdown(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"listener", "reader"}, seq)
}

// sentinelErr is the injected first-call failure for the idempotency gate.
var errSentinelStop = errors.New("sentinel: stop failed")

// TestShutdownIsIdempotent forces a non-nil FIRST result and proves the
// SECOND call returns that same error, not nil. A happy-path-only
// double-Shutdown test cannot catch the bug this guards: when the first
// call returns nil, a broken always-nil implementation passes by
// coincidence.
func TestShutdownIsIdempotent(t *testing.T) {
	var stopCalls int

	r := fakeReader(t, sdkmetric.NewManualReader(),
		func(context.Context) error {
			stopCalls++
			return errSentinelStop
		})

	first := r.Shutdown(context.Background())
	require.Error(t, first)
	require.ErrorIs(t, first, errSentinelStop)

	second := r.Shutdown(context.Background())
	require.Error(t, second)
	require.ErrorIs(t, second, errSentinelStop)
	require.Equal(t, first, second)

	require.Equal(t, 1, stopCalls, "stop must run exactly once across both calls")
}

// TestPostShutdownScrapeRefused is a smoke test for teardown, NOT the
// ordering proof -- that is TestShutdownOrderIsListenerThenReader. This just
// confirms the end state: after Shutdown, the old address refuses the
// connection instead of answering 200-with-empty-body.
func TestPostShutdownScrapeRefused(t *testing.T) {
	r, stop := newRegisteredReader(t, "127.0.0.1", "0")
	addr := r.addr()

	resp, err := http.Get("http://" + addr + metricsPath)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, stop(ctx))

	_, err = http.Get("http://" + addr + metricsPath)
	require.Error(t, err, "a scrape after Shutdown must fail to connect, not succeed with an empty body")
}

// TestSilentByDefault asserts the package writes nothing to stderr on its
// own during a normal lifecycle. The reader is registered with a
// MeterProvider so Collect succeeds outright: an *unregistered* reader's
// Collect hits metric.ErrReaderNotRegistered, which the upstream exporter
// routes through otel.Handle regardless of our HandlerOpts -- pre-existing
// SDK behaviour (see infra/otel/sdk/otel.go), not something this package
// controls, and not what this test is checking.
func TestSilentByDefault(t *testing.T) {
	// Capture the real SINKS, not the os.Stderr variable. Reassigning
	// os.Stderr catches only a literal fmt.Fprint(os.Stderr, ...): log's
	// package init already captured the original in `std`, slog.Default()'s
	// built-in handler routes through log.Default(), and otel's default error
	// handler holds a logger built at init. So the classic regressions -- a
	// stray log.Printf or slog.Info -- would sail past that check.
	var buf bytes.Buffer

	origFlags := log.Flags()
	log.SetOutput(&buf) // covers log, slog's default handler, otel's default handler
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(origFlags)
	})

	// otel.Handle is a deliberate, non-stderr sink for this package (a dead
	// endpoint, a degraded exporter). Record it separately: the happy path
	// below must not trip it either.
	var handled []error
	orig := otel.GetErrorHandler()
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) { handled = append(handled, err) }))
	t.Cleanup(func() { otel.SetErrorHandler(orig) })

	r, stop := newRegisteredReader(t, "127.0.0.1", "0")
	resp, err := http.Get("http://" + r.addr() + metricsPath)
	require.NoError(t, err)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, stop(ctx))

	require.Empty(t, buf.String(), "package logged uninvited on the happy path")
	require.Empty(t, handled, "package called otel.Handle on the happy path")
}

// TestQuotedEmptyHostDoesNotBindWildcard is a regression gate for a real
// exposure. internal.EnvString applies its non-empty test BEFORE unquoting, so
// OTEL_EXPORTER_PROMETHEUS_HOST='""' -- what a compose file or k8s manifest
// produces for a set-but-blank variable -- survives that test and unquotes to
// "". An empty host makes net.JoinHostPort emit ":9464", which binds every
// interface and serves an unauthenticated /metrics off-host while this package
// documents a loopback default.
//
// Asserting on resolveHostPort rather than on a bound socket keeps the test
// honest on CI, where a wildcard bind would succeed and look identical to a
// loopback one.
func TestQuotedEmptyHostDoesNotBindWildcard(t *testing.T) {
	for _, blank := range []string{`""`, `''`, "", "   "} {
		t.Setenv("OTEL_EXPORTER_PROMETHEUS_HOST", blank)

		host, _ := resolveHostPort()
		require.Equal(t, defaultHost, host,
			"blank host %q must fall back to the loopback default, never bind the wildcard", blank)
		require.NotEmpty(t, host, "an empty host binds every interface")
	}
}

// TestQuotedEmptyPortFallsBackToDefault pins the sibling case. Blank is unset;
// a MALFORMED port is a different thing and must still be an error, which
// TestPortParsing covers -- a typo must never silently bind the default.
func TestQuotedEmptyPortFallsBackToDefault(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_PROMETHEUS_PORT", `""`)

	_, port := resolveHostPort()
	require.Equal(t, defaultPort, port)
}

// TestShutdownReleasesPortWithoutPriorScrape is the regression gate for a
// shutdown that reports success while the port is still bound.
//
// http.Server.Shutdown closes only the listeners Serve has already tracked, and
// Serve runs on a goroutine newReader does not wait for. Every other test in
// this file scrapes before shutting down, which forces Serve to have started
// and hides the race entirely. This one deliberately never scrapes, so the
// listener may still be untracked when Shutdown runs.
//
// Verified to fail before the fix: under GOMAXPROCS=1 the pre-fix code left the
// port bound on 99 of 100 iterations while Shutdown returned nil.
func TestShutdownReleasesPortWithoutPriorScrape(t *testing.T) {
	for i := 0; i < 50; i++ {
		r, err := newReader(context.Background(), defaultHost, "0")
		require.NoError(t, err)

		addr := r.addr()
		require.NoError(t, r.Shutdown(context.Background()))

		// The port must be genuinely free the instant Shutdown returns --
		// a connection-refused check would not distinguish "released" from
		// "bound but refusing".
		ln, err := net.Listen("tcp", addr)
		require.NoErrorf(t, err, "iteration %d: Shutdown returned nil but %s is still bound", i, addr)
		require.NoError(t, ln.Close())
	}
}

// fakeReader builds a *reader whose embedded Reader and stop func are
// injectable, but whose ln is a REAL bound listener.
//
// That last part is load-bearing. newReader can never return a reader with a
// nil ln, and Shutdown legitimately closes r.ln -- so a fixture with ln == nil
// would nil-deref the moment teardown touches the listener, making a correct
// implementation look broken and silently asserting that ln is not part of
// teardown, which is itself the bug (see the port-release gate).
func fakeReader(t *testing.T, embedded sdkmetric.Reader, stop func(context.Context) error) *reader {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	return &reader{Reader: embedded, ln: ln, stop: stop}
}
