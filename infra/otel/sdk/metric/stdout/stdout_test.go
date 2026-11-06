package stdout_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/webitel/webitel-go-kit/infra/otel/sdk/metric"
	"github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/stdout"
)

// export records one measurement through dsn and shuts the provider down.
func export(t *testing.T, dsn string) {
	t.Helper()
	t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "3600000")

	ctx := context.Background()
	mp, err := metric.NewProvider(ctx, dsn)
	require.NoError(t, err)
	counter, err := mp.Meter("test").Int64Counter("requests")
	require.NoError(t, err)
	counter.Add(ctx, 1)
	require.NoError(t, mp.Shutdown(ctx))
}

func TestFileWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	export(t, "file:"+path+";max-size=1")

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(out), `"Name":"requests"`)
}

func TestStdoutWrites(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	prev := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = prev })

	export(t, "stdout")
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Contains(t, string(out), `"Name":"requests"`)
}

func TestOptionsErrors(t *testing.T) {
	for _, dsn := range []string{"file:", "file:/", "bogus"} {
		_, err := stdout.Options(context.Background(), dsn)
		require.Error(t, err, dsn)
	}
}
