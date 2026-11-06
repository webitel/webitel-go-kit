package logging_test

import (
	"context"
	"errors"
	"log/slog"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"

	"github.com/webitel/webitel-go-kit/pkg/logging"
	"github.com/webitel/webitel-go-kit/pkg/logging/logtest"
)

type levelMethod func(*logging.Logger, context.Context, string, ...logging.Field)

func TestLogger_DebugCtx(t *testing.T) {
	testLevelMethod(t, (*logging.Logger).DebugCtx, logging.LevelDebug, log.SeverityDebug, "DEBUG")
}

func TestLogger_InfoCtx(t *testing.T) {
	testLevelMethod(t, (*logging.Logger).InfoCtx, logging.LevelInfo, log.SeverityInfo, "INFO")
}

func TestLogger_WarnCtx(t *testing.T) {
	testLevelMethod(t, (*logging.Logger).WarnCtx, logging.LevelWarn, log.SeverityWarn, "WARN")
}

func TestLogger_ErrorCtx(t *testing.T) {
	testLevelMethod(t, (*logging.Logger).ErrorCtx, logging.LevelError, log.SeverityError, "ERROR")
}

func TestLogger_Log(t *testing.T) {
	cases := map[string]struct {
		lvl  logging.Level
		sev  log.Severity
		text string
	}{
		"trace":  {logging.LevelTrace, log.SeverityTrace, "DEBUG-4"},
		"debug":  {logging.LevelDebug, log.SeverityDebug, "DEBUG"},
		"info":   {logging.LevelInfo, log.SeverityInfo, "INFO"},
		"offset": {logging.LevelInfo + 2, log.SeverityInfo3, "INFO+2"},
		"warn":   {logging.LevelWarn, log.SeverityWarn, "WARN"},
		"error":  {logging.LevelError, log.SeverityError, "ERROR"},
		"fatal":  {logging.LevelFatal, log.SeverityFatal, "ERROR+4"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			l, rec := newLogger()
			l.Log(context.Background(), tc.lvl, "hello", logging.Int("n", 1))

			r := single(t, rec)
			assert.Equal(t, tc.sev, r.Severity())
			assert.Equal(t, tc.text, r.SeverityText())
			assert.Equal(t, "hello", r.Body().AsString())
			assert.Equal(t, map[string]any{"n": int64(1)}, attrs(r))
		})
	}

	t.Run("level var", func(t *testing.T) {
		var lv slog.LevelVar
		lv.Set(slog.LevelWarn)
		l, rec := newLogger(logging.WithLevel(&lv))

		l.Log(context.Background(), logging.LevelInfo, "dropped")
		lv.Set(slog.LevelDebug)
		l.Log(context.Background(), logging.LevelInfo, "kept")

		assert.Equal(t, "kept", single(t, rec).Body().AsString())
	})

	t.Run("default level", func(t *testing.T) {
		rec := logtest.NewRecorder()
		l := logging.New(logging.WithProvider(rec))
		l.Log(context.Background(), logging.LevelDebug, "dropped")
		l.Log(context.Background(), logging.LevelInfo, "kept")
		assert.Equal(t, "kept", single(t, rec).Body().AsString())
	})
}

func TestLogger_Emit(t *testing.T) {
	ev := testEvent{name: "user.login", attrs: []attribute.KeyValue{attribute.String("user.id", "42")}}

	t.Run("record", func(t *testing.T) {
		l, rec := newLogger()
		l.Emit(context.Background(), ev, logging.Int("n", 1))

		r := single(t, rec)
		assert.Equal(t, "user.login", r.EventName())
		assert.Equal(t, log.SeverityInfo, r.Severity())
		assert.Equal(t, "INFO", r.SeverityText())
		assert.Equal(t, attribute.EMPTY, r.Body().Type())
		assert.False(t, r.Timestamp().IsZero())
		assert.Equal(t, map[string]any{"user.id": "42", "n": int64(1)}, attrs(r))
	})

	t.Run("nil event", func(t *testing.T) {
		l, rec := newLogger()
		assert.NotPanics(t, func() { l.Emit(context.Background(), nil, logging.Int("n", 1)) })

		r := single(t, rec)
		assert.Empty(t, r.EventName())
		assert.Equal(t, map[string]any{"n": int64(1)}, attrs(r))
	})

	t.Run("below level", func(t *testing.T) {
		l, rec := newLogger(logging.WithLevel(logging.LevelWarn))
		l.Emit(context.Background(), ev)
		assert.Empty(t, rec.Records())
	})

	t.Run("provider filter", func(t *testing.T) {
		rec := logtest.NewRecorder(logtest.WithEnabled(func(_ context.Context, _ logtest.Scope, p log.EnabledParameters) bool {
			return p.EventName != "user.login"
		}))
		l := logging.New(logging.WithProvider(rec))
		l.Emit(context.Background(), ev)
		l.Emit(context.Background(), testEvent{name: "user.logout"})

		assert.Equal(t, "user.logout", single(t, rec).EventName())
	})

	t.Run("group", func(t *testing.T) {
		l, rec := newLogger()
		l.With(logging.Str("a", "1")).WithGroup("g").With(logging.Str("b", "2")).
			Emit(context.Background(), ev, logging.Str("c", "3"))

		assert.Equal(t, map[string]any{
			"user.id": "42",
			"a":       "1",
			"g":       map[string]any{"b": "2", "c": "3"},
		}, attrs(single(t, rec)))
	})

	t.Run("context", func(t *testing.T) {
		l, rec := newLogger()
		l.Emit(context.WithValue(context.Background(), ctxKey{}, "v"), ev)
		assert.Equal(t, "v", single(t, rec).Context.Value(ctxKey{}))
	})

	t.Run("nil context", func(t *testing.T) {
		l, rec := newLogger()
		var ctx context.Context
		assert.NotPanics(t, func() { l.Emit(ctx, ev) })
		assert.Len(t, rec.Records(), 1)
	})

	t.Run("parity", func(t *testing.T) {
		err := errors.New("boom")
		l, rec := newLogger()
		l = l.With(logging.Str("a", "1")).WithGroup("g").With(logging.Int("b", 2))
		fields := []logging.Field{
			logging.Str("c", "3"),
			logging.Group("grp", logging.Bool("d", true)),
			logging.Any("e", []any{1, "x"}),
			logging.Err(err),
		}

		l.InfoCtx(context.Background(), "", fields...)
		l.Emit(context.Background(), testEvent{name: "ev"}, fields...)

		recs := rec.Records()
		require.Len(t, recs, 2)
		assert.Equal(t, attrs(&recs[0]), attrs(&recs[1]))
		assert.Equal(t, err, recs[0].Err())
		assert.Equal(t, err, recs[1].Err())
	})
}

func TestLogger_Enabled(t *testing.T) {
	t.Run("at level", func(t *testing.T) {
		l, _ := newLogger(logging.WithLevel(logging.LevelWarn))
		assert.True(t, l.Enabled(context.Background(), logging.LevelWarn))
		assert.True(t, l.Enabled(context.Background(), logging.LevelError))
	})

	t.Run("below level", func(t *testing.T) {
		l, _ := newLogger(logging.WithLevel(logging.LevelWarn))
		assert.False(t, l.Enabled(context.Background(), logging.LevelInfo))
	})

	t.Run("level var", func(t *testing.T) {
		var lv slog.LevelVar
		lv.Set(slog.LevelError)
		l, _ := newLogger(logging.WithLevel(&lv))
		require.False(t, l.Enabled(context.Background(), logging.LevelWarn))

		lv.Set(slog.LevelWarn)
		assert.True(t, l.Enabled(context.Background(), logging.LevelWarn))
	})

	t.Run("provider", func(t *testing.T) {
		rec := logtest.NewRecorder(logtest.WithEnabled(func(_ context.Context, _ logtest.Scope, p log.EnabledParameters) bool {
			return p.Severity >= log.SeverityError
		}))
		l := logging.New(logging.WithProvider(rec), logging.WithLevel(logging.LevelTrace))
		assert.False(t, l.Enabled(context.Background(), logging.LevelWarn))
		assert.True(t, l.Enabled(context.Background(), logging.LevelError))
	})

	t.Run("nil context", func(t *testing.T) {
		l, _ := newLogger()
		var ctx context.Context
		assert.True(t, l.Enabled(ctx, logging.LevelInfo))
	})
}

func TestLogger_EventEnabled(t *testing.T) {
	ev := testEvent{name: "user.login"}

	t.Run("enabled", func(t *testing.T) {
		l, _ := newLogger()
		assert.True(t, l.EventEnabled(context.Background(), ev))
	})

	t.Run("nil event", func(t *testing.T) {
		l, _ := newLogger()
		assert.False(t, l.EventEnabled(context.Background(), nil))
	})

	t.Run("below level", func(t *testing.T) {
		l, _ := newLogger(logging.WithLevel(logging.LevelWarn))
		assert.False(t, l.EventEnabled(context.Background(), ev))
	})

	t.Run("provider", func(t *testing.T) {
		var got log.EnabledParameters
		rec := logtest.NewRecorder(logtest.WithEnabled(func(_ context.Context, _ logtest.Scope, p log.EnabledParameters) bool {
			got = p
			return p.EventName != "user.login"
		}))
		l := logging.New(logging.WithProvider(rec))

		assert.False(t, l.EventEnabled(context.Background(), ev))
		assert.Equal(t, log.EnabledParameters{Severity: log.SeverityInfo, EventName: "user.login"}, got)
		assert.True(t, l.EventEnabled(context.Background(), testEvent{name: "user.logout"}))
	})

	t.Run("nil context", func(t *testing.T) {
		l, _ := newLogger()
		var ctx context.Context
		assert.True(t, l.EventEnabled(ctx, ev))
	})
}

func TestLogger_With(t *testing.T) {
	t.Run("fields", func(t *testing.T) {
		l, rec := newLogger()
		l.With(logging.Str("a", "1")).InfoCtx(context.Background(), "m", logging.Str("b", "2"))
		assert.Equal(t, map[string]any{"a": "1", "b": "2"}, attrs(single(t, rec)))
	})

	t.Run("empty", func(t *testing.T) {
		l, _ := newLogger()
		assert.Same(t, l, l.With())
	})

	t.Run("chained", func(t *testing.T) {
		l, rec := newLogger()
		l.With(logging.Str("a", "1")).With(logging.Str("b", "2")).InfoCtx(context.Background(), "m")
		assert.Equal(t, map[string]any{"a": "1", "b": "2"}, attrs(single(t, rec)))
	})

	t.Run("parent unchanged", func(t *testing.T) {
		l, rec := newLogger()
		_ = l.With(logging.Str("a", "1"))
		l.InfoCtx(context.Background(), "m")
		assert.Empty(t, attrs(single(t, rec)))
	})

	t.Run("siblings", func(t *testing.T) {
		l, rec := newLogger()
		parent := l.With(logging.Str("a", "1"), logging.Str("x", "0"))
		left := parent.With(logging.Str("b", "2"))
		right := parent.With(logging.Str("c", "3"))

		left.InfoCtx(context.Background(), "left")
		right.InfoCtx(context.Background(), "right")

		recs := rec.Records()
		require.Len(t, recs, 2)
		assert.Equal(t, map[string]any{"a": "1", "x": "0", "b": "2"}, attrs(&recs[0]))
		assert.Equal(t, map[string]any{"a": "1", "x": "0", "c": "3"}, attrs(&recs[1]))
	})

	t.Run("group siblings", func(t *testing.T) {
		l, rec := newLogger()
		parent := l.WithGroup("g").With(logging.Str("a", "1"))
		left := parent.With(logging.Str("b", "2"))
		right := parent.With(logging.Str("c", "3"))

		left.InfoCtx(context.Background(), "left")
		right.InfoCtx(context.Background(), "right")
		parent.InfoCtx(context.Background(), "parent")

		recs := rec.Records()
		require.Len(t, recs, 3)
		assert.Equal(t, map[string]any{"g": map[string]any{"a": "1", "b": "2"}}, attrs(&recs[0]))
		assert.Equal(t, map[string]any{"g": map[string]any{"a": "1", "c": "3"}}, attrs(&recs[1]))
		assert.Equal(t, map[string]any{"g": map[string]any{"a": "1"}}, attrs(&recs[2]))
	})

	t.Run("event", func(t *testing.T) {
		l, rec := newLogger()
		l.With(logging.Ev(testEvent{name: "ev"})).InfoCtx(context.Background(), "m")
		assert.Equal(t, "ev", single(t, rec).EventName())
	})
}

func TestLogger_WithGroup(t *testing.T) {
	cases := map[string]struct {
		build  func(*logging.Logger) *logging.Logger
		fields []logging.Field
		want   map[string]any
	}{
		"call fields": {
			build:  func(l *logging.Logger) *logging.Logger { return l.WithGroup("g") },
			fields: []logging.Field{logging.Str("a", "1")},
			want:   map[string]any{"g": map[string]any{"a": "1"}},
		},
		"nested": {
			build: func(l *logging.Logger) *logging.Logger {
				return l.With(logging.Str("a", "1")).WithGroup("g1").With(logging.Str("b", "2")).WithGroup("g2")
			},
			fields: []logging.Field{logging.Str("c", "3")},
			want: map[string]any{
				"a":  "1",
				"g1": map[string]any{"b": "2", "g2": map[string]any{"c": "3"}},
			},
		},
		"empty": {
			build: func(l *logging.Logger) *logging.Logger { return l.WithGroup("g") },
			want:  map[string]any{},
		},
		"empty after fields": {
			build: func(l *logging.Logger) *logging.Logger { return l.With(logging.Str("a", "1")).WithGroup("g") },
			want:  map[string]any{"a": "1"},
		},
		"empty middle": {
			build:  func(l *logging.Logger) *logging.Logger { return l.WithGroup("g1").WithGroup("g2") },
			fields: []logging.Field{logging.Str("c", "3")},
			want:   map[string]any{"g1": map[string]any{"g2": map[string]any{"c": "3"}}},
		},
		"empty inner": {
			build: func(l *logging.Logger) *logging.Logger {
				return l.WithGroup("g1").With(logging.Str("b", "2")).WithGroup("g2")
			},
			want: map[string]any{"g1": map[string]any{"b": "2"}},
		},
		"empty name": {
			build:  func(l *logging.Logger) *logging.Logger { return l.WithGroup("") },
			fields: []logging.Field{logging.Str("a", "1")},
			want:   map[string]any{"a": "1"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			l, rec := newLogger()
			tc.build(l).InfoCtx(context.Background(), "m", tc.fields...)
			assert.Equal(t, tc.want, attrs(single(t, rec)))
		})
	}

	t.Run("empty name same", func(t *testing.T) {
		l, _ := newLogger()
		assert.Same(t, l, l.WithGroup(""))
	})

	t.Run("parent unchanged", func(t *testing.T) {
		l, rec := newLogger()
		_ = l.WithGroup("g")
		l.InfoCtx(context.Background(), "m", logging.Str("a", "1"))
		assert.Equal(t, map[string]any{"a": "1"}, attrs(single(t, rec)))
	})

	t.Run("event root", func(t *testing.T) {
		l, rec := newLogger()
		ev := testEvent{name: "ev", attrs: []attribute.KeyValue{attribute.String("x", "0")}}
		l.WithGroup("g").Emit(context.Background(), ev, logging.Str("c", "3"))
		assert.Equal(t, map[string]any{"x": "0", "g": map[string]any{"c": "3"}}, attrs(single(t, rec)))
	})

	t.Run("deep", func(t *testing.T) {
		const depth = 64
		l, rec := newLogger()
		for range depth {
			l = l.WithGroup("g")
		}
		assert.NotPanics(t, func() { l.InfoCtx(context.Background(), "m", logging.Str("leaf", "1")) })

		cur := attrs(single(t, rec))
		for range depth {
			next, ok := cur["g"].(map[string]any)
			require.True(t, ok)
			cur = next
		}
		assert.Equal(t, map[string]any{"leaf": "1"}, cur)
	})
}

func TestLogger_WithComponent(t *testing.T) {
	t.Run("scope", func(t *testing.T) {
		l, rec := newLogger()
		l.WithComponent("svc.sub").InfoCtx(context.Background(), "m")
		assert.Equal(t, "svc.sub", single(t, rec).Scope.Name)
	})

	t.Run("event scope", func(t *testing.T) {
		l, rec := newLogger()
		l.WithComponent("svc.sub").Emit(context.Background(), testEvent{name: "ev"})
		assert.Equal(t, "svc.sub", single(t, rec).Scope.Name)
	})

	t.Run("keeps fields", func(t *testing.T) {
		l, rec := newLogger()
		l.With(logging.Str("a", "1")).WithGroup("g").WithComponent("svc.sub").
			InfoCtx(context.Background(), "m", logging.Str("b", "2"))
		assert.Equal(t, map[string]any{"a": "1", "g": map[string]any{"b": "2"}}, attrs(single(t, rec)))
	})

	t.Run("keeps options", func(t *testing.T) {
		l, rec := newLogger(logging.WithLevel(logging.LevelWarn), logging.WithVersion("v1"))
		c := l.WithComponent("svc.sub")
		c.InfoCtx(context.Background(), "dropped")
		c.WarnCtx(context.Background(), "kept")

		r := single(t, rec)
		assert.Equal(t, "kept", r.Body().AsString())
		assert.Equal(t, "v1", r.Scope.Version)
	})

	t.Run("parent unchanged", func(t *testing.T) {
		l, rec := newLogger()
		_ = l.WithComponent("svc.sub")
		l.InfoCtx(context.Background(), "m")
		assert.Equal(t, string(logging.DefaultComponent), single(t, rec).Scope.Name)
	})
}

func TestLogger_Slog(t *testing.T) {
	t.Run("provider", func(t *testing.T) {
		l, rec := newLogger()
		l.Slog().Info("hello", "n", 1)

		r := single(t, rec)
		assert.Equal(t, "hello", r.Body().AsString())
		assert.Equal(t, log.SeverityInfo, r.Severity())
		assert.Equal(t, string(logging.DefaultComponent), r.Scope.Name)
		assert.Equal(t, map[string]any{"n": int64(1)}, attrs(r))
	})

	t.Run("prefix", func(t *testing.T) {
		l, rec := newLogger()
		l.With(logging.Str("a", "1")).WithGroup("g").With(logging.Str("b", "2")).Slog().Info("m", "c", "3")
		assert.Equal(t, map[string]any{
			"a": "1",
			"g": map[string]any{"b": "2", "c": "3"},
		}, attrs(single(t, rec)))
	})

	t.Run("level", func(t *testing.T) {
		l, rec := newLogger(logging.WithLevel(logging.LevelInfo))
		s := l.Slog()
		s.Debug("dropped")
		s.Info("kept")
		assert.Equal(t, "kept", single(t, rec).Body().AsString())
	})

	t.Run("component", func(t *testing.T) {
		l, rec := newLogger()
		l.WithComponent("svc.sub").Slog().Info("m")
		assert.Equal(t, "svc.sub", single(t, rec).Scope.Name)
	})
}

func TestLogger_OTel(t *testing.T) {
	t.Run("scope", func(t *testing.T) {
		l, rec := newLogger(logging.WithVersion("v1"))
		var r log.Record
		r.SetBody(attribute.StringValue("m"))
		l.WithComponent("svc.sub").OTel().Emit(context.Background(), r)

		got := single(t, rec)
		assert.Equal(t, "svc.sub", got.Scope.Name)
		assert.Equal(t, "v1", got.Scope.Version)
	})
}

func TestNew(t *testing.T) {
	t.Run("default scope", func(t *testing.T) {
		l, rec := newLogger()
		l.InfoCtx(context.Background(), "m")

		s := single(t, rec).Scope
		assert.Equal(t, string(logging.DefaultComponent), s.Name)
		assert.Empty(t, s.Version)
		assert.Empty(t, s.SchemaURL)
		assert.Zero(t, s.Attributes.Len())
	})

	t.Run("scope options", func(t *testing.T) {
		l, rec := newLogger(
			logging.WithComponent("svc.sub"),
			logging.WithVersion("v1"),
			logging.WithSchemaURL("https://example.com/schema"),
			logging.WithAttributes(attribute.String("k", "v")),
		)
		l.InfoCtx(context.Background(), "m")
		l.Emit(context.Background(), testEvent{name: "ev"})

		want := logtest.Scope{
			Name:       "svc.sub",
			Version:    "v1",
			SchemaURL:  "https://example.com/schema",
			Attributes: attribute.NewSet(attribute.String("k", "v")),
		}
		recs := rec.Records()
		require.Len(t, recs, 2)
		assert.Equal(t, want, recs[0].Scope)
		assert.Equal(t, want, recs[1].Scope)
	})

	t.Run("global provider", func(t *testing.T) {
		rec := logtest.NewRecorder()
		otel.SetLoggerProvider(rec)
		logging.New().InfoCtx(context.Background(), "m")
		assert.Equal(t, "m", single(t, rec).Body().AsString())
	})
}

func TestWithSource(t *testing.T) {
	source := func(t *testing.T, r *logtest.Record, pc uintptr, file string, line int) {
		t.Helper()
		a := attrs(r)
		assert.Equal(t, file, a["code.file.path"])
		assert.Equal(t, runtime.FuncForPC(pc).Name(), a["code.function.name"])
		assert.Equal(t, int64(line+1), a["code.line.number"])
	}

	t.Run("level method", func(t *testing.T) {
		l, rec := newLogger(logging.WithSource(true))
		pc, file, line, _ := runtime.Caller(0)
		l.InfoCtx(context.Background(), "m")
		source(t, single(t, rec), pc, file, line)
	})

	t.Run("log", func(t *testing.T) {
		l, rec := newLogger(logging.WithSource(true))
		pc, file, line, _ := runtime.Caller(0)
		l.Log(context.Background(), logging.LevelInfo, "m")
		source(t, single(t, rec), pc, file, line)
	})

	t.Run("emit", func(t *testing.T) {
		l, rec := newLogger(logging.WithSource(true))
		pc, file, line, _ := runtime.Caller(0)
		l.Emit(context.Background(), testEvent{name: "ev"})
		source(t, single(t, rec), pc, file, line)
	})

	t.Run("off", func(t *testing.T) {
		l, rec := newLogger()
		l.InfoCtx(context.Background(), "m")
		l.Emit(context.Background(), testEvent{name: "ev"})

		recs := rec.Records()
		require.Len(t, recs, 2)
		assert.Empty(t, attrs(&recs[0]))
		assert.Empty(t, attrs(&recs[1]))
	})
}

func TestBootstrap(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		out := captureStderr(t, func() {
			logging.Bootstrap().InfoCtx(context.Background(), "hello", logging.Int("n", 1))
		})
		assert.Contains(t, out, "level=INFO msg=hello n=1")
		assert.NotContains(t, out, "otel.scope.name")
	})

	t.Run("level names", func(t *testing.T) {
		out := captureStderr(t, func() {
			l := logging.Bootstrap(logging.WithLevel(logging.LevelTrace))
			l.Log(context.Background(), logging.LevelTrace, "t")
			l.Log(context.Background(), logging.LevelFatal, "f")
			l.Log(context.Background(), logging.LevelInfo+2, "o")
		})
		assert.Contains(t, out, "level=TRACE msg=t")
		assert.Contains(t, out, "level=FATAL msg=f")
		assert.Contains(t, out, "level=INFO+2 msg=o")
	})

	t.Run("below level", func(t *testing.T) {
		out := captureStderr(t, func() {
			logging.Bootstrap().DebugCtx(context.Background(), "hello")
		})
		assert.Empty(t, out)
	})

	t.Run("group", func(t *testing.T) {
		out := captureStderr(t, func() {
			logging.Bootstrap().With(logging.Str("a", "1")).WithGroup("g").
				InfoCtx(context.Background(), "m", logging.Str("b", "2"))
		})
		assert.Contains(t, out, "a=1 g.b=2")
	})

	t.Run("event", func(t *testing.T) {
		ev := testEvent{name: "user.login", attrs: []attribute.KeyValue{attribute.String("user.id", "42")}}
		out := captureStderr(t, func() {
			logging.Bootstrap().Emit(context.Background(), ev)
		})
		assert.Contains(t, out, "user.id=42")
		assert.Contains(t, out, "event.name=user.login")
	})

	t.Run("error", func(t *testing.T) {
		out := captureStderr(t, func() {
			logging.Bootstrap().ErrorCtx(context.Background(), "failed", logging.Err(errors.New("boom")))
		})
		assert.Contains(t, out, "level=ERROR msg=failed")
		assert.Contains(t, out, "error.message=boom")
	})

	t.Run("component", func(t *testing.T) {
		out := captureStderr(t, func() {
			logging.Bootstrap(logging.WithComponent("svc.sub")).InfoCtx(context.Background(), "m")
			logging.Bootstrap().WithComponent("svc.other").InfoCtx(context.Background(), "m")
		})
		assert.Contains(t, out, "otel.scope.name=svc.sub")
		assert.Contains(t, out, "otel.scope.name=svc.other")
	})

	t.Run("ignores provider", func(t *testing.T) {
		rec := logtest.NewRecorder()
		out := captureStderr(t, func() {
			logging.Bootstrap(logging.WithProvider(rec)).InfoCtx(context.Background(), "hello")
		})
		assert.Contains(t, out, "msg=hello")
		assert.Empty(t, rec.Records())
	})

	t.Run("source", func(t *testing.T) {
		out := captureStderr(t, func() {
			logging.Bootstrap(logging.WithSource(true)).InfoCtx(context.Background(), "m")
		})
		assert.Contains(t, out, "logger_test.go:")
	})
}

func testLevelMethod(t *testing.T, call levelMethod, lvl logging.Level, sev log.Severity, text string) {
	t.Helper()

	t.Run("record", func(t *testing.T) {
		l, rec := newLogger()
		call(l, context.Background(), "hello")

		r := single(t, rec)
		assert.Equal(t, sev, r.Severity())
		assert.Equal(t, text, r.SeverityText())
		assert.Equal(t, "hello", r.Body().AsString())
		assert.Empty(t, r.EventName())
		assert.False(t, r.Timestamp().IsZero())
		assert.Equal(t, string(logging.DefaultComponent), r.Scope.Name)
	})

	t.Run("fields", func(t *testing.T) {
		l, rec := newLogger()
		call(l, context.Background(), "hello", logging.Str("s", "v"), logging.Int("n", 1))
		assert.Equal(t, map[string]any{"s": "v", "n": int64(1)}, attrs(single(t, rec)))
	})

	t.Run("below level", func(t *testing.T) {
		l, rec := newLogger(logging.WithLevel(lvl + 1))
		call(l, context.Background(), "hello")
		assert.Empty(t, rec.Records())
	})

	t.Run("at level", func(t *testing.T) {
		l, rec := newLogger(logging.WithLevel(lvl))
		call(l, context.Background(), "hello")
		assert.Len(t, rec.Records(), 1)
	})

	t.Run("provider disabled", func(t *testing.T) {
		rec := logtest.NewRecorder(logtest.WithEnabled(func(context.Context, logtest.Scope, log.EnabledParameters) bool {
			return false
		}))
		call(logging.New(logging.WithProvider(rec), logging.WithLevel(logging.LevelTrace)), context.Background(), "hello")
		assert.Empty(t, rec.Records())
	})

	t.Run("context", func(t *testing.T) {
		l, rec := newLogger()
		call(l, context.WithValue(context.Background(), ctxKey{}, "v"), "hello")
		assert.Equal(t, "v", single(t, rec).Context.Value(ctxKey{}))
	})

	t.Run("nil context", func(t *testing.T) {
		l, rec := newLogger()
		var ctx context.Context
		assert.NotPanics(t, func() { call(l, ctx, "hello") })
		assert.Len(t, rec.Records(), 1)
	})
}
