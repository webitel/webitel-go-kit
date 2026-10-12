package logtest_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"

	"github.com/webitel/webitel-go-kit/pkg/logging/logtest"
)

type ctxKey struct{}

func TestRecorder_Logger(t *testing.T) {
	t.Run("scope", func(t *testing.T) {
		rec := logtest.NewRecorder()
		rec.Logger("svc.sub",
			log.WithInstrumentationVersion("v1.2.3"),
			log.WithSchemaURL("https://example.com/schema"),
			log.WithInstrumentationAttributes(attribute.String("k", "v")),
		).Emit(context.Background(), record("a"))

		recs := rec.Records()
		require.Len(t, recs, 1)
		assert.Equal(t, logtest.Scope{
			Name:       "svc.sub",
			Version:    "v1.2.3",
			SchemaURL:  "https://example.com/schema",
			Attributes: attribute.NewSet(attribute.String("k", "v")),
		}, recs[0].Scope)
	})

	t.Run("empty scope", func(t *testing.T) {
		rec := logtest.NewRecorder()
		rec.Logger("").Emit(context.Background(), record("a"))

		recs := rec.Records()
		require.Len(t, recs, 1)
		s := recs[0].Scope
		assert.Empty(t, s.Name)
		assert.Empty(t, s.Version)
		assert.Empty(t, s.SchemaURL)
		assert.Zero(t, s.Attributes.Len())
	})

	t.Run("shared records", func(t *testing.T) {
		rec := logtest.NewRecorder()
		rec.Logger("a").Emit(context.Background(), record("1"))
		rec.Logger("b").Emit(context.Background(), record("2"))

		recs := rec.Records()
		require.Len(t, recs, 2)
		assert.Equal(t, "a", recs[0].Scope.Name)
		assert.Equal(t, "b", recs[1].Scope.Name)
	})
}

func TestRecorder_Records(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		assert.Empty(t, logtest.NewRecorder().Records())
	})

	t.Run("order", func(t *testing.T) {
		rec := logtest.NewRecorder()
		l := rec.Logger("a")
		for _, b := range []string{"1", "2", "3"} {
			l.Emit(context.Background(), record(b))
		}
		assert.Equal(t, []string{"1", "2", "3"}, bodies(rec.Records()))
	})

	t.Run("context", func(t *testing.T) {
		rec := logtest.NewRecorder()
		ctx := context.WithValue(context.Background(), ctxKey{}, "v")
		rec.Logger("a").Emit(ctx, record("1"))

		recs := rec.Records()
		require.Len(t, recs, 1)
		assert.Equal(t, "v", recs[0].Context.Value(ctxKey{}))
	})

	t.Run("clone record", func(t *testing.T) {
		rec := logtest.NewRecorder()
		kvs := make([]attribute.KeyValue, 0, 10)
		for _, k := range []string{"a", "b", "c", "d", "e", "f"} {
			kvs = append(kvs, attribute.String(k, k))
		}
		r := record("1", kvs...)
		rec.Logger("a").Emit(context.Background(), r)
		r.AddAttributes(attribute.String("g", "g"))
		r.SetBody(attribute.StringValue("2"))

		recs := rec.Records()
		require.Len(t, recs, 1)
		assert.Equal(t, "1", recs[0].Body().AsString())
		assert.Equal(t, kvs, recs[0].Attributes())
	})

	t.Run("copy results records", func(t *testing.T) {
		rec := logtest.NewRecorder()
		rec.Logger("a").Emit(context.Background(), record("1"))

		got := rec.Records()
		got[0] = logtest.Record{}
		assert.Equal(t, []string{"1"}, bodies(rec.Records()))
	})

	t.Run("concurrency", func(t *testing.T) {
		rec := logtest.NewRecorder()
		l := rec.Logger("a")
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				for range 50 {
					l.Emit(context.Background(), record("x"))
					rec.Records()
				}
			})
		}
		wg.Wait()
		assert.Len(t, rec.Records(), 400)
	})
}

func TestRecorder_Reset(t *testing.T) {
	t.Run("clears", func(t *testing.T) {
		rec := logtest.NewRecorder()
		rec.Logger("a").Emit(context.Background(), record("1"))
		rec.Reset()
		assert.Empty(t, rec.Records())
	})

	t.Run("records after", func(t *testing.T) {
		rec := logtest.NewRecorder()
		l := rec.Logger("a")
		l.Emit(context.Background(), record("1"))
		rec.Reset()
		l.Emit(context.Background(), record("2"))
		assert.Equal(t, []string{"2"}, bodies(rec.Records()))
	})
}

func TestRecord_Attributes(t *testing.T) {
	t.Run("order", func(t *testing.T) {
		kvs := []attribute.KeyValue{attribute.Int("b", 2), attribute.Int("a", 1), attribute.Int("c", 3)}
		rec := logtest.NewRecorder()
		rec.Logger("a").Emit(context.Background(), record("1", kvs...))

		recs := rec.Records()
		require.Len(t, recs, 1)
		assert.Equal(t, kvs, recs[0].Attributes())
	})

	t.Run("empty", func(t *testing.T) {
		assert.Empty(t, logtest.Record{}.Attributes())
	})
}

func TestWithEnabled(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		l := logtest.NewRecorder().Logger("a")
		assert.True(t, l.Enabled(context.Background(), log.EnabledParameters{Severity: log.SeverityTrace1}))
	})

	t.Run("arguments", func(t *testing.T) {
		var gotScope logtest.Scope
		var gotParam log.EnabledParameters
		var gotCtx context.Context
		rec := logtest.NewRecorder(logtest.WithEnabled(func(ctx context.Context, s logtest.Scope, p log.EnabledParameters) bool {
			gotCtx, gotScope, gotParam = ctx, s, p
			return true
		}))
		ctx := context.WithValue(context.Background(), ctxKey{}, "v")
		param := log.EnabledParameters{Severity: log.SeverityWarn, EventName: "ev"}
		rec.Logger("a", log.WithInstrumentationVersion("v1")).Enabled(ctx, param)

		assert.Equal(t, "a", gotScope.Name)
		assert.Equal(t, "v1", gotScope.Version)
		assert.Equal(t, param, gotParam)
		assert.Equal(t, "v", gotCtx.Value(ctxKey{}))
	})

	t.Run("filter", func(t *testing.T) {
		rec := logtest.NewRecorder(logtest.WithEnabled(func(_ context.Context, s logtest.Scope, p log.EnabledParameters) bool {
			return s.Name == "a" && p.Severity >= log.SeverityWarn
		}))
		ctx := context.Background()
		assert.True(t, rec.Logger("a").Enabled(ctx, log.EnabledParameters{Severity: log.SeverityError}))
		assert.False(t, rec.Logger("a").Enabled(ctx, log.EnabledParameters{Severity: log.SeverityInfo}))
		assert.False(t, rec.Logger("b").Enabled(ctx, log.EnabledParameters{Severity: log.SeverityError}))
	})

	t.Run("emit unfiltered", func(t *testing.T) {
		rec := logtest.NewRecorder(logtest.WithEnabled(func(context.Context, logtest.Scope, log.EnabledParameters) bool {
			return false
		}))
		rec.Logger("a").Emit(context.Background(), record("1"))
		assert.Len(t, rec.Records(), 1)
	})
}

func record(body string, kvs ...attribute.KeyValue) log.Record {
	var r log.Record
	r.SetBody(attribute.StringValue(body))
	r.AddAttributes(kvs...)
	return r
}

func bodies(recs []logtest.Record) []string {
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = r.Body().AsString()
	}
	return out
}
