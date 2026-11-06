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

func TestTransportLifecycle(t *testing.T) {
	type step struct {
		start   bool // Start, otherwise Stop
		wantErr bool
		metrics bool // metrics registered after the step
	}

	var (
		start      = step{start: true, metrics: true}
		startAgain = step{start: true, wantErr: true, metrics: true}
		stop       = step{}
	)

	tests := []struct {
		name         string
		nilTransport bool
		steps        []step
	}{
		{name: "stop never started", steps: []step{stop}},
		{name: "start then stop", steps: []step{start, stop}},
		{name: "start twice", steps: []step{start, startAgain, stop}},
		{name: "stop twice", steps: []step{start, stop, stop}},
		{name: "start after stop", steps: []step{start, stop, start, stop}},
		{name: "nil transport", nilTransport: true, steps: []step{{start: true}, stop}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			t.Cleanup(func() { require.NoError(t, mp.Shutdown(context.Background())) })

			reg := health.New(health.DefaultConfig(), nil)

			tr := NewTransport(WithMeterProvider(mp))
			if tt.nilTransport {
				tr = nil
			}

			for i, s := range tt.steps {
				var err error
				if s.start {
					err = tr.Start(context.Background(), reg)
				} else {
					err = tr.Stop(context.Background())
				}

				if s.wantErr {
					require.Error(t, err, "step %d", i)
				} else {
					require.NoError(t, err, "step %d", i)
				}

				require.Equal(t, s.metrics, len(collectScope(t, reader)) > 0, "metrics registered after step %d", i)
			}
		})
	}
}
