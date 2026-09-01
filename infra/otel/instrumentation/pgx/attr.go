package pgx

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

const (
	_maxStringValueLength = 256
	_shortenedPattern     = "... (more than 256 chars)"

	sqlOperationUnknown = "UNKNOWN"
)

// SQLOperationName attempts to get the first 'word' from a given SQL query, which usually
// is the operation name (e.g. 'SELECT').
func SQLOperationName(query string) string {
	parts := strings.Fields(query)
	if len(parts) == 0 {
		// Fall back to a fixed value to prevent creating lots of tracing operations
		// differing only by the amount of whitespace in them (in case we'd fall back
		// to the full query or a cut-off version).
		return sqlOperationUnknown
	}

	return strings.ToUpper(parts[0])
}

// FromNamedValue converts driver.NamedValue to attribute.KeyValue.
func FromNamedValue(arg driver.NamedValue) attribute.KeyValue {
	return KeyValue(KeyFromNamedValue(arg), arg.Value)
}

// KeyFromNamedValue returns an attribute.Key from a given driver.NamedValue.
func KeyFromNamedValue(arg driver.NamedValue) attribute.Key {
	name := arg.Name
	if name == "" {
		// db.operation.parameter uses a 0-based index for unnamed parameters,
		// while driver.NamedValue.Ordinal starts at one.
		name = strconv.Itoa(arg.Ordinal - 1)
	}

	// db.operation.parameter is a template attribute, so upstream exports a
	// constructor rather than a Key. The empty value is discarded — KeyValue
	// below types the real one.
	return semconv.DBOperationParameter(name, "").Key
}

// KeyValue returns an attribute.KeyValue from a given value.
// nolint: cyclop
func KeyValue(key attribute.Key, val interface{}) attribute.KeyValue {
	switch v := val.(type) {
	case nil:
		return key.String("")

	case int:
		return key.Int(v)

	case int64:
		return key.Int64(v)

	case float64:
		return key.Float64(v)

	case bool:
		return key.Bool(v)

	case []byte:
		return key.String(shortenString(string(v)))

	case string:
		return key.String(shortenString(v))

	case []int:
		return key.IntSlice(v)

	case []int64:
		return key.Int64Slice(v)

	case []float64:
		return key.Float64Slice(v)

	case []bool:
		return key.BoolSlice(v)

	case *int, *int64, *float64, *bool, *string:
		val := reflect.ValueOf(v)
		if val.IsNil() {
			return key.String("")
		}

		return KeyValue(key, val.Elem().Interface())

	case time.Duration:
		return KeyValueDuration(key, v)

	default:
		return key.String(shortenString(fmt.Sprintf("%v", v)))
	}
}

// KeyValueDuration converts time.Duration to attribute.KeyValue.
func KeyValueDuration(key attribute.Key, d time.Duration) attribute.KeyValue {
	if time.Microsecond <= d && d < time.Millisecond {
		var sb strings.Builder

		sb.WriteString(strconv.FormatInt(d.Microseconds(), 10))
		sb.WriteString("us")

		return key.String(sb.String())
	}

	return key.String(d.String())
}

func shortenString(s string) string {
	runes := []rune(s)
	if len(runes) <= _maxStringValueLength {
		return s
	}

	end := _maxStringValueLength - len(_shortenedPattern)
	sb := strings.Builder{}
	sb.Grow(_maxStringValueLength)
	sb.WriteString(string(runes[:end]))
	sb.WriteString(_shortenedPattern)

	return sb.String()
}
