package otelsdk_test

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	otelog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	otelsdk "github.com/webitel/webitel-go-kit/infra/otel/sdk"
)

func TestConfigureDefaultLogsExporterIsStdout(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "")
	t.Setenv("OTEL_LOGRECORD_CODEC", "text")
	stdout := captureFile(t, &os.Stdout)

	shutdown, err := otelsdk.Configure(context.Background())
	require.NoError(t, err)
	require.IsType(t, &sdklog.LoggerProvider{}, global.GetLoggerProvider())

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

func TestConfigureLogsExporterNone(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "none")
	before := global.GetLoggerProvider()

	shutdown, err := otelsdk.Configure(context.Background())
	require.NoError(t, err)
	require.NoError(t, shutdown(context.Background()))

	require.Same(t, before, global.GetLoggerProvider())
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
	rec.SetBody(otelog.StringValue(marker))
	global.GetLoggerProvider().Logger("test").Emit(context.Background(), rec)
}
