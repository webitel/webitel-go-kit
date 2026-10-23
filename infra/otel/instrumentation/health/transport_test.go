package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/webitel/webitel-go-kit/infra/health"
	"github.com/webitel/webitel-go-kit/infra/otel/semconv/v0.4.0/healthconv"
)

func TestTransportFollowsRegistry(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { require.NoError(t, mp.Shutdown(context.Background())) })

	reg := health.New(health.Config{
		Interval:      5 * time.Millisecond,
		Timeout:       2 * time.Millisecond,
		FailThreshold: 1,
		MinUnready:    time.Millisecond,
		StaleAfter:    time.Second,
		DrainHold:     time.Millisecond,
	}, nil, health.WithTransport(NewTransport(WithMeterProvider(mp))))
	reg.Critical("live", func(context.Context) error { return nil })

	require.Empty(t, collectScope(t, reader), "metrics registered before Start")

	require.NoError(t, reg.Start(context.Background()))

	require.Eventually(t, func() bool {
		_, ok := collectScope(t, reader)[healthconv.StatusObservable{}.Name()]

		return ok
	}, 3*time.Second, 5*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, reg.Shutdown(ctx))

	require.Empty(t, collectScope(t, reader), "metrics still registered after Stop")
}

func TestTransportStopNeverStarted(t *testing.T) {
	require.NoError(t, NewTransport().Stop(context.Background()))

	var nilTransport *Transport
	require.NoError(t, nilTransport.Start(context.Background(), nil))
	require.NoError(t, nilTransport.Stop(context.Background()))
}

func TestTransportStartTwice(t *testing.T) {
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewManualReader()))
	t.Cleanup(func() { require.NoError(t, mp.Shutdown(context.Background())) })

	reg := health.New(health.DefaultConfig(), nil)
	tr := NewTransport(WithMeterProvider(mp))

	require.NoError(t, tr.Start(context.Background(), reg))
	require.Error(t, tr.Start(context.Background(), reg))

	require.NoError(t, tr.Stop(context.Background()))
	require.NoError(t, tr.Stop(context.Background()))
}
