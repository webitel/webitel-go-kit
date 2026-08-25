package otelsdk_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	otelsdk "github.com/webitel/webitel-go-kit/infra/otel/sdk"
)

// runtimeMetricNames is the set contrib v0.68.0 emits on every platform,
// asserted as a contract so a contrib bump that adds, drops or renames a
// metric fails here instead of silently reshaping every dashboard built on it.
//
// go.memory.limit is deliberately absent: upstream observes it only when a
// memory limit is set, so it does not appear on a normal machine. Do not try
// to force it with t.Setenv("GOMEMLIMIT", ...) -- the Go runtime reads that at
// process start, so only debug.SetMemoryLimit would work.
var runtimeMetricNames = []string{
	"go.memory.used",
	"go.memory.allocated",
	"go.memory.allocations",
	"go.memory.gc.goal",
	"go.goroutine.count",
	"go.processor.limit",
	"go.config.gogc",
}

// clearRuntimeEnv stops an ambient environment deciding these outcomes:
// a runner that exports OTEL_METRICS_RUNTIME=true would turn every
// off-by-default assertion into a false failure. EnvString treats an empty
// value as unset, so this is equivalent to unsetting.
//
// The deprecated-dictionary variable is cleared for the same reason: we never
// set it (AC-7), but an ambient value emits process.runtime.go.* names and
// would break the namespace assertion.
func clearRuntimeEnv(t *testing.T) {
	t.Helper()
	t.Setenv("OTEL_METRICS_RUNTIME", "")
	t.Setenv("OTEL_GO_X_DEPRECATED_RUNTIME_METRICS", "")
}

// collectRuntime builds a fresh ManualReader, runs Configure with that reader
// prepended to the caller's options, tears down under a bounded context in
// t.Cleanup (t.Errorf, never t.Fatalf -- Cleanup runs after the test body has
// already decided pass/fail), and returns the names of every metric reported
// under the contrib runtime instrumentation scope.
//
// If Collect returns metric.ErrReaderNotRegistered, that IS the no-op
// assertion: Configure took an early return (OTEL_SDK_DISABLED) or built no
// MeterProvider at all (no metrics exporter configured), so the reader was
// never registered. Callers rely on collectRuntime returning nil names in
// that case rather than failing the test.
func collectRuntime(t *testing.T, opts ...otelsdk.Option) []string {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	all := append([]otelsdk.Option{otelsdk.WithMetricOptions(sdkmetric.WithReader(reader))}, opts...)

	shutdown, err := otelsdk.Configure(context.Background(), all...)
	require.NoError(t, err)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			t.Errorf("Shutdown: %v", err)
		}
	})

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		require.ErrorIs(t, err, sdkmetric.ErrReaderNotRegistered)
		return nil
	}

	var names []string
	for _, sm := range rm.ScopeMetrics {
		if sm.Scope.Name != otelruntime.ScopeName {
			continue
		}
		for _, m := range sm.Metrics {
			names = append(names, m.Name)
		}
	}
	return names
}

// TestRuntimeMetricsOffByDefault guards AC-1: no option and no env must yield
// zero metrics under the runtime scope. Wiring a new contrib module must never
// change behaviour for an existing consumer that does not opt in.
func TestRuntimeMetricsOffByDefault(t *testing.T) {
	clearRuntimeEnv(t)

	names := collectRuntime(t)
	require.Empty(t, names)
}

// TestRuntimeMetricsEnabledByOption guards AC-2: WithRuntimeMetrics(true)
// turns the set on. Asserts the full contract rather than a single probe --
// see runtimeMetricNames for why go.memory.limit is excluded.
func TestRuntimeMetricsEnabledByOption(t *testing.T) {
	clearRuntimeEnv(t)

	names := collectRuntime(t, otelsdk.WithRuntimeMetrics(true))
	require.Subset(t, names, runtimeMetricNames)
	// Nothing may escape the go. namespace: these names come hardcoded from
	// upstream contrib, and this instrumentation must never squat in the
	// webitel.* registry that WTEL-10157 defines.
	for _, name := range names {
		require.Truef(t, strings.HasPrefix(name, "go."),
			"unexpected metric name %q outside the go. namespace", name)
	}
}

// TestRuntimeMetricsEnabledByEnv guards AC-3: OTEL_METRICS_RUNTIME=true must
// yield the identical metric set with no code-side option at all -- the
// environment variable is a first-class, standalone way to opt in.
func TestRuntimeMetricsEnabledByEnv(t *testing.T) {
	clearRuntimeEnv(t)
	t.Setenv("OTEL_METRICS_RUNTIME", "true")

	names := collectRuntime(t)
	require.Subset(t, names, runtimeMetricNames)
}

// TestRuntimeMetricsOptionOverridesEnv guards AC-4: newOptions applies
// environment configuration first and explicit Options last, so
// WithRuntimeMetrics(false) must win over OTEL_METRICS_RUNTIME=true. This is
// the existing env-first/options-last precedent applied to a new flag, not a
// special case for it.
func TestRuntimeMetricsOptionOverridesEnv(t *testing.T) {
	clearRuntimeEnv(t)
	t.Setenv("OTEL_METRICS_RUNTIME", "true")

	names := collectRuntime(t, otelsdk.WithRuntimeMetrics(false))
	require.Empty(t, names)
}

// TestRuntimeMetricsNoopWithoutMetricsExporter guards AC-5: with no metrics
// exporter configured, the existing `if len(setup.Metrics) > 0` guard must
// stop the instrumentation ever starting. That guard IS the acceptance
// criterion; this test is its proof, not a second guard.
//
// The decoy provider is the whole point. An earlier version collected from a
// reader it never passed to Configure and asserted ErrReaderNotRegistered --
// which an unregistered reader returns unconditionally, whatever the code
// does, so hoisting otelruntime.Start out of the metrics guard left it green.
// Installing a global provider we control gives a violation somewhere visible
// to land.
func TestRuntimeMetricsNoopWithoutMetricsExporter(t *testing.T) {
	clearRuntimeEnv(t)

	decoy := sdkmetric.NewManualReader()
	decoyProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(decoy))
	otel.SetMeterProvider(decoyProvider)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := decoyProvider.Shutdown(ctx); err != nil {
			t.Errorf("decoy Shutdown: %v", err)
		}
	})

	// No WithMetricOptions at all, so setup.Metrics stays empty.
	shutdown, err := otelsdk.Configure(context.Background(), otelsdk.WithRuntimeMetrics(true))
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			t.Errorf("Shutdown: %v", err)
		}
	})

	var rm metricdata.ResourceMetrics
	require.NoError(t, decoy.Collect(context.Background(), &rm))
	for _, sm := range rm.ScopeMetrics {
		require.NotEqualf(t, otelruntime.ScopeName, sm.Scope.Name,
			"runtime metrics started with no metrics exporter configured")
	}
}

// TestRuntimeMetricsNoopWhenSDKDisabled guards AC-6: OTEL_SDK_DISABLED=true
// must suppress runtime metrics with zero new guard code -- Configure's
// existing early return, which runs before anything is built, already
// satisfies this.
func TestRuntimeMetricsNoopWhenSDKDisabled(t *testing.T) {
	clearRuntimeEnv(t)
	t.Setenv("OTEL_SDK_DISABLED", "true")

	names := collectRuntime(t, otelsdk.WithRuntimeMetrics(true))
	require.Empty(t, names)
}

// TestRuntimeMetricsInvalidEnvValueIsOff is the regression test for the
// sharpest trap in this change: a plausible mistake like "yes" instead of
// "true" must leave runtime metrics off AND must never make Configure fail,
// which would take logs and traces down with it. See the callback comment at
// the METRICS_RUNTIME env row in otel.go for why routing that parse error
// through otel.Handle would do exactly that.
//
// require.NoError inside collectRuntime is the load-bearing assertion here --
// the empty name set alone would also hold if Configure had errored.
func TestRuntimeMetricsInvalidEnvValueIsOff(t *testing.T) {
	clearRuntimeEnv(t)
	t.Setenv("OTEL_METRICS_RUNTIME", "yes")

	names := collectRuntime(t)
	require.Empty(t, names)
}
