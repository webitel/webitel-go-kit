package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/webitel/webitel-go-kit/infra/health"
	"github.com/webitel/webitel-go-kit/infra/otel/semconv"
	"github.com/webitel/webitel-go-kit/infra/otel/semconv/webitelconv"
)

type fakeSnapshotter struct {
	snap health.Snapshot
}

func (f fakeSnapshotter) Snapshot() health.Snapshot { return f.snap }

// collectScope runs one collection and returns this package's metrics by name.
func collectScope(t *testing.T, reader *sdkmetric.ManualReader) map[string]metricdata.Metrics {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	byName := map[string]metricdata.Metrics{}
	for _, sm := range rm.ScopeMetrics {
		if sm.Scope.Name != scopeName {
			continue
		}
		for _, m := range sm.Metrics {
			byName[m.Name] = m
		}
	}
	return byName
}

func TestCollect(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { require.NoError(t, mp.Shutdown(context.Background())) })

	now := time.Now()
	snap := fakeSnapshotter{snap: health.Snapshot{
		State: health.StateDegraded,
		Checks: []health.CheckResult{
			{Name: "loop", Group: health.GroupLiveness, Status: health.StatusOK, LastRun: now,
				LastDuration: time.Millisecond, Transitions: health.Transitions{OK: 1}},
			{Name: "postgres", Group: health.GroupCritical, Status: health.StatusOK, LastRun: now,
				LastDuration: 123 * time.Millisecond, Transitions: health.Transitions{OK: 1}},
			{Name: "cache", Group: health.GroupInformational, Status: health.StatusFail, LastRun: now,
				LastDuration: 45 * time.Millisecond, Transitions: health.Transitions{OK: 1, Fail: 2}},
			{Name: "never-ran", Group: health.GroupCritical, Status: health.StatusUnknown},
		},
	}}

	reg, err := Start(snap, WithMeterProvider(mp))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reg.Unregister()) })

	byName := collectScope(t, reader)
	require.Len(t, byName, 4)

	readyM, ok := byName[webitelconv.HealthReady{}.Name()].Data.(metricdata.Gauge[int64])
	require.True(t, ok)
	require.Len(t, readyM.DataPoints, 1)
	require.EqualValues(t, 1, readyM.DataPoints[0].Value)
	require.Equal(t, 0, readyM.DataPoints[0].Attributes.Len())

	stateM, ok := byName[webitelconv.HealthCheckState{}.Name()].Data.(metricdata.Gauge[int64])
	require.True(t, ok)
	stateByName := map[string]int64{}
	groupByName := map[string]string{}
	for _, dp := range stateM.DataPoints {
		name, _ := dp.Attributes.Value(semconv.WebitelHealthCheckNameKey)
		group, _ := dp.Attributes.Value(semconv.WebitelHealthCheckGroupKey)
		stateByName[name.AsString()] = dp.Value
		groupByName[name.AsString()] = group.AsString()
	}
	require.Equal(t, map[string]int64{"loop": 1, "postgres": 1, "cache": 0, "never-ran": 0}, stateByName)
	require.Equal(t, map[string]string{
		"loop": "liveness", "postgres": "critical", "cache": "informational", "never-ran": "critical",
	}, groupByName)

	trM, ok := byName[webitelconv.HealthCheckTransitions{}.Name()].Data.(metricdata.Sum[int64])
	require.True(t, ok)
	require.True(t, trM.IsMonotonic)
	require.Equal(t, metricdata.CumulativeTemporality, trM.Temporality)
	trByNameStatus := map[string]int64{}
	for _, dp := range trM.DataPoints {
		name, _ := dp.Attributes.Value(semconv.WebitelHealthCheckNameKey)
		status, _ := dp.Attributes.Value(semconv.WebitelHealthCheckStatusKey)
		trByNameStatus[name.AsString()+"/"+status.AsString()] = dp.Value
	}
	require.Equal(t, map[string]int64{
		"loop/ok": 1, "loop/fail": 0,
		"postgres/ok": 1, "postgres/fail": 0,
		"cache/ok": 1, "cache/fail": 2,
		"never-ran/ok": 0, "never-ran/fail": 0,
	}, trByNameStatus)

	durM, ok := byName[webitelconv.HealthCheckDuration{}.Name()].Data.(metricdata.Gauge[float64])
	require.True(t, ok)
	durByName := map[string]float64{}
	for _, dp := range durM.DataPoints {
		name, _ := dp.Attributes.Value(semconv.WebitelHealthCheckNameKey)
		durByName[name.AsString()] = dp.Value
	}
	require.Len(t, durByName, 3, "a never-run check has no duration series")
	require.InDelta(t, 0.123, durByName["postgres"], 0.0005)
	require.InDelta(t, 0.045, durByName["cache"], 0.0005)
}

func TestReadyReflectsEveryState(t *testing.T) {
	for _, tc := range []struct {
		state health.State
		want  int64
	}{
		{health.StateReady, 1},
		{health.StateDegraded, 1},
		{health.StateNotReady, 0},
		{health.StateUnknown, 0},
		{health.StateStopping, 0},
	} {
		t.Run(tc.state.String(), func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			t.Cleanup(func() { require.NoError(t, mp.Shutdown(context.Background())) })

			reg, err := Start(fakeSnapshotter{snap: health.Snapshot{State: tc.state}}, WithMeterProvider(mp))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, reg.Unregister()) })

			got, ok := collectScope(t, reader)[webitelconv.HealthReady{}.Name()].Data.(metricdata.Gauge[int64])
			require.True(t, ok)
			require.Len(t, got.DataPoints, 1)
			require.EqualValues(t, tc.want, got.DataPoints[0].Value)
		})
	}
}

func TestLiveRegistry(t *testing.T) {
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
	}, nil)
	reg.Critical("live", func(context.Context) error { return nil })

	require.NoError(t, reg.Start(context.Background()))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, reg.Stop(ctx))
	})

	handle, err := Start(reg, WithMeterProvider(mp))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, handle.Unregister()) })

	require.Eventually(t, func() bool {
		byName := collectScope(t, reader)

		readyM, ok := byName[webitelconv.HealthReady{}.Name()].Data.(metricdata.Gauge[int64])
		if !ok || len(readyM.DataPoints) != 1 || readyM.DataPoints[0].Value != 1 {
			return false
		}

		durM, ok := byName[webitelconv.HealthCheckDuration{}.Name()].Data.(metricdata.Gauge[float64])

		return ok && len(durM.DataPoints) == 1
	}, 3*time.Second, 5*time.Millisecond)
}
