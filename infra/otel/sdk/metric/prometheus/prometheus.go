// Package prometheus registers the "prometheus" scheme for
// go.opentelemetry.io/otel/sdk/metric, serving /metrics on its own listener
// bound from OTEL_EXPORTER_PROMETHEUS_HOST/_PORT (defaults localhost:9464).
//
// Import it for its side effect, as with the other exporter plugins:
//
//	import _ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/prometheus"
//
// The listener's lifetime is tied to the metric SDK: it is stopped when the
// MeterProvider shuts down. Nothing needs mounting in a caller's router.
package prometheus

import (
	"context"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/webitel/webitel-go-kit/infra/otel/internal"
	"github.com/webitel/webitel-go-kit/infra/otel/sdk/metric"
)

// resolveHostPort reads OTEL_EXPORTER_PROMETHEUS_HOST/_PORT, seeded with the
// spec defaults. internal.EnvString's callback never fires for an unset or
// whitespace-only variable, so the defaults survive untouched. Split out
// from Options so tests can assert the env-resolution behaviour without
// binding a socket.
func resolveHostPort() (host, port string) {
	host, port = defaultHost, defaultPort
	internal.Environment.Apply(
		// The empty guard is load-bearing, not defensive noise.
		// internal.EnvString applies its own non-empty test BEFORE unquoting,
		// so a quoted-empty value -- OTEL_EXPORTER_PROMETHEUS_HOST='""', which
		// is what a compose file or k8s manifest yields for a set-but-blank
		// variable -- passes that test and arrives here as "". An empty host
		// makes net.JoinHostPort produce ":9464", which binds every interface
		// and publishes an unauthenticated /metrics (plus every resource
		// attribute in target_info) off-host, contradicting the loopback
		// default this package documents. Treat blank as unset.
		internal.EnvString("EXPORTER_PROMETHEUS_HOST", func(s string) {
			if s != "" {
				host = s
			}
		}),
		// Blank is likewise unset. A malformed value is NOT: it still reaches
		// newReader and is rejected there, so a typo'd port can never silently
		// bind the default.
		internal.EnvString("EXPORTER_PROMETHEUS_PORT", func(s string) {
			if s != "" {
				port = s
			}
		}),
	)
	return host, port
}

// Options builds the prometheus metric.Option set. Configuration is env-only
// by design (OTEL_EXPORTER_PROMETHEUS_HOST/_PORT); the DSN body carries
// nothing and rawDSN is ignored past the scheme.
func Options(ctx context.Context, rawDSN string) ([]metric.Option, error) {
	host, port := resolveHostPort()

	r, err := newReader(ctx, host, port)
	if err != nil {
		return nil, err
	}

	// Attach the wrapper directly, never the bare *promexporter.Exporter:
	// only the wrapper carries the listener's lifetime. No periodic reader
	// -- the Prometheus exporter is pull-based, unlike otlp and stdout.
	return []metric.Option{sdkmetric.WithReader(r)}, nil
}

func init() {
	metric.Register("prometheus", Options)
}
