// Package redact renders sensitive values for telemetry. It holds the
// sensitivity classes, the process-wide [Policy] and the renderers shared by
// the logging facade, telemetry processors and domain types implementing
// [slog.LogValuer].
//
// Rendering is closed by default: until [SetPolicy] is called, [ClassPII]
// values are dropped and [ClassSecret] values show only their presence.
package redact

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

// Class is the sensitivity class of a value.
type Class uint8

const (
	// ClassRegular marks a value that is recorded as is.
	ClassRegular Class = iota
	// ClassPII marks personal data: it is replaced by a keyed pseudonym that
	// keeps records of the same value correlated.
	ClassPII
	// ClassSecret marks a value that never appears; only its presence does.
	ClassSecret
)

// String returns the lower-case class name.
func (c Class) String() string {
	switch c {
	case ClassRegular:
		return "regular"
	case ClassPII:
		return "pii"
	case ClassSecret:
		return "secret"
	default:
		return fmt.Sprintf("class(%d)", uint8(c))
	}
}

// Mode selects how [ClassSecret] values are rendered.
type Mode uint8

const (
	// Presence records SecretSet or SecretEmpty in place of the value.
	Presence Mode = iota
	// Drop removes the value.
	Drop
)

const (
	PseudonymPrefix = "pii:"

	SecretSet   = "set"
	SecretEmpty = "empty"

	pseudonymBytes = 16
)

// Policy configures rendering of sensitive values.
type Policy struct {
	// Salt is the secret HMAC key of the deployment. Without it, ClassPII
	// values are dropped. It never appears in telemetry.
	Salt []byte
	// SecretMode selects how ClassSecret values are rendered.
	SecretMode Mode
}

var (
	policy   atomic.Pointer[Policy]
	saltWarn sync.Once
)

func SetPolicy(p Policy) {
	p.Salt = bytes.Clone(p.Salt)
	policy.Store(&p)
}

// Render returns the text recorded for a value of class c. s is the text
// form of the value, empty reports whether the value is empty (zero number,
// empty string, list or map; a boolean is never empty).
// keep is false when the value must be dropped. Unknown classes are dropped.
func Render(c Class, s string, empty bool) (out string, keep bool) {
	switch c {
	case ClassRegular:
		return s, true
	case ClassPII:
		p := policy.Load()
		if p == nil {
			return "", false
		}
		if len(p.Salt) == 0 {
			saltWarn.Do(func() {
				fmt.Fprintln(os.Stderr, "redact: no HMAC salt configured, PII values are dropped")
			})
			return "", false
		}
		return pseudonym(p.Salt, s), true
	case ClassSecret:
		if p := policy.Load(); p != nil && p.SecretMode == Drop {
			return "", false
		}
		if empty {
			return SecretEmpty, true
		}
		return SecretSet, true
	default:
		return "", false
	}
}

// Value returns v rendered for class c. A dropped value is the zero
// [slog.Value]. ClassRegular returns v unchanged.
func Value(c Class, v any) slog.Value {
	out, _ := value(c, v)
	return out
}

// Map returns m as an [slog.KindGroup] value sorted by key, with each value
// rendered for the class classOf gives its key. Dropped values are left out
// and m is not modified. A nil classOf records every value as is.
func Map[V any](m map[string]V, classOf func(string) Class) slog.Value {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	attrs := make([]slog.Attr, 0, len(keys))
	for _, k := range keys {
		c := ClassRegular
		if classOf != nil {
			c = classOf(k)
		}
		if v, keep := value(c, m[k]); keep {
			attrs = append(attrs, slog.Attr{Key: k, Value: v})
		}
	}
	return slog.GroupValue(attrs...)
}

func pseudonym(salt []byte, s string) string {
	mac := hmac.New(sha256.New, salt)
	mac.Write([]byte(s))
	return PseudonymPrefix + hex.EncodeToString(mac.Sum(nil)[:pseudonymBytes])
}

func value(c Class, v any) (slog.Value, bool) {
	if c == ClassRegular {
		return slog.AnyValue(v), true
	}
	s, empty := text(v)
	out, keep := Render(c, s, empty)
	if !keep {
		return slog.Value{}, false
	}
	return slog.StringValue(out), true
}

func text(v any) (string, bool) {
	sv := slog.AnyValue(v).Resolve()
	switch sv.Kind() {
	case slog.KindString:
		s := sv.String()
		return s, s == ""
	case slog.KindInt64:
		return sv.String(), sv.Int64() == 0
	case slog.KindUint64:
		return sv.String(), sv.Uint64() == 0
	case slog.KindFloat64:
		return sv.String(), sv.Float64() == 0
	case slog.KindDuration:
		return sv.String(), sv.Duration() == 0
	case slog.KindTime:
		return sv.String(), sv.Time().IsZero()
	case slog.KindGroup:
		return toString(sv), len(sv.Group()) == 0
	case slog.KindAny:
		// continue to string type assertion
	default:
		return sv.String(), false
	}

	switch x := sv.Any().(type) {
	case nil:
		return "", true
	case []byte:
		return string(x), len(x) == 0
	case fmt.Stringer:
		if isNil(reflect.ValueOf(x)) {
			return "", true
		}
		s := x.String()
		return s, s == ""
	}

	rv := reflect.ValueOf(sv.Any())
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return "", true
		}
		return text(rv.Elem().Interface())
	case reflect.Slice, reflect.Map, reflect.Array, reflect.String:
		return fmt.Sprint(sv.Any()), rv.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprint(sv.Any()), rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return fmt.Sprint(sv.Any()), rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return fmt.Sprint(sv.Any()), rv.Float() == 0
	default:
		return fmt.Sprint(sv.Any()), false
	}
}

// toString formats [slog.Value] as string. Groups are written as {key=value ...}
func toString(v slog.Value) string {
	var b strings.Builder
	writeValue(&b, v)
	return b.String()
}

func writeValue(b *strings.Builder, v slog.Value) {
	v = v.Resolve()
	if v.Kind() != slog.KindGroup {
		b.WriteString(v.String())
		return
	}

	b.WriteByte('{')
	for i, a := range v.Group() {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(a.Key)
		b.WriteByte('=')
		writeValue(b, a.Value)
	}
	b.WriteByte('}')
}

func isNil(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}
