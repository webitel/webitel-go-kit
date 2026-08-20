package prometheus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/otlptranslator"
	"go.opentelemetry.io/otel"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const (
	// metricsPath is the endpoint the OTel Prometheus spec requires.
	metricsPath = "/metrics"
	// defaultHost and defaultPort are the OTel spec defaults for
	// OTEL_EXPORTER_PROMETHEUS_HOST / OTEL_EXPORTER_PROMETHEUS_PORT.
	defaultHost = "localhost"
	defaultPort = "9464"
)

// discardLog is the sink for per-scrape HTTP noise (http.Server.ErrorLog and
// promhttp's ErrorLog): a library must not write to stderr uninvited, and the
// registry calls Options with a fixed signature that leaves nowhere to inject
// a logger. Those two are not lost telemetry -- promhttp already surfaces them
// as promhttp_metric_handler_errors_total on the endpoint itself.
//
// Anything that means the endpoint is DEAD goes to otel.Handle instead, which
// sdk/otel.go wires to slog. See newReader's serve goroutine.
func discardLog() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// reader wraps an sdkmetric.Reader with the lifetime of the HTTP listener
// that serves it. Embedding the interface promotes the SDK's unexported
// Reader methods, so *reader satisfies sdkmetric.Reader without
// reimplementing anything.
type reader struct {
	sdkmetric.Reader
	ln   net.Listener
	stop func(context.Context) error

	once    sync.Once
	shutErr error // result of the first Shutdown; returned by every later call
}

// addr reports the bound address. It is the test seam: Options returns only
// []metric.Option and cannot expose the port a caller asked for as "0".
func (r *reader) addr() string {
	return r.ln.Addr().String()
}

// Shutdown tears the HTTP listener down FIRST, then the embedded Reader.
//
// On a shut-down Reader, the Prometheus exporter's Collect returns having
// sent nothing and without an error, so a Reader-first order would yield an
// HTTP 200 with an empty body: Prometheus records that as a successful
// scrape and keeps `up=1` while the process is dying. Listener-first refuses
// the connection instead, so `up` correctly goes to 0.
func (r *reader) Shutdown(ctx context.Context) error {
	r.once.Do(func() {
		srvErr := r.stop(ctx)            // listener first: refuse connections
		rdrErr := r.Reader.Shutdown(ctx) // then the reader
		r.shutErr = errors.Join(srvErr, rdrErr)
	})
	return r.shutErr
}

// newReader validates host/port, binds the listener synchronously, and
// starts serving /metrics in the background. It is the seam tests drive
// directly, since Options can only return []metric.Option.
func newReader(ctx context.Context, host, port string) (*reader, error) {
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return nil, fmt.Errorf(
			"otel/metric/prometheus: OTEL_EXPORTER_PROMETHEUS_PORT %q: want 0..65535 (default %s): %w",
			port, defaultPort, err,
		)
	}

	// net.JoinHostPort is mandatory, not cosmetic: it brackets IPv6 literals
	// so "::1" becomes "[::1]:9464", which net.Listen accepts. The host
	// itself is not validated here; net.Listen is the validator and its
	// bind error already reports precisely.
	addr := net.JoinHostPort(host, port)

	log := discardLog()

	// A private registry, never client_golang's process-global
	// DefaultRegisterer: the global would make a second Configure fail with
	// an already-registered error, and it is exactly the package-level
	// state this design rejects.
	reg := promclient.NewRegistry()

	exporter, err := promexporter.New(
		promexporter.WithRegisterer(reg),
		// The upstream default translation strategy is conditional on
		// prometheus/common's NameValidationScheme and is documented as
		// changing in a future release; passing it explicitly pins today's
		// behaviour. WTEL-10157 decides the OTel-native metric names, this
		// is where they are translated to Prometheus form.
		promexporter.WithTranslationStrategy(otlptranslator.UnderscoreEscapingWithSuffixes),
	)
	if exporter == nil {
		return nil, fmt.Errorf("otel/metric/prometheus: new exporter: %w", err)
	}
	if err != nil {
		// Upstream returns a fully usable *Exporter alongside a non-nil error
		// when only its own self-observability instruments failed to build
		// (gated on OTEL_GO_X_OBSERVABILITY). Aborting here would fail
		// Options -> metric.NewOptions -> Configure, taking LOGS AND TRACES
		// down over degraded exporter-internal telemetry -- reintroducing the
		// exact failure this ticket exists to remove.
		otel.Handle(fmt.Errorf("otel/metric/prometheus: exporter self-observability degraded: %w", err))
	}

	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		// One bad metric must not 500 the whole scrape; an all-failing
		// gather still falls through to a 500 on its own.
		ErrorHandling: promhttp.ContinueOnError,
		ErrorLog:      slog.NewLogLogger(log.Handler(), slog.LevelWarn),
		// Handler errors become a scrapable counter
		// (promhttp_metric_handler_errors_total) rather than a log line.
		Registry: reg,
		// The exporter emits exemplars that the classic text format
		// silently drops; the repo declares zero metrics today, so there
		// is no series identity to break by enabling this now.
		EnableOpenMetrics:   true,
		MaxRequestsInFlight: 4,
		// Strictly BELOW the http.Server WriteTimeout below. promhttp wraps
		// the handler in http.TimeoutHandler, which writes a 503 on expiry --
		// but the write deadline is armed when the request is read, so equal
		// values race and the scraper sees a reset instead of the diagnostic.
		Timeout: 8 * time.Second,
	})

	mux := http.NewServeMux()
	mux.Handle(metricsPath, handler)

	srv := &http.Server{
		Handler: mux,
		// Timeouts and MaxHeaderBytes copied verbatim from
		// infra/health/http/server.go.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 13,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}

	// Listen synchronously so a bind error is returned, not just logged.
	// ListenConfig rather than net.Listen so a cancelled Configure stops
	// acquiring OS resources instead of binding anyway.
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("otel/metric/prometheus: listen %s: %w", addr, err)
	}

	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			// NOT the discard logger. This is the only signal a permanently
			// dead scrape endpoint ever produces: the goroutine exits, `up`
			// goes 0 for the whole service, and nothing else reports why.
			// otel.Handle is the module's sink, wired to slog by Configure.
			otel.Handle(fmt.Errorf("otel/metric/prometheus: serve %s: %w", ln.Addr(), serveErr))
		}
	}()

	return &reader{
		Reader: exporter,
		ln:     ln,
		stop: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			err := srv.Shutdown(ctx)
			if err != nil {
				// The graceful drain did not finish. Do not leave in-flight
				// connections (reclaimed only when Write/IdleTimeout fires)
				// and the listener behind just because ctx expired.
				err = errors.Join(err, srv.Close())
			}
			// http.Server.Shutdown closes only the listeners Serve has already
			// tracked, and Serve runs on the goroutine started above. A
			// Shutdown that wins that race returns nil with the port still
			// bound -- so close the listener directly rather than relying on
			// the goroutine having been scheduled. Serve's own deferred close
			// makes the second close a no-op; net.ErrClosed is that no-op and
			// is not a failure.
			if closeErr := ln.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
				err = errors.Join(err, closeErr)
			}
			return err
		},
	}, nil
}
