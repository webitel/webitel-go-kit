package redact

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"maps"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testSalt = []byte("test-salt")

func TestClass_String(t *testing.T) {
	cases := map[string]struct {
		class Class
		want  string
	}{
		"regular": {ClassRegular, "regular"},
		"pii":     {ClassPII, "pii"},
		"secret":  {ClassSecret, "secret"},
		"unknown": {Class(7), "class(7)"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.class.String())
		})
	}
}

func TestSetPolicy(t *testing.T) {
	t.Run("clones salt", func(t *testing.T) {
		resetPolicy(t)
		salt := []byte("mutable")
		SetPolicy(Policy{Salt: salt})
		before, _ := Render(ClassPII, "alice", false)

		salt[0] = 'X'
		after, _ := Render(ClassPII, "alice", false)

		assert.Equal(t, before, after)
		assert.Equal(t, expectedPseudonym([]byte("mutable"), "alice"), after)
	})

	t.Run("replaces", func(t *testing.T) {
		resetPolicy(t)
		SetPolicy(Policy{SecretMode: Presence})
		_, keep := Render(ClassSecret, "token", false)
		require.True(t, keep)

		SetPolicy(Policy{SecretMode: Drop})
		_, keep = Render(ClassSecret, "token", false)
		assert.False(t, keep)
	})

	t.Run("concurrent", func(t *testing.T) {
		resetPolicy(t)
		var wg sync.WaitGroup
		for i := range 8 {
			wg.Go(func() {
				for range 100 {
					if i%2 == 0 {
						SetPolicy(Policy{Salt: testSalt})
					} else {
						Render(ClassPII, "alice", false)
						Render(ClassSecret, "token", false)
					}
				}
			})
		}
		wg.Wait()
	})
}

func TestRender(t *testing.T) {
	pseudonym := expectedPseudonym([]byte("other-salt"), "alice")
	cases := map[string]struct {
		policy   *Policy
		class    Class
		in       string
		empty    bool
		want     string
		wantKeep bool
	}{
		"regular":               {nil, ClassRegular, "x", false, "x", true},
		"regular with policy":   {&Policy{SecretMode: Drop}, ClassRegular, "x", false, "x", true},
		"pii closed":            {nil, ClassPII, "alice", false, "", false},
		"pii no salt":           {&Policy{}, ClassPII, "alice", false, "", false},
		"pii salt":              {&Policy{Salt: testSalt}, ClassPII, "alice", false, expectedPseudonym(testSalt, "alice"), true},
		"pii empty":             {&Policy{Salt: testSalt}, ClassPII, "", true, expectedPseudonym(testSalt, ""), true},
		"secret closed":         {nil, ClassSecret, "token", false, SecretSet, true},
		"secret closed empty":   {nil, ClassSecret, "", true, SecretEmpty, true},
		"secret presence":       {&Policy{SecretMode: Presence}, ClassSecret, "token", false, SecretSet, true},
		"secret presence empty": {&Policy{SecretMode: Presence}, ClassSecret, "", true, SecretEmpty, true},
		"secret drop":           {&Policy{SecretMode: Drop}, ClassSecret, "token", false, "", false},
		"secret drop empty":     {&Policy{SecretMode: Drop}, ClassSecret, "", true, "", false},
		"unknown closed":        {nil, Class(7), "x", false, "", false},
		"unknown":               {&Policy{Salt: testSalt}, Class(7), "x", false, "", false},

		"pii rendered pseudonym":  {&Policy{Salt: testSalt}, ClassPII, pseudonym, false, pseudonym, true},
		"pii rendered closed":     {nil, ClassPII, pseudonym, false, pseudonym, true},
		"pii rendered no salt":    {&Policy{}, ClassPII, pseudonym, false, pseudonym, true},
		"pii rendered set":        {&Policy{Salt: testSalt}, ClassPII, SecretSet, false, SecretSet, true},
		"pii rendered empty":      {&Policy{Salt: testSalt}, ClassPII, SecretEmpty, false, SecretEmpty, true},
		"pii short pseudonym":     {&Policy{Salt: testSalt}, ClassPII, "pii:abc", false, expectedPseudonym(testSalt, "pii:abc"), true},
		"pii upper hex pseudonym": {&Policy{Salt: testSalt}, ClassPII, "pii:0123456789ABCDEF0123456789ABCDEF", false, expectedPseudonym(testSalt, "pii:0123456789ABCDEF0123456789ABCDEF"), true},
		"secret rendered set":     {nil, ClassSecret, SecretSet, false, SecretSet, true},
		"secret rendered empty":   {nil, ClassSecret, SecretEmpty, false, SecretEmpty, true},
		"secret rendered drop":    {&Policy{SecretMode: Drop}, ClassSecret, SecretSet, false, "", false},
		"secret pseudonym":        {nil, ClassSecret, pseudonym, false, SecretSet, true},
		"regular rendered":        {nil, ClassRegular, pseudonym, false, pseudonym, true},
		"unknown rendered":        {nil, Class(7), SecretSet, false, "", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resetPolicy(t)
			if tc.policy != nil {
				SetPolicy(*tc.policy)
			}

			got, keep := Render(tc.class, tc.in, tc.empty)
			assert.Equal(t, tc.wantKeep, keep)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("idempotent", func(t *testing.T) {
		resetPolicy(t)
		SetPolicy(Policy{Salt: testSalt})
		for _, c := range []Class{ClassPII, ClassSecret} {
			for _, in := range []string{"alice", ""} {
				once, keep := Render(c, in, in == "")
				require.True(t, keep)
				twice, keep := Render(c, once, once == "")
				require.True(t, keep)
				assert.Equal(t, once, twice, "class %s, value %q", c, in)
			}
		}
	})

	t.Run("pseudonym format", func(t *testing.T) {
		resetPolicy(t)
		SetPolicy(Policy{Salt: testSalt})
		got, _ := Render(ClassPII, "alice", false)
		assert.Regexp(t, `^pii:[0-9a-f]{32}$`, got)
	})

	t.Run("pseudonym stable", func(t *testing.T) {
		resetPolicy(t)
		SetPolicy(Policy{Salt: testSalt})
		a, _ := Render(ClassPII, "alice", false)
		b, _ := Render(ClassPII, "alice", false)
		assert.Equal(t, a, b)
	})

	t.Run("pseudonym by value", func(t *testing.T) {
		resetPolicy(t)
		SetPolicy(Policy{Salt: testSalt})
		a, _ := Render(ClassPII, "alice", false)
		b, _ := Render(ClassPII, "bob", false)
		assert.NotEqual(t, a, b)
	})

	t.Run("pseudonym by salt", func(t *testing.T) {
		resetPolicy(t)
		SetPolicy(Policy{Salt: testSalt})
		a, _ := Render(ClassPII, "alice", false)
		SetPolicy(Policy{Salt: []byte("other-salt")})
		b, _ := Render(ClassPII, "alice", false)
		assert.NotEqual(t, a, b)
	})
}

func TestValue(t *testing.T) {
	var nilStringer *stringer
	var nilString *string
	alice := "alice"

	cases := map[string]struct {
		class Class
		in    any
		want  string
	}{
		"secret string":       {ClassSecret, "token", SecretSet},
		"secret empty string": {ClassSecret, "", SecretEmpty},
		"secret nil":          {ClassSecret, nil, SecretEmpty},
		"secret nil pointer":  {ClassSecret, nilString, SecretEmpty},
		"secret bytes":        {ClassSecret, []byte("token"), SecretSet},
		"secret empty bytes":  {ClassSecret, []byte{}, SecretEmpty},
		"secret empty slice":  {ClassSecret, []int{}, SecretEmpty},
		"secret empty map":    {ClassSecret, map[string]int{}, SecretEmpty},
		"secret stringer":     {ClassSecret, &stringer{"token"}, SecretSet},
		"secret nil stringer": {ClassSecret, nilStringer, SecretEmpty},
		"secret empty group":  {ClassSecret, valuer{slog.GroupValue()}, SecretEmpty},
		"secret zero int":     {ClassSecret, 0, SecretEmpty},
		"secret int":          {ClassSecret, 7, SecretSet},
		"secret zero uint":    {ClassSecret, uint64(0), SecretEmpty},
		"secret zero float":   {ClassSecret, 0.0, SecretEmpty},
		"secret float":        {ClassSecret, 0.5, SecretSet},
		"secret zero dur":     {ClassSecret, time.Duration(0), SecretEmpty},
		"secret zero named":   {ClassSecret, id(0), SecretEmpty},
		"secret named":        {ClassSecret, id(7), SecretSet},
		"secret zero float32": {ClassSecret, float32(0), SecretEmpty},
		"secret false":        {ClassSecret, false, SecretSet},
		"secret true":         {ClassSecret, true, SecretSet},
		"pii zero int":        {ClassPII, 0, expectedPseudonym(testSalt, "0")},
		"secret group":        {ClassSecret, valuer{slog.GroupValue(slog.Int("a", 1))}, SecretSet},
		"pii string":          {ClassPII, "alice", expectedPseudonym(testSalt, "alice")},
		"pii pointer":         {ClassPII, &alice, expectedPseudonym(testSalt, "alice")},
		"pii bytes":           {ClassPII, []byte("alice"), expectedPseudonym(testSalt, "alice")},
		"pii stringer":        {ClassPII, &stringer{"alice"}, expectedPseudonym(testSalt, "alice")},
		"pii int":             {ClassPII, 42, expectedPseudonym(testSalt, "42")},
		"pii valuer":          {ClassPII, valuer{slog.StringValue("alice")}, expectedPseudonym(testSalt, "alice")},
		"pii group":           {ClassPII, valuer{slog.GroupValue(slog.Int("a", 1), slog.String("b", "x"))}, expectedPseudonym(testSalt, "{a=1 b=x}")},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resetPolicy(t)
			SetPolicy(Policy{Salt: testSalt})

			got := Value(tc.class, tc.in)
			require.Equal(t, slog.KindString, got.Kind())
			assert.Equal(t, tc.want, got.String())
		})
	}

	t.Run("regular", func(t *testing.T) {
		resetPolicy(t)
		got := Value(ClassRegular, 42)
		require.Equal(t, slog.KindInt64, got.Kind())
		assert.Equal(t, int64(42), got.Int64())
	})

	t.Run("dropped", func(t *testing.T) {
		resetPolicy(t)
		SetPolicy(Policy{SecretMode: Drop})
		assert.True(t, Value(ClassSecret, "token").Equal(slog.Value{}))
	})

	t.Run("unknown class", func(t *testing.T) {
		resetPolicy(t)
		SetPolicy(Policy{Salt: testSalt})
		assert.True(t, Value(Class(7), "x").Equal(slog.Value{}))
	})
}

func TestMap(t *testing.T) {
	classOf := func(k string) Class {
		switch k {
		case "email":
			return ClassPII
		case "password":
			return ClassSecret
		default:
			return ClassRegular
		}
	}

	t.Run("sorted", func(t *testing.T) {
		resetPolicy(t)
		got := Map(map[string]int{"c": 3, "a": 1, "b": 2}, nil)
		require.Equal(t, slog.KindGroup, got.Kind())

		keys := make([]string, 0, 3)
		for _, a := range got.Group() {
			keys = append(keys, a.Key)
		}
		assert.Equal(t, []string{"a", "b", "c"}, keys)
	})

	t.Run("classes", func(t *testing.T) {
		resetPolicy(t)
		SetPolicy(Policy{Salt: testSalt})
		got := Map(map[string]string{"email": "alice@example.com", "password": "p", "name": "Alice"}, classOf)
		assert.Equal(t, map[string]any{
			"email":    expectedPseudonym(testSalt, "alice@example.com"),
			"password": SecretSet,
			"name":     "Alice",
		}, groupMap(got))
	})

	t.Run("dropped", func(t *testing.T) {
		resetPolicy(t)
		SetPolicy(Policy{SecretMode: Drop})
		got := Map(map[string]string{"email": "alice@example.com", "password": "p", "name": "Alice"}, classOf)
		assert.Equal(t, map[string]any{"name": "Alice"}, groupMap(got))
	})

	t.Run("nil classOf", func(t *testing.T) {
		resetPolicy(t)
		got := Map(map[string]string{"password": "p"}, nil)
		assert.Equal(t, map[string]any{"password": "p"}, groupMap(got))
	})

	t.Run("input unchanged", func(t *testing.T) {
		resetPolicy(t)
		SetPolicy(Policy{Salt: testSalt})
		in := map[string]string{"email": "alice@example.com", "password": "p"}
		orig := maps.Clone(in)
		Map(in, classOf)
		assert.Equal(t, orig, in)
	})

	t.Run("nil map", func(t *testing.T) {
		resetPolicy(t)
		got := Map[string](nil, classOf)
		require.Equal(t, slog.KindGroup, got.Kind())
		assert.Empty(t, got.Group())
	})
}

type stringer struct{ s string }

func (s *stringer) String() string { return s.s }

type id int

type valuer struct{ v slog.Value }

func (v valuer) LogValue() slog.Value { return v.v }

func resetPolicy(t *testing.T) {
	t.Helper()
	policy.Store(nil)
	t.Cleanup(func() { policy.Store(nil) })
}

func expectedPseudonym(salt []byte, s string) string {
	mac := hmac.New(sha256.New, salt)
	mac.Write([]byte(s))
	return PseudonymPrefix + hex.EncodeToString(mac.Sum(nil)[:pseudonymBytes])
}

func groupMap(v slog.Value) map[string]any {
	out := make(map[string]any)
	for _, a := range v.Group() {
		out[a.Key] = a.Value.Any()
	}
	return out
}
