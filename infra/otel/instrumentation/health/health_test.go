package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/webitel/webitel-go-kit/infra/health"
	semconv "github.com/webitel/webitel-go-kit/infra/otel/semconv/v0.2.0"
	"github.com/webitel/webitel-go-kit/infra/otel/semconv/v0.2.0/healthconv"
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

	require.Equal(t, map[string]int64{"ready": 0, "degraded": 1, "not_ready": 0}, nodeStatus(t, byName))

	stM, ok := byName[healthconv.CheckStatusObservable{}.Name()].Data.(metricdata.Sum[int64])
	require.True(t, ok)
	require.False(t, stM.IsMonotonic)
	stByNameState := map[string]int64{}
	groupByName := map[string]string{}
	for _, dp := range stM.DataPoints {
		name, _ := dp.Attributes.Value(semconv.WebitelHealthCheckNameKey)
		group, _ := dp.Attributes.Value(semconv.WebitelHealthCheckGroupKey)
		state, _ := dp.Attributes.Value(semconv.WebitelHealthCheckStateKey)
		stByNameState[name.AsString()+"/"+state.AsString()] = dp.Value
		groupByName[name.AsString()] = group.AsString()
	}
	require.Equal(t, map[string]int64{
		"loop/ok": 1, "loop/fail": 0, "loop/unknown": 0,
		"postgres/ok": 1, "postgres/fail": 0, "postgres/unknown": 0,
		"cache/ok": 0, "cache/fail": 1, "cache/unknown": 0,
		"never-ran/ok": 0, "never-ran/fail": 0, "never-ran/unknown": 1,
	}, stByNameState)
	require.Equal(t, map[string]string{
		"loop": "liveness", "postgres": "critical", "cache": "informational", "never-ran": "critical",
	}, groupByName)

	trM, ok := byName[healthconv.CheckTransitionsObservable{}.Name()].Data.(metricdata.Sum[int64])
	require.True(t, ok)
	require.True(t, trM.IsMonotonic)
	require.Equal(t, metricdata.CumulativeTemporality, trM.Temporality)
	trByNameStatus := map[string]int64{}
	for _, dp := range trM.DataPoints {
		name, _ := dp.Attributes.Value(semconv.WebitelHealthCheckNameKey)
		state, _ := dp.Attributes.Value(semconv.WebitelHealthCheckStateKey)
		trByNameStatus[name.AsString()+"/"+state.AsString()] = dp.Value
	}
	require.Equal(t, map[string]int64{
		"loop/ok": 1, "loop/fail": 0,
		"postgres/ok": 1, "postgres/fail": 0,
		"cache/ok": 1, "cache/fail": 2,
		"never-ran/ok": 0, "never-ran/fail": 0,
	}, trByNameStatus)

	durM, ok := byName[healthconv.CheckDurationObservable{}.Name()].Data.(metricdata.Gauge[float64])
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

// nodeStatus returns webitel.health.status by the webitel.health.state value.
func nodeStatus(t *testing.T, byName map[string]metricdata.Metrics) map[string]int64 {
	t.Helper()

	m, ok := byName[healthconv.StatusObservable{}.Name()].Data.(metricdata.Sum[int64])
	require.True(t, ok)
	require.False(t, m.IsMonotonic)

	byState := map[string]int64{}
	for _, dp := range m.DataPoints {
		require.Equal(t, 1, dp.Attributes.Len())
		state, _ := dp.Attributes.Value(semconv.WebitelHealthStateKey)
		byState[state.AsString()] = dp.Value
	}
	return byState
}

func TestStatusReflectsEveryState(t *testing.T) {
	for _, tc := range []struct {
		state health.State
		want  string
	}{
		{health.StateReady, "ready"},
		{health.StateDegraded, "degraded"},
		{health.StateNotReady, "not_ready"},
		{health.StateUnknown, "not_ready"},
		{health.StateStopping, "not_ready"},
	} {
		t.Run(tc.state.String(), func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			t.Cleanup(func() { require.NoError(t, mp.Shutdown(context.Background())) })

			reg, err := Start(fakeSnapshotter{snap: health.Snapshot{State: tc.state}}, WithMeterProvider(mp))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, reg.Unregister()) })

			want := map[string]int64{"ready": 0, "degraded": 0, "not_ready": 0}
			want[tc.want] = 1
			require.Equal(t, want, nodeStatus(t, collectScope(t, reader)))
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

		statusM, ok := byName[healthconv.StatusObservable{}.Name()].Data.(metricdata.Sum[int64])
		if !ok {
			return false
		}
		for _, dp := range statusM.DataPoints {
			state, _ := dp.Attributes.Value(semconv.WebitelHealthStateKey)
			if state.AsString() == "ready" && dp.Value != 1 {
				return false
			}
		}

		durM, ok := byName[healthconv.CheckDurationObservable{}.Name()].Data.(metricdata.Gauge[float64])

		return ok && len(durM.DataPoints) == 1
	}, 3*time.Second, 5*time.Millisecond)
}
