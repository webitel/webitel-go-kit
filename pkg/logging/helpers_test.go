package logging_test

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/webitel/webitel-go-kit/pkg/logging"
	"github.com/webitel/webitel-go-kit/pkg/logging/logtest"
	"github.com/webitel/webitel-go-kit/pkg/logging/redact"
)

var testSalt = []byte("test-salt")

type ctxKey struct{}

type testEvent struct {
	name  string
	attrs []attribute.KeyValue
}

func (e testEvent) EventName() string                { return e.name }
func (e testEvent) Attributes() []attribute.KeyValue { return e.attrs }

func newLogger(opts ...logging.Option) (*logging.Logger, *logtest.Recorder) {
	rec := logtest.NewRecorder()
	base := []logging.Option{logging.WithProvider(rec), logging.WithLevel(logging.LevelTrace)}
	return logging.New(append(base, opts...)...), rec
}

func single(t *testing.T, rec *logtest.Recorder) *logtest.Record {
	t.Helper()
	recs := rec.Records()
	require.Len(t, recs, 1)
	return &recs[0]
}

func attrs(r *logtest.Record) map[string]any {
	return plainMap(r.Attributes())
}

func plainMap(kvs []attribute.KeyValue) map[string]any {
	out := make(map[string]any, len(kvs))
	for _, kv := range kvs {
		out[string(kv.Key)] = plain(kv.Value)
	}
	return out
}

func plain(v attribute.Value) any {
	switch v.Type() {
	case attribute.MAP:
		return plainMap(v.AsMap())
	case attribute.SLICE:
		items := v.AsSlice()
		out := make([]any, len(items))
		for i, item := range items {
			out[i] = plain(item)
		}
		return out
	default:
		return v.AsInterface()
	}
}

func pseudonym(s string) string {
	out, _ := redact.Render(redact.ClassPII, s, s == "")
	return out
}

func captureStderr(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)

	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	f()
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}
