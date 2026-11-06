package logging_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/webitel/webitel-go-kit/pkg/logging"
	"github.com/webitel/webitel-go-kit/pkg/logging/redact"
)

func TestGroup(t *testing.T) {
	cases := map[string]struct {
		field logging.Field
		want  map[string]any
	}{
		"flat": {
			field: logging.Group("g", logging.Str("a", "1"), logging.Int("b", 2)),
			want:  map[string]any{"g": map[string]any{"a": "1", "b": int64(2)}},
		},
		"nested": {
			field: logging.Group("a", logging.Group("b", logging.Int("c", 1))),
			want:  map[string]any{"a": map[string]any{"b": map[string]any{"c": int64(1)}}},
		},
		"empty": {
			field: logging.Group("g"),
			want:  map[string]any{},
		},
		"empty nested": {
			field: logging.Group("a", logging.Group("b")),
			want:  map[string]any{},
		},
		"inline": {
			field: logging.Group("", logging.Str("a", "1"), logging.Str("b", "2")),
			want:  map[string]any{"a": "1", "b": "2"},
		},
		"inline nested": {
			field: logging.Group("g", logging.Group("", logging.Str("a", "1"))),
			want:  map[string]any{"g": map[string]any{"a": "1"}},
		},
		"zero fields": {
			field: logging.Group("g", logging.Err(nil), logging.Ev(nil), logging.Str("a", "1")),
			want:  map[string]any{"g": map[string]any{"a": "1"}},
		},
		"secret": {
			field: logging.Group("g", logging.Str("a", "1")).Secret(),
			want:  map[string]any{"g": redact.SecretSet},
		},
		"secret child": {
			field: logging.Group("g", logging.Str("token", "t").Secret(), logging.Str("a", "1")),
			want:  map[string]any{"g": map[string]any{"token": redact.SecretSet, "a": "1"}},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			redact.SetPolicy(redact.Policy{Salt: testSalt})
			assert.Equal(t, tc.want, logged(t, tc.field))
		})
	}

	t.Run("pii", func(t *testing.T) {
		redact.SetPolicy(redact.Policy{Salt: testSalt})
		got := logged(t, logging.Group("g", logging.Str("a", "1")).PII())
		assert.Regexp(t, `^pii:[0-9a-f]{32}$`, got["g"])
	})

	t.Run("pii child", func(t *testing.T) {
		redact.SetPolicy(redact.Policy{Salt: testSalt})
		got := logged(t, logging.Group("g", logging.Str("email", "alice@example.com").PII()))
		assert.Equal(t, map[string]any{"g": map[string]any{"email": pseudonym("alice@example.com")}}, got)
	})

	t.Run("deep", func(t *testing.T) {
		const depth = 64
		f := logging.Str("leaf", "1")
		for range depth {
			f = logging.Group("g", f)
		}

		var got map[string]any
		require.NotPanics(t, func() { got = logged(t, f) })
		for range depth {
			next, ok := got["g"].(map[string]any)
			require.True(t, ok)
			got = next
		}
		assert.Equal(t, map[string]any{"leaf": "1"}, got)
	})

	t.Run("error", func(t *testing.T) {
		err := errors.New("boom")
		l, rec := newLogger()
		l.InfoCtx(context.Background(), "m", logging.Group("g", logging.Err(err)))

		r := single(t, rec)
		assert.Equal(t, err, r.Err())
		g, ok := attrs(r)["g"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, g, "error")
	})
}

func TestAny(t *testing.T) {
	now := time.Unix(1700000000, 5)
	n := 42
	var nilPtr *int
	var nilStringer *stringer
	var nilErr *ptrError

	cases := map[string]struct {
		in   any
		want any
	}{
		"nil":            {nil, nil},
		"string":         {"x", "x"},
		"bool":           {true, true},
		"int8":           {int8(-3), int64(-3)},
		"uint16":         {uint16(3), int64(3)},
		"uint64":         {uint64(3), int64(3)},
		"uint64 max":     {uint64(math.MaxUint64), "18446744073709551615"},
		"float32":        {float32(1.5), 1.5},
		"duration":       {time.Second, int64(time.Second)},
		"time":           {now, now.UnixNano()},
		"bytes":          {[]byte("ab"), []byte("ab")},
		"strings":        {[]string{"a", "b"}, []string{"a", "b"}},
		"ints":           {[]int{1, 2}, []int64{1, 2}},
		"mixed slice":    {[]any{1, "a", nil}, []any{int64(1), "a", nil}},
		"array":          {[2]string{"a", "b"}, []any{"a", "b"}},
		"nil slice":      {[]any(nil), nil},
		"map":            {map[string]int{"a": 1}, map[string]any{"a": int64(1)}},
		"int keys":       {map[int]string{1: "a"}, map[string]any{"1": "a"}},
		"nested map":     {map[string]any{"a": map[string]any{"b": []any{1}}}, map[string]any{"a": map[string]any{"b": []any{int64(1)}}}},
		"pointer":        {&n, int64(42)},
		"nil pointer":    {nilPtr, nil},
		"struct":         {struct{ A int }{1}, "{A:1}"},
		"stringer":       {&stringer{"s"}, "s"},
		"nil stringer":   {nilStringer, nil},
		"error":          {errors.New("boom"), "boom"},
		"nil error":      {nilErr, nil},
		"typed nil":      {error(nilErr), nil},
		"slog value":     {slog.IntValue(1), int64(1)},
		"attribute":      {attribute.StringSliceValue([]string{"a"}), []string{"a"}},
		"valuer":         {valuer{slog.StringValue("v")}, "v"},
		"valuer group":   {valuer{slog.GroupValue(slog.Int("a", 1))}, map[string]any{"a": int64(1)}},
		"valuer depth 3": {valuer{slog.AnyValue(valuer{slog.AnyValue(valuer{slog.StringValue("deep")})})}, "deep"},
		"valuer in map":  {map[string]any{"a": valuer{slog.StringValue("v")}}, map[string]any{"a": "v"}},
		"valuer in list": {[]any{valuer{slog.StringValue("v")}}, []any{"v"}},
		"panic stringer": {panicStringer{}, "!PANIC: boom"},
		"panic error":    {panicError{}, "!PANIC: boom"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var got map[string]any
			require.NotPanics(t, func() { got = logged(t, logging.Any("x", tc.in)) })
			assert.Equal(t, map[string]any{"x": tc.want}, got)
		})
	}

	t.Run("nil empty key", func(t *testing.T) {
		assert.Empty(t, logged(t, logging.Any("", nil)))
	})

	t.Run("max depth", func(t *testing.T) {
		got := logged(t, logging.Any("x", nested(32, "leaf")))
		assert.Equal(t, "<max-depth-exceeded>", innermost(got["x"]))
	})

	t.Run("under max depth", func(t *testing.T) {
		got := logged(t, logging.Any("x", nested(8, "leaf")))
		assert.Equal(t, "leaf", innermost(got["x"]))
	})

	t.Run("map cycle", func(t *testing.T) {
		m := map[string]any{}
		m["self"] = m
		assert.NotPanics(t, func() { logged(t, logging.Any("x", m)) })
	})

	t.Run("slice cycle", func(t *testing.T) {
		s := []any{nil}
		s[0] = s
		var got map[string]any
		require.NotPanics(t, func() { got = logged(t, logging.Any("x", s)) })
		assert.Equal(t, "<max-depth-exceeded>", innermost(got["x"]))
	})

	t.Run("pointer cycle", func(t *testing.T) {
		var v any
		v = &v
		assert.NotPanics(t, func() { logged(t, logging.Any("x", v)) })
	})

	t.Run("valuer self", func(t *testing.T) {
		var got map[string]any
		require.NotPanics(t, func() { got = logged(t, logging.Any("x", selfValuer{})) })
		assert.Contains(t, got["x"], "LogValue called too many times")
	})

	t.Run("valuer panic", func(t *testing.T) {
		var got map[string]any
		require.NotPanics(t, func() { got = logged(t, logging.Any("x", panicValuer{})) })
		s, ok := got["x"].(string)
		require.True(t, ok)
		assert.True(t, strings.HasPrefix(s, "LogValue panicked"))
	})

	t.Run("pii", func(t *testing.T) {
		redact.SetPolicy(redact.Policy{Salt: testSalt})
		assert.Equal(t, map[string]any{"x": pseudonym("42")}, logged(t, logging.Any("x", 42).PII()))
	})
}

func TestAttr(t *testing.T) {
	cases := map[string]struct {
		in   slog.Attr
		want map[string]any
	}{
		"scalar": {
			in:   slog.Int("a", 1),
			want: map[string]any{"a": int64(1)},
		},
		"group": {
			in:   slog.Group("g", slog.Int("a", 1), slog.Group("h", slog.String("b", "2"))),
			want: map[string]any{"g": map[string]any{"a": int64(1), "h": map[string]any{"b": "2"}}},
		},
		"inline group": {
			in:   slog.Group("", slog.Int("a", 1), slog.Int("b", 2)),
			want: map[string]any{"a": int64(1), "b": int64(2)},
		},
		"empty group": {
			in:   slog.Group("g"),
			want: map[string]any{},
		},
		"empty nested group": {
			in:   slog.Group("g", slog.Group("h"), slog.Int("a", 1)),
			want: map[string]any{"g": map[string]any{"a": int64(1)}},
		},
		"nil empty key": {
			in:   slog.Any("", nil),
			want: map[string]any{},
		},
		"valuer": {
			in:   slog.Any("v", valuer{slog.GroupValue(slog.Int("a", 1))}),
			want: map[string]any{"v": map[string]any{"a": int64(1)}},
		},
		"valuer in group": {
			in:   slog.Group("g", slog.Any("v", valuer{slog.StringValue("x")})),
			want: map[string]any{"g": map[string]any{"v": "x"}},
		},
		"inline valuer": {
			in:   slog.Any("", valuer{slog.GroupValue(slog.Int("a", 1))}),
			want: map[string]any{"a": int64(1)},
		},
		"duration": {
			in:   slog.Duration("d", time.Millisecond),
			want: map[string]any{"d": int64(time.Millisecond)},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, logged(t, logging.Attr(tc.in)))
		})
	}

	t.Run("max depth", func(t *testing.T) {
		a := slog.String("leaf", "1")
		for range 32 {
			a = slog.Group("g", a)
		}
		got := logged(t, logging.Attr(a))

		var v any = got["g"]
		for {
			m, ok := v.(map[string]any)
			if !ok {
				break
			}
			v = m["g"]
			if v == nil {
				v = m["leaf"]
			}
		}
		assert.Equal(t, "<max-depth-exceeded>", v)
	})

	t.Run("secret", func(t *testing.T) {
		redact.SetPolicy(redact.Policy{Salt: testSalt})
		assert.Equal(t, map[string]any{"token": redact.SecretSet}, logged(t, logging.Attr(slog.String("token", "t")).Secret()))
	})
}

func TestKV(t *testing.T) {
	t.Run("flat", func(t *testing.T) {
		got := logged(t, logging.KV(attribute.String("a", "1"), attribute.Int("b", 2)))
		assert.Equal(t, map[string]any{"a": "1", "b": int64(2)}, got)
	})

	t.Run("empty", func(t *testing.T) {
		assert.Empty(t, logged(t, logging.KV()))
	})

	t.Run("in group", func(t *testing.T) {
		got := logged(t, logging.Group("g", logging.KV(attribute.String("a", "1"))))
		assert.Equal(t, map[string]any{"g": map[string]any{"a": "1"}}, got)
	})

	t.Run("map value", func(t *testing.T) {
		got := logged(t, logging.KV(attribute.Map("m", attribute.String("a", "1"))))
		assert.Equal(t, map[string]any{"m": map[string]any{"a": "1"}}, got)
	})

	t.Run("secret", func(t *testing.T) {
		redact.SetPolicy(redact.Policy{Salt: testSalt})
		got := logged(t, logging.KV(attribute.String("a", "1"), attribute.String("b", "")).Secret())
		assert.Equal(t, map[string]any{"a": redact.SecretSet, "b": redact.SecretEmpty}, got)
	})
}

func TestMap(t *testing.T) {
	classOf := func(k string) logging.Class {
		switch k {
		case "email":
			return redact.ClassPII
		case "password":
			return redact.ClassSecret
		default:
			return redact.ClassRegular
		}
	}

	t.Run("classes", func(t *testing.T) {
		redact.SetPolicy(redact.Policy{Salt: testSalt})
		got := logged(t, logging.Map("u", map[string]string{"email": "alice@example.com", "password": "p", "name": "Alice"}, classOf))
		assert.Equal(t, map[string]any{"u": map[string]any{
			"email":    pseudonym("alice@example.com"),
			"password": redact.SecretSet,
			"name":     "Alice",
		}}, got)
	})

	t.Run("dropped", func(t *testing.T) {
		redact.SetPolicy(redact.Policy{SecretMode: redact.Drop})
		got := logged(t, logging.Map("u", map[string]string{"password": "p", "name": "Alice"}, classOf))
		assert.Equal(t, map[string]any{"u": map[string]any{"name": "Alice"}}, got)
	})

	t.Run("nil classOf", func(t *testing.T) {
		got := logged(t, logging.Map("u", map[string]int{"a": 1}, nil))
		assert.Equal(t, map[string]any{"u": map[string]any{"a": int64(1)}}, got)
	})

	t.Run("nested values", func(t *testing.T) {
		in := map[string]any{"a": map[string]any{"b": []any{1, valuer{slog.StringValue("v")}}}}
		got := logged(t, logging.Map("u", in, nil))
		assert.Equal(t, map[string]any{"u": map[string]any{"a": map[string]any{"b": []any{int64(1), "v"}}}}, got)
	})

	t.Run("nil map", func(t *testing.T) {
		got := logged(t, logging.Map[string]("u", nil, classOf))
		assert.Equal(t, map[string]any{"u": map[string]any{}}, got)
	})

	t.Run("in group", func(t *testing.T) {
		redact.SetPolicy(redact.Policy{Salt: testSalt})
		got := logged(t, logging.Group("g", logging.Map("u", map[string]string{"password": "p"}, classOf)))
		assert.Equal(t, map[string]any{"g": map[string]any{"u": map[string]any{"password": redact.SecretSet}}}, got)
	})

	t.Run("cycle", func(t *testing.T) {
		m := map[string]any{}
		m["self"] = m
		assert.NotPanics(t, func() { logged(t, logging.Map("u", m, nil)) })
	})
}

func TestEv(t *testing.T) {
	ev := testEvent{name: "user.login", attrs: []attribute.KeyValue{attribute.String("user.id", "42")}}

	t.Run("field", func(t *testing.T) {
		l, rec := newLogger()
		l.WarnCtx(context.Background(), "hello", logging.Ev(ev), logging.Int("n", 1))

		r := single(t, rec)
		assert.Equal(t, "user.login", r.EventName())
		assert.Equal(t, "hello", r.Body().AsString())
		assert.Equal(t, "WARN", r.SeverityText())
		assert.Equal(t, map[string]any{"user.id": "42", "n": int64(1)}, attrs(r))
	})

	t.Run("nil", func(t *testing.T) {
		l, rec := newLogger()
		l.InfoCtx(context.Background(), "m", logging.Ev(nil), logging.Int("n", 1))

		r := single(t, rec)
		assert.Empty(t, r.EventName())
		assert.Equal(t, map[string]any{"n": int64(1)}, attrs(r))
	})

	t.Run("from with", func(t *testing.T) {
		l, rec := newLogger()
		l.WithGroup("g").With(logging.Ev(ev)).InfoCtx(context.Background(), "m", logging.Int("n", 1))

		r := single(t, rec)
		assert.Equal(t, "user.login", r.EventName())
		assert.Equal(t, map[string]any{"user.id": "42", "g": map[string]any{"n": int64(1)}}, attrs(r))
	})

	t.Run("below level", func(t *testing.T) {
		l, rec := newLogger(logging.WithLevel(logging.LevelWarn))
		l.InfoCtx(context.Background(), "m", logging.Ev(ev))
		assert.Empty(t, rec.Records())
	})

	groupCases := map[string]struct {
		field logging.Field
		want  map[string]any
	}{
		"in group": {
			field: logging.Group("g", logging.Ev(ev), logging.Str("a", "1")),
			want:  map[string]any{"g": map[string]any{"a": "1"}},
		},
		"in inline group": {
			field: logging.Group("", logging.Ev(ev), logging.Str("a", "1")),
			want:  map[string]any{"a": "1"},
		},
		"in nested group": {
			field: logging.Group("g", logging.Group("h", logging.Ev(ev), logging.Str("a", "1"))),
			want:  map[string]any{"g": map[string]any{"h": map[string]any{"a": "1"}}},
		},
		"only in group": {
			field: logging.Group("g", logging.Ev(ev)),
			want:  map[string]any{},
		},
	}
	for name, tc := range groupCases {
		t.Run(name, func(t *testing.T) {
			l, rec := newLogger()
			l.InfoCtx(context.Background(), "m", tc.field)

			r := single(t, rec)
			assert.Empty(t, r.EventName())
			assert.Equal(t, tc.want, attrs(r))
		})
	}

	t.Run("group and root", func(t *testing.T) {
		l, rec := newLogger()
		inner := testEvent{name: "inner", attrs: []attribute.KeyValue{attribute.String("inner", "1")}}
		l.InfoCtx(context.Background(), "m", logging.Ev(ev), logging.Group("g", logging.Ev(inner), logging.Str("a", "1")))

		r := single(t, rec)
		assert.Equal(t, "user.login", r.EventName())
		assert.Equal(t, map[string]any{"user.id": "42", "g": map[string]any{"a": "1"}}, attrs(r))
	})

	t.Run("in group bootstrap", func(t *testing.T) {
		out := captureStderr(t, func() {
			logging.Bootstrap().InfoCtx(context.Background(), "m", logging.Group("g", logging.Ev(ev), logging.Str("a", "1")))
		})
		assert.Contains(t, out, "g.a=1")
		assert.NotContains(t, out, "event.name")
		assert.NotContains(t, out, "user.id")
	})
}

func TestField_Secret(t *testing.T) {
	cases := map[string]struct {
		field logging.Field
		want  string
	}{
		"string":      {logging.Str("x", "t"), redact.SecretSet},
		"empty":       {logging.Str("x", ""), redact.SecretEmpty},
		"int":         {logging.Int("x", 7), redact.SecretSet},
		"zero int":    {logging.Int("x", 0), redact.SecretEmpty},
		"zero int64":  {logging.Int64("x", 0), redact.SecretEmpty},
		"zero uint64": {logging.Uint64("x", 0), redact.SecretEmpty},
		"float":       {logging.Float64("x", 0.5), redact.SecretSet},
		"zero float":  {logging.Float64("x", 0), redact.SecretEmpty},
		"dur":         {logging.Dur("x", time.Second), redact.SecretSet},
		"zero dur":    {logging.Dur("x", 0), redact.SecretEmpty},
		"true":        {logging.Bool("x", true), redact.SecretSet},
		"false":       {logging.Bool("x", false), redact.SecretSet},
		"zero named":  {logging.Any("x", id(0)), redact.SecretEmpty},
		"named":       {logging.Any("x", id(7)), redact.SecretSet},
		"nil":         {logging.Any("x", nil), redact.SecretEmpty},
		"empty slice": {logging.Any("x", []string{}), redact.SecretEmpty},
		"slice":       {logging.Any("x", []string{"a"}), redact.SecretSet},
		"empty group": {logging.Attr(slog.Any("x", valuer{slog.GroupValue()})), redact.SecretEmpty},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			redact.SetPolicy(redact.Policy{Salt: testSalt})
			assert.Equal(t, map[string]any{"x": tc.want}, logged(t, tc.field.Secret()))
		})
	}

	t.Run("drop", func(t *testing.T) {
		redact.SetPolicy(redact.Policy{SecretMode: redact.Drop})
		assert.Empty(t, logged(t, logging.Str("x", "t").Secret(), logging.Int("y", 0).Secret()))
	})

	t.Run("original kept", func(t *testing.T) {
		redact.SetPolicy(redact.Policy{Salt: testSalt})
		f := logging.Str("x", "t")
		_ = f.Secret()
		assert.Equal(t, map[string]any{"x": "t"}, logged(t, f))
	})
}

func TestField_PII(t *testing.T) {
	cases := map[string]struct {
		field logging.Field
		text  string
	}{
		"string":   {logging.Str("x", "alice"), "alice"},
		"zero int": {logging.Int("x", 0), "0"},
		"int":      {logging.Int("x", 42), "42"},
		"false":    {logging.Bool("x", false), "false"},
		"any":      {logging.Any("x", &stringer{"alice"}), "alice"},
		"attr":     {logging.Attr(slog.String("x", "alice")), "alice"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			redact.SetPolicy(redact.Policy{Salt: testSalt})
			assert.Equal(t, map[string]any{"x": pseudonym(tc.text)}, logged(t, tc.field.PII()))
		})
	}

	t.Run("with", func(t *testing.T) {
		redact.SetPolicy(redact.Policy{Salt: testSalt})
		l, rec := newLogger()
		l.With(logging.Str("x", "alice").PII()).InfoCtx(context.Background(), "m")
		assert.Equal(t, map[string]any{"x": pseudonym("alice")}, attrs(single(t, rec)))
	})

	t.Run("policy at emit", func(t *testing.T) {
		redact.SetPolicy(redact.Policy{Salt: testSalt})
		l, rec := newLogger()
		l = l.With(logging.Str("x", "alice").PII())
		redact.SetPolicy(redact.Policy{Salt: []byte("other-salt")})
		l.InfoCtx(context.Background(), "m")
		assert.Equal(t, map[string]any{"x": pseudonym("alice")}, attrs(single(t, rec)))
	})
}

func TestErr(t *testing.T) {
	errorAttr := func(t *testing.T, err error) map[string]any {
		t.Helper()
		var got map[string]any
		require.NotPanics(t, func() { got = logged(t, logging.Err(err)) })
		e, ok := got["error"].(map[string]any)
		require.True(t, ok)
		return e
	}

	t.Run("nil", func(t *testing.T) {
		l, rec := newLogger()
		l.InfoCtx(context.Background(), "m", logging.Err(nil))

		r := single(t, rec)
		assert.Empty(t, attrs(r))
		assert.NoError(t, r.Err())
	})

	t.Run("plain", func(t *testing.T) {
		err := errors.New("boom")
		l, rec := newLogger()
		l.InfoCtx(context.Background(), "m", logging.Err(err))

		r := single(t, rec)
		assert.Equal(t, err, r.Err())
		assert.Equal(t, map[string]any{"error": map[string]any{
			"type":    "*errors.errorString",
			"message": "boom",
		}}, attrs(r))
	})

	t.Run("wrapped", func(t *testing.T) {
		err := fmt.Errorf("outer: %w", errors.New("inner"))
		assert.Equal(t, map[string]any{
			"type":    "*fmt.wrapError",
			"message": "outer: inner",
			"chain":   []string{"outer: inner", "inner"},
		}, errorAttr(t, err))
	})

	t.Run("dedupe", func(t *testing.T) {
		err := &wrapError{msg: "same", err: &wrapError{msg: "same", err: errors.New("root")}}
		assert.Equal(t, []string{"same", "root"}, errorAttr(t, err)["chain"])
	})

	t.Run("join", func(t *testing.T) {
		err := errors.Join(errors.New("a"), errors.New("b"))
		assert.Equal(t, []string{"a\nb", "a", "b"}, errorAttr(t, err)["chain"])
	})

	t.Run("join tree", func(t *testing.T) {
		err := errors.Join(fmt.Errorf("x: %w", errors.New("a")), errors.Join(errors.New("b"), errors.New("c")))
		assert.Equal(t, []string{"x: a\nb\nc", "x: a", "a", "b\nc", "b", "c"}, errorAttr(t, err)["chain"])
	})

	t.Run("nil branch", func(t *testing.T) {
		err := multiError{nil, errors.New("a"), nil}
		assert.Equal(t, []string{"multi", "a"}, errorAttr(t, err)["chain"])
	})

	t.Run("nil wrapped", func(t *testing.T) {
		err := &wrapError{msg: "outer"}
		e := errorAttr(t, err)
		assert.Equal(t, "outer", e["message"])
		assert.NotContains(t, e, "chain")
	})

	t.Run("nil pointer", func(t *testing.T) {
		var err *ptrError
		e := errorAttr(t, err)
		assert.Equal(t, "*logging_test.ptrError", e["type"])
		assert.Contains(t, e["message"], "!PANIC:")
	})

	t.Run("nil pointer wrapped", func(t *testing.T) {
		var inner *ptrError
		e := errorAttr(t, &wrapError{msg: "outer", err: inner})
		chain, ok := e["chain"].([]string)
		require.True(t, ok)
		assert.Len(t, chain, 2)
	})

	t.Run("panic", func(t *testing.T) {
		e := errorAttr(t, &wrapError{msg: "outer", err: panicError{}})
		assert.Equal(t, []string{"outer", "!PANIC: boom"}, e["chain"])
	})

	t.Run("cycle", func(t *testing.T) {
		a := &loopError{msg: "a"}
		b := &loopError{msg: "b", next: a}
		a.next = b

		chain, ok := errorAttr(t, a)["chain"].([]string)
		require.True(t, ok)
		assert.Len(t, chain, 32)
	})

	t.Run("self cycle", func(t *testing.T) {
		a := &loopError{msg: "a"}
		a.next = a

		e := errorAttr(t, a)
		assert.Equal(t, "a", e["message"])
		assert.NotContains(t, e, "chain")
	})

	t.Run("values", func(t *testing.T) {
		err := &valueError{
			msg:    "outer",
			values: []slog.Attr{slog.Int("code", 1)},
			err: &valueError{
				msg:    "inner",
				values: []slog.Attr{slog.Int("code", 2), slog.String("id", "x")},
			},
		}
		e := errorAttr(t, err)
		assert.Equal(t, int64(1), e["code"])
		assert.Equal(t, "x", e["id"])
	})

	t.Run("reserved keys", func(t *testing.T) {
		err := &valueError{
			msg:    "real",
			values: []slog.Attr{slog.String("message", "fake"), slog.String("type", "fake"), slog.String("chain", "fake")},
		}
		e := errorAttr(t, err)
		assert.Equal(t, "real", e["message"])
		assert.Equal(t, "*logging_test.valueError", e["type"])
		assert.NotContains(t, e, "chain")
	})

	t.Run("nested values", func(t *testing.T) {
		err := &valueError{msg: "m", values: []slog.Attr{slog.Group("g", slog.Any("v", valuer{slog.StringValue("x")}))}}
		assert.Equal(t, map[string]any{"v": "x"}, errorAttr(t, err)["g"])
	})

	t.Run("scalar value", func(t *testing.T) {
		assert.Equal(t, int64(7), errorAttr(t, &scalarError{msg: "m"})["value"])
	})

	t.Run("nil pointer valuer", func(t *testing.T) {
		var inner *valueError
		assert.NotPanics(t, func() { errorAttr(t, &wrapError{msg: "outer", err: inner}) })
	})

	t.Run("event path", func(t *testing.T) {
		err := errors.New("boom")
		l, rec := newLogger()
		l.Emit(context.Background(), testEvent{name: "ev"}, logging.Err(err))
		assert.Equal(t, err, single(t, rec).Err())
	})

	t.Run("with", func(t *testing.T) {
		err := errors.New("boom")
		l, rec := newLogger()
		l.With(logging.Err(err)).InfoCtx(context.Background(), "m")

		r := single(t, rec)
		assert.Equal(t, err, r.Err())
		assert.Contains(t, attrs(r), "error")
	})

	t.Run("pii", func(t *testing.T) {
		redact.SetPolicy(redact.Policy{Salt: testSalt})
		l, rec := newLogger()
		l.InfoCtx(context.Background(), "m", logging.Err(errors.New("alice@example.com")).PII())

		r := single(t, rec)
		assert.NoError(t, r.Err())
		assert.Regexp(t, `^pii:[0-9a-f]{32}$`, attrs(r)["error"])
	})

	t.Run("secret", func(t *testing.T) {
		redact.SetPolicy(redact.Policy{Salt: testSalt})
		l, rec := newLogger()
		l.InfoCtx(context.Background(), "m", logging.Err(errors.New("token")).Secret())

		r := single(t, rec)
		assert.NoError(t, r.Err())
		assert.Equal(t, map[string]any{"error": redact.SecretSet}, attrs(r))
	})
}

type id int

type valuer struct{ v slog.Value }

func (v valuer) LogValue() slog.Value { return v.v }

type selfValuer struct{}

func (s selfValuer) LogValue() slog.Value { return slog.AnyValue(s) }

type panicValuer struct{}

func (panicValuer) LogValue() slog.Value { panic("boom") }

type panicStringer struct{}

func (panicStringer) String() string { panic("boom") }

type stringer struct{ s string }

func (s *stringer) String() string { return s.s }

type panicError struct{}

func (panicError) Error() string { panic("boom") }

type ptrError struct{ msg string }

func (e *ptrError) Error() string { return e.msg }

type wrapError struct {
	msg string
	err error
}

func (e *wrapError) Error() string { return e.msg }
func (e *wrapError) Unwrap() error { return e.err }

type multiError []error

func (m multiError) Error() string   { return "multi" }
func (m multiError) Unwrap() []error { return m }

type valueError struct {
	msg    string
	err    error
	values []slog.Attr
}

func (e *valueError) Error() string        { return e.msg }
func (e *valueError) Unwrap() error        { return e.err }
func (e *valueError) LogValue() slog.Value { return slog.GroupValue(e.values...) }

type scalarError struct{ msg string }

func (e *scalarError) Error() string        { return e.msg }
func (e *scalarError) LogValue() slog.Value { return slog.IntValue(7) }

type loopError struct {
	msg  string
	next *loopError
}

func (e *loopError) Error() string { return e.msg }
func (e *loopError) Unwrap() error { return e.next }

func logged(t *testing.T, fields ...logging.Field) map[string]any {
	t.Helper()
	l, rec := newLogger()
	l.InfoCtx(context.Background(), "m", fields...)
	return attrs(single(t, rec))
}

func nested(depth int, leaf any) any {
	v := leaf
	for range depth {
		v = []any{v}
	}
	return v
}

func innermost(v any) any {
	for {
		s, ok := v.([]any)
		if !ok || len(s) != 1 {
			return v
		}
		v = s[0]
	}
}
