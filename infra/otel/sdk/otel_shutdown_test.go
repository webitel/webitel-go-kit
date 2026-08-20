package otelsdk_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	otelsdk "github.com/webitel/webitel-go-kit/infra/otel/sdk"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/prometheus"
)

// TestConfigureErrorReleasesMetricReaders guards a resource leak, not a
// cosmetic error path.
//
// An exporter's option constructor may acquire an OS resource while options
// are still being built -- the prometheus metrics exporter binds its listener
// there, because that is the only place a bind error can still be reported and
// returned. Configure's early error return happens BEFORE the MeterProvider is
// constructed, so without an explicit owner nothing ever closes that listener.
//
// The trigger is the nasty part: ANY signal's bad exporter lands on that path.
// Here it is OTEL_LOGS_EXPORTER that is misconfigured, yet it is the METRICS
// port that would be stranded -- for the life of the process, while the
// returned ShutdownFunc reports success.
func TestConfigureErrorReleasesMetricReaders(t *testing.T) {
	// A free port, released immediately so Configure can claim it.
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := probe.Addr().String()
	host, port, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	require.NoError(t, probe.Close())

	t.Setenv("OTEL_METRICS_EXPORTER", "prometheus")
	t.Setenv("OTEL_EXPORTER_PROMETHEUS_HOST", host)
	t.Setenv("OTEL_EXPORTER_PROMETHEUS_PORT", port)
	// A DIFFERENT signal is broken. This is what aborts Configure.
	t.Setenv("OTEL_LOGS_EXPORTER", "bogus-scheme-that-is-not-registered")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	shutdown, err := otelsdk.Configure(ctx)
	require.Error(t, err, "a bad logs exporter must still fail Configure")
	require.NotNil(t, shutdown, "Configure must return a usable ShutdownFunc even on error")

	require.NoError(t, shutdown(ctx))

	// The metrics listener must be gone. Rebinding is the proof: a
	// connection-refused check cannot tell "released" from "bound but
	// refusing".
	ln, err := net.Listen("tcp", addr)
	require.NoErrorf(t, err, "%s still bound after a failed Configure + shutdown()", addr)
	require.NoError(t, ln.Close())
}
