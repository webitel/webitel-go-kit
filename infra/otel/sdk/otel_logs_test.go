package otelsdk_test

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	otelsdk "github.com/webitel/webitel-go-kit/infra/otel/sdk"
	"github.com/webitel/webitel-go-kit/infra/otel/sdk/log/throttle"
)

func TestConfigureDefaultLogsExporterIsStdout(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "")
	t.Setenv("OTEL_LOGRECORD_CODEC", "text")
	stdout := captureFile(t, &os.Stdout)

	shutdown, err := otelsdk.Configure(context.Background())
	require.NoError(t, err)
	require.IsType(t, &throttle.LoggerProvider{}, otel.GetLoggerProvider())

	emitMarker("default-exporter-marker")
	require.NoError(t, shutdown(context.Background()))

	require.Contains(t, stdout(), "default-exporter-marker")
}

func TestConfigureLogsExporterList(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "stdout:, stderr:")
	t.Setenv("OTEL_LOGRECORD_CODEC", "text")
	stdout := captureFile(t, &os.Stdout)
	stderr := captureFile(t, &os.Stderr)

	shutdown, err := otelsdk.Configure(context.Background())
	require.NoError(t, err)

	emitMarker("exporter-list-marker")
	require.NoError(t, shutdown(context.Background()))

	require.Contains(t, stdout(), "exporter-list-marker")
	require.Contains(t, stderr(), "exporter-list-marker")
}

func TestConfigureLogsThrottle(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "stdout:")
	t.Setenv("OTEL_LOGRECORD_CODEC", "text")
	t.Setenv("OTEL_LOGS_THROTTLE", "first=2,every=100")
	stdout := captureFile(t, &os.Stdout)

	shutdown, err := otelsdk.Configure(context.Background())
	require.NoError(t, err)
	require.IsType(t, &throttle.LoggerProvider{}, otel.GetLoggerProvider())

	for range 5 {
		emitMarker("throttle-marker")
	}
	require.NoError(t, shutdown(context.Background()))

	require.Equal(t, 2, strings.Count(stdout(), "throttle-marker"))
}

func TestConfigureLogsThrottleOff(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "stdout:")
	t.Setenv("OTEL_LOGS_THROTTLE", "off")

	shutdown, err := otelsdk.Configure(context.Background())
	require.NoError(t, err)
	require.IsType(t, &sdklog.LoggerProvider{}, otel.GetLoggerProvider())
	require.NoError(t, shutdown(context.Background()))
}

func TestConfigureLogsThrottleOption(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "stdout:")
	t.Setenv("OTEL_LOGS_THROTTLE", "first=2")

	shutdown, err := otelsdk.Configure(context.Background(), otelsdk.WithLogThrottle(nil))
	require.NoError(t, err)
	require.IsType(t, &sdklog.LoggerProvider{}, otel.GetLoggerProvider())
	require.NoError(t, shutdown(context.Background()))
}

func TestConfigureLogsExporterNone(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "none")
	before := otel.GetLoggerProvider()

	shutdown, err := otelsdk.Configure(context.Background())
	require.NoError(t, err)
	require.NoError(t, shutdown(context.Background()))

	require.Same(t, before, otel.GetLoggerProvider())
}

func TestConfigureLogsExporterNoneMixed(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "none,stdout:")

	shutdown, err := otelsdk.Configure(context.Background())
	require.Error(t, err)
	require.NoError(t, shutdown(context.Background()))
}

func captureFile(t *testing.T, target **os.File) func() string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	orig := *target
	*target = w
	t.Cleanup(func() { *target = orig })
	return func() string {
		*target = orig
		require.NoError(t, w.Close())
		out, err := io.ReadAll(r)
		require.NoError(t, err)
		return string(out)
	}
}

func emitMarker(marker string) {
	var rec otelog.Record
	rec.SetSeverity(otelog.SeverityInfo)
	rec.SetBody(attribute.StringValue(marker))
	otel.GetLoggerProvider().Logger("test").Emit(context.Background(), rec)
}
