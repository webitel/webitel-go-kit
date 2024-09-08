package log

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Severity represents a log record severity (also known as log level).
// Smaller numerical values correspond to less severe log records (such as debug events),
// larger numerical values correspond to more severe log records (such as errors and critical events).
// Wrap over https://pkg.go.dev/go.opentelemetry.io/otel/log#Severity
type Severity int

const (
	// NONE represents an unset Severity.
	NONE Severity = iota // UNDEFINED

	// A fine-grained debugging log record. Typically disabled in default
	// configurations.
	TRACE
	TRACE2
	TRACE3
	TRACE4

	// A debugging log record.
	DEBUG
	DEBUG2
	DEBUG3
	DEBUG4

	// An informational log record. Indicates that an event happened.
	INFO
	INFO2
	INFO3
	INFO4

	// A warning log record. Not an error but is likely more important than an
	// informational event.
	WARN
	WARN2
	WARN3
	WARN4

	// An error log record. Something went wrong.
	ERROR
	ERROR2
	ERROR3
	ERROR4

	// A fatal log record such as application or system crash.
	FATAL
	FATAL2
	FATAL3
	FATAL4

	// Convenience definitions for the base severity of each level.
	// SeverityTrace = SeverityTrace1
	// SeverityDebug = SeverityDebug1
	// SeverityInfo  = SeverityInfo1
	// SeverityWarn  = SeverityWarn1
	// SeverityError = SeverityError1
	// SeverityFatal = SeverityFatal1
)

// String returns a name for the level.
// If the level has a name, then that name
// in uppercase is returned.
// If the level is between named values, then
// an integer is appended to the uppercased name.
// Examples:
//
//	LevelWarn.String() => "WARN"
//	(LevelInfo+2).String() => "INFO+2"
func (v Severity) String() string {
	str := func(base string, val Severity) string {
		if val == 0 {
			return base
		}
		return fmt.Sprintf("%s%+d", base, val)
	}

	switch {
	// case v <= NONE:
	case v < DEBUG:
		return str("TRACE", v-TRACE)
	case v < INFO:
		return str("DEBUG", v-DEBUG)
	case v < WARN:
		return str("INFO", v-INFO)
	case v < ERROR:
		return str("WARN", v-WARN)
	case v < FATAL:
		return str("ERROR", v-ERROR)
	default:
		return str("FATAL", v-FATAL)
	}
}

// MarshalJSON implements [encoding/json.Marshaler]
// by quoting the output of [Level.String].
func (v Severity) MarshalJSON() ([]byte, error) {
	// AppendQuote is sufficient for JSON-encoding all Level strings.
	// They don't contain any runes that would produce invalid JSON
	// when escaped.
	return strconv.AppendQuote(nil, v.String()), nil
}

// UnmarshalJSON implements [encoding/json.Unmarshaler]
// It accepts any string produced by [Level.MarshalJSON],
// ignoring case.
// It also accepts numeric offsets that would result in a different string on
// output. For example, "Error-8" would marshal as "INFO".
func (v *Severity) UnmarshalJSON(data []byte) error {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	return v.parse(s)
}

// MarshalText implements [encoding.TextMarshaler]
// by calling [Level.String].
func (v Severity) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

// UnmarshalText implements [encoding.TextUnmarshaler].
// It accepts any string produced by [Level.MarshalText],
// ignoring case.
// It also accepts numeric offsets that would result in a different string on
// output. For example, "Error-8" would marshal as "INFO".
func (v *Severity) UnmarshalText(data []byte) error {
	return v.parse(string(data))
}

func (v *Severity) parse(s string) (err error) {
	defer func() {
		if err != nil {
			// err = fmt.Errorf("otel/log: level(%s); %w", s, err)
			err = fmt.Errorf("otel/log: severity string %q: %w", s, err)
		}
	}()

	name := s
	depth := 0
	if i := strings.IndexAny(s, "+-"); i >= 0 {
		name = s[:i]
		depth, err = strconv.Atoi(s[i:])
		if err != nil {
			return err
		}
	}
	switch strings.ToUpper(name) {
	case "TRACE":
		*v = TRACE
	case "DEBUG":
		*v = DEBUG
	case "INFO":
		*v = INFO
	case "WARN":
		*v = WARN
	case "ERROR":
		*v = ERROR
	case "FATAL":
		*v = FATAL
	default:
		// return errors.New("!SEVERITY")
		return errors.New("unknown name")
	}
	*v += Severity(depth)
	return nil
}
