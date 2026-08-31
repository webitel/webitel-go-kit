package otlp_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/webitel/webitel-go-kit/infra/otel/sdk/metric"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/otlp"
)

func TestSchemesResolve(t *testing.T) {
	for _, dsn := range []string{"otlphttp", "otlpgrpc"} {
		opts, err := metric.NewOptions(context.Background(), dsn)
		require.NoError(t, err, dsn)

		// No collector is listening; stop the reader without waiting on retries.
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		_ = sdkmetric.NewMeterProvider(opts...).Shutdown(ctx)
		cancel()
	}
}

func TestHTTPExporterWrites(t *testing.T) {
	var requests atomic.Int32
	var contentType atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.URL.Path == "/v1/metrics" && len(body) > 0 {
			requests.Add(1)
			contentType.Store(r.Header.Get("Content-Type"))
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", srv.URL+"/v1/metrics")

	ctx := context.Background()
	mp, err := metric.NewProvider(ctx, "otlphttp")
	require.NoError(t, err)
	counter, err := mp.Meter("test").Int64Counter("requests")
	require.NoError(t, err)
	counter.Add(ctx, 1)
	require.NoError(t, mp.Shutdown(ctx))

	require.EqualValues(t, 1, requests.Load())
	require.Equal(t, "application/x-protobuf", contentType.Load())
}
